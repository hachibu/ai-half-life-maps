"""Deterministic gates. Each gate is a pure function: (facts) -> Result(ok | pending | fail, reasons).

    python -m tools.pipeline.gates pr 12 [--apply]     evaluate a pull request (apply: status, comment, merge, ledger)
    python -m tools.pipeline.gates issue 7 [--apply]   evaluate a task issue (apply: ready/blocked labels, claim check)

`facts` are gathered from the GitHub REST API by collect_pr()/collect_issue(); the gate functions themselves
never touch the network, so tools/pipeline/test_gates.py can test every rule offline.
"""
import json
import re
import sys
from dataclasses import dataclass, field
from pathlib import Path

from tools.pipeline import model as M

OK, PENDING, FAIL = "ok", "pending", "fail"
PR_SECTIONS = ("summary", "what a player will notice", "screenshots", "changes", "check results", "not verified / risks")
STALE_CLAIM_HOURS = 48
WIP_LIMIT = 2


@dataclass
class Result:
    gate: str
    state: str = OK
    reasons: list = field(default_factory=list)

    def fail(self, why):
        self.state = FAIL; self.reasons.append(why); return self

    def wait(self, why):
        if self.state != FAIL:
            self.state = PENDING
        self.reasons.append(why); return self


# ================================================================== task gates
def gate_spec(task, closed_numbers, existing_maps):
    """idea -> ready: the issue is a complete, unblocked task."""
    r = Result("spec")
    if not M.filled(task["goal"]): r.fail("Goal is empty")
    if not M.filled(task["criteria"]): r.fail("Acceptance criteria are empty")
    if not task["paths"]: r.fail("Paths is empty (list the folders/files this task may change)")
    for p in task["paths"]:
        if not p.startswith(M.ALLOWED_ROOTS): r.fail("path %r is outside the repo areas %s" % (p, ", ".join(M.ALLOWED_ROOTS)))
    mp = task["map"]
    if mp.lower() == "none":
        pass                                           # tools/docs task: no map (M.filled() treats "none" as empty)
    elif not M.filled(mp): r.fail("Map is empty (a map name, new:<name>, or none)")
    elif mp != "none" and not mp.startswith("new:") and mp not in existing_maps:
        r.fail("map %r does not exist (use new:<name> for a new map)" % mp)
    if r.state == OK:
        open_deps = [d for d in task["deps"] if d not in closed_numbers]
        if open_deps: r.wait("blocked by open dependencies %s" % ", ".join("#%d" % d for d in open_deps))
        if task["hold"]: r.wait("on hold (label `hold`)")
    return r


def gate_claim(task, other_claims, claim_counts):
    """ready -> claimed: one agent, no overlapping claimed paths (tasks a human assigns aren't WIP-limited)."""
    r = Result("claim")
    if len(task["claimed"]) > 1: r.fail("claimed by more than one agent: %s" % ", ".join(task["claimed"]))
    if not task["claimed"]:
        return r.wait("not claimed yet")
    a = task["claimed"][0]
    for o in other_claims:      # earlier claims on overlapping paths win
        hits = [(p, q) for p in task["paths"] for q in o["paths"] if M.path_overlap(p, q)]
        if hits:
            r.fail("paths overlap #%d (claimed earlier by %s): %s vs %s" % (o["number"], o["agent"], hits[0][0], hits[0][1]))
    return r


# ================================================================== PR gates
def gate_scope(pr, agent, files, tasks_by_number, labels):
    """claimed -> pr: the PR closes a task its agent claimed and stays inside that task's paths."""
    r = Result("scope")
    names = [f["filename"] for f in files]
    if any(n.startswith("build/") for n in names): r.fail("build/ output is committed")
    for f in files:
        if f.get("status") != "removed" and f.get("additions", 0) == 0 and f.get("changes", 0) == 0 and \
                Path(f["filename"]).suffix in (".bsp", ".wad", ".zip", ".mp4", ".png", ".tga"):
            r.fail("binary build artifact %s is committed" % f["filename"])
    protected = [n for n in names if n.startswith(M.PROTECTED)]
    if protected and "human-approved" not in labels:
        r.wait("changes the rules of the system (%s): needs the `human-approved` label from a human" % ", ".join(protected[:3]))
    if agent == "human":
        return r
    nums = M.closes(pr.get("body"))
    if not nums:
        return r.fail("description must link its task: `Closes #<issue>`")
    allowed = ["docs/lessons/"]
    for n in nums:
        t = tasks_by_number.get(n)
        if t is None: r.fail("#%d is not an open task issue" % n); continue
        if agent not in t["claimed"]: r.fail("#%d is not claimed by %s (label `claimed:%s`)" % (n, agent, agent))
        allowed += t["paths"]
    stray = [n for n in names if not M.covered(n, allowed)]
    if stray: r.fail("files outside the task's paths: %s" % ", ".join(stray[:5]))
    if any(n.startswith("tools/") for n in names) and any(n.startswith("maps/") for n in names):
        if not (any(p.startswith("tools") for p in allowed) and any(p.startswith("maps") for p in allowed)):
            r.fail("tools/ and maps/ changed together without a task covering both")
    return r


def gate_description(pr, files):
    """The PR template is filled in, so a human can follow the change."""
    r = Result("description")
    s = M.sections(pr.get("body") or "", "##")
    maps_changed = M.changed_maps(files)
    for h in PR_SECTIONS:
        txt = s.get(h)
        if txt is None: r.fail("section '## %s' is missing" % h.title()); continue
        clean = re.sub(r"(?m)^\s*[-*]\s*(`view_name`:)?\s*$", "", txt)
        clean = clean.replace("paste here", "").replace("```", "").replace("|  |  |", "")
        clean = re.sub(r"\|\s*File or zone\s*\|\s*What changed and why\s*\|\s*\n?\|---\|---\|", "", clean)
        if not M.filled(clean.strip()):
            if h == "screenshots" and not maps_changed:
                continue
            r.fail("section '## %s' is empty" % h.title())
    if maps_changed and "screenshots" in s and not re.search(r"`[\w-]+`", s["screenshots"]):
        r.fail("'## Screenshots' must name at least one view (e.g. `pond_from_bridge`: what it shows)")
    return r


def gate_checks(runs, jobs_by_run, head_sha):
    """pr -> built -> shown: map-check (build, tests, screenshots + visual gate) passed on the head commit."""
    r = Result("checks")
    mine = [x for x in runs if x.get("head_sha") == head_sha]
    if not mine:
        return r.wait("map-check has not started for %s" % head_sha[:7])
    run = max(mine, key=lambda x: x.get("run_number", 0))
    if run.get("status") != "completed":
        return r.wait("map-check is %s" % run.get("status"))
    if run.get("conclusion") not in ("success",):
        bad = [j["name"] for j in jobs_by_run.get(run["id"], []) if j.get("conclusion") not in ("success", "skipped", None)]
        return r.fail("map-check %s: %s" % (run.get("conclusion"), ", ".join(bad) or "see the run"))
    return r


def gate_tests_ratchet(files, labels):
    """Checks may grow but not shrink: a PR that deletes more than it adds in a map's tests needs a human."""
    r = Result("ratchet")
    for f in files:
        n = f["filename"]
        if re.match(r"^maps/[^/]+/(tests/|zones/[^/]+\.py$)", n) or n.startswith("tools/checks/"):
            if n.startswith("tools/checks/") or "/tests/" in n:
                if f.get("status") == "removed" or f.get("deletions", 0) > f.get("additions", 0):
                    if "human-approved" not in labels:
                        r.fail("%s shrinks (-%d/+%d): removing checks needs `human-approved`"
                               % (n, f.get("deletions", 0), f.get("additions", 0)))
    return r


def gate_review(agent, head_sha, comments, reviews, review_comments, head_date, labels):
    """built -> reviewed: nobody else asked for changes on this exact commit.

    Approval is optional: agents run only when a human asks, so there may be no second agent to review, and an
    agent may merge its own PR once every other gate is green. A review still counts when one happens: a
    `verdict: changes` from anyone but the author, or Codex's automatic review leaving inline findings on the
    head commit, blocks the merge until a new commit is pushed. A human can pause anything with `hold`."""
    r = Result("review")
    if "hold" in labels:
        return r.wait("on hold (label `hold`)")
    if "human-approved" in labels:
        return r                                       # a human may always approve
    objections = []
    for c in list(comments) + list(reviews):
        rv = M.parse_agent_review(c.get("body"))
        if not rv or len(rv["sha"]) < 7 or not head_sha.lower().startswith(rv["sha"][:7]):
            continue                                   # not a review block, or a review of an older commit
        if rv["reviewer"] == agent:
            continue                                   # an author can't block (or approve) their own PR
        if rv["verdict"] != "approve":
            objections.append(rv["reviewer"])
    if agent != "codex":                               # Codex's automatic GitHub review: inline findings block
        flagged = [x for x in review_comments if M.is_codex_login(x.get("user", {}).get("login"))
                   and x.get("commit_id") == head_sha]
        if flagged:
            objections.append("codex (%d inline finding%s)" % (len(flagged), "s" if len(flagged) > 1 else ""))
    if objections:
        return r.fail("changes requested on %s by %s" % (head_sha[:7], ", ".join(objections)))
    return r


def gate_learning(files, branch_runs, closes_incident, lesson_texts):
    """reviewed -> learned: a PR that hit a red check (or fixes an incident) must add a lesson."""
    r = Result("learning")
    lessons = [f for f in files if M.is_lesson_path(f["filename"]) and f.get("status") in ("added", "modified")]
    for f in lessons:
        probs = M.lesson_problems(lesson_texts.get(f["filename"], ""))
        if probs: r.fail("%s: %s" % (f["filename"], "; ".join(probs)))
    failed = [x for x in branch_runs if x.get("conclusion") == "failure"]
    if (failed or closes_incident) and not lessons:
        why = "fixes an incident" if closes_incident else "map-check failed %d time(s) on this branch" % len(failed)
        r.fail("%s: add a lesson in docs/lessons/ (what happened, why, rule)" % why)
    return r


def overall(results):
    if any(x.state == FAIL for x in results): return FAIL
    if any(x.state == PENDING for x in results): return PENDING
    return OK


# ================================================================== collect facts + apply
def collect_pr(n):
    from tools.pipeline import gh
    pr = gh.api("repos/{repo}/pulls/%d" % n)
    files = gh.pr_files(n)
    head = pr["head"]["sha"]
    tasks = {i["number"]: M.parse_task(i) for i in gh.issues("open", "task")}
    runs = gh.runs_for_sha(head)
    jobs = {x["id"]: gh.run_jobs(x["id"]) for x in runs if x.get("head_sha") == head and x.get("conclusion") not in ("success", None)}
    lesson_texts = {}
    for f in files:
        if M.is_lesson_path(f["filename"]) and f.get("status") != "removed":
            c = gh.api("repos/{repo}/contents/%s?ref=%s" % (f["filename"], head))
            import base64
            lesson_texts[f["filename"]] = base64.b64decode(c["content"]).decode("utf-8", "replace")
    incidents = {i["number"] for i in gh.issues("all", "incident")}
    return dict(pr=pr, files=files, head=head, agent=M.pr_agent(pr), labels=M.labels(pr), tasks=tasks,
                runs=runs, jobs=jobs, branch_runs=gh.runs_for_branch(pr["head"]["ref"]),
                comments=gh.comments(n), reviews=gh.reviews(n), review_comments=gh.review_comments(n),
                head_date=gh.commit(head)["commit"]["committer"]["date"], lesson_texts=lesson_texts,
                closes_incident=bool(set(M.closes(pr.get("body"))) & incidents))


def evaluate_pr(f):
    return [
        gate_scope(f["pr"], f["agent"], f["files"], f["tasks"], f["labels"]),
        gate_description(f["pr"], f["files"]),
        gate_checks(f["runs"], f["jobs"], f["head"]),
        gate_tests_ratchet(f["files"], f["labels"]),
        gate_review(f["agent"], f["head"], f["comments"], f["reviews"], f["review_comments"], f["head_date"], f["labels"]),
        gate_learning(f["files"], f["branch_runs"], f["closes_incident"], f["lesson_texts"]),
    ]


STATUS_MARK = "<!-- pipeline-status"
ICON = {OK: "✅", PENDING: "⏳", FAIL: "❌"}


def status_comment(n, head, agent, results):
    state = overall(results)
    data = dict(pr=n, sha=head, agent=agent, state=state, gates={x.gate: dict(state=x.state, reasons=x.reasons) for x in results})
    lines = ["%s %s -->" % (STATUS_MARK, json.dumps(data)), "## Pipeline: %s %s" % (ICON[state], state.upper()), "",
             "Commit `%s` · author agent: **%s**" % (head[:7], agent), "", "| gate | state | details |", "|---|---|---|"]
    for x in results:
        lines.append("| %s | %s | %s |" % (x.gate, ICON[x.state], "<br>".join(x.reasons).replace("|", "\\|") or ""))
    lines += ["", "_Deterministic gates from `tools/pipeline/gates.py`, evaluated by CI. "
              "Fix the ❌ rows and push; ⏳ rows clear on their own. See docs/pipeline.md._"]
    return "\n".join(lines), data


def apply_pr(n, results, f, dry=False):
    from tools.pipeline import gh, ledger
    body, data = status_comment(n, f["head"], f["agent"], results)
    state = data["state"]
    print(body)
    if dry:
        return state
    gh.api("repos/{repo}/statuses/%s" % f["head"], "POST", {
        "state": {"ok": "success", "pending": "pending", "fail": "failure"}[state], "context": "pipeline/gate",
        "description": ("all gates passed" if state == OK else
                        "; ".join("%s: %s" % (x.gate, x.state) for x in results if x.state != OK))[:139]})
    old = [c for c in f["comments"] if (c.get("body") or "").startswith(STATUS_MARK)]
    if old:
        if old[0]["body"] != body:
            gh.api("repos/{repo}/issues/comments/%d" % old[0]["id"], "PATCH", {"body": body})
    else:
        gh.api("repos/{repo}/issues/%d/comments" % n, "POST", {"body": body})
    ledger.record_gates("pr#%d" % n, f["agent"], f["head"], results)
    if state == OK and not f["pr"].get("draft") and f["pr"]["state"] == "open":
        title = f["pr"]["title"]
        gh.api("repos/{repo}/pulls/%d/merge" % n, "PUT", {"merge_method": "squash", "sha": f["head"],
                                                           "commit_title": "%s (#%d)" % (title, n)})
        ledger.record("merge", "pr#%d" % n, f["agent"], True, f["head"], {"title": title})
        try:
            gh.api("repos/{repo}/git/refs/heads/%s" % f["pr"]["head"]["ref"], "DELETE")
        except RuntimeError:
            pass
        gh.api("repos/{repo}/actions/workflows/release.yml/dispatches", "POST", {"ref": "main"})
        print("merged #%d and dispatched release" % n)
    return state


def collect_issue(n):
    from tools.pipeline import gh
    issue = gh.api("repos/{repo}/issues/%d" % n)
    task = M.parse_task(issue)
    all_issues = gh.issues("all")
    closed = {i["number"] for i in all_issues if i["state"] == "closed"}
    open_tasks = [M.parse_task(i) for i in all_issues if i["state"] == "open" and "task" in M.labels(i)]
    maps = {p.name for p in Path("maps").iterdir() if p.is_dir()} if Path("maps").is_dir() else set()

    def claim_time(t):
        ev = [e for e in gh.label_events(t["number"]) if e["event"] == "labeled" and e["label"]["name"].startswith("claimed:")]
        return ev[-1]["created_at"] if ev else t["created"]
    mine = claim_time(task) if task["claimed"] else None
    others = []
    for t in open_tasks:
        if t["number"] != n and t["claimed"] and mine and claim_time(t) < mine:
            others.append(dict(number=t["number"], agent=t["claimed"][0], paths=t["paths"]))
    counts = {}
    for t in open_tasks:
        for a in t["claimed"]:
            counts[a] = counts.get(a, 0) + 1
    return dict(issue=issue, task=task, closed=closed, maps=maps, others=others, counts=counts)


def apply_issue(n, f, dry=False):
    from tools.pipeline import gh, ledger
    t = f["task"]
    spec = gate_spec(t, f["closed"], f["maps"])
    res = [spec] + ([gate_claim(t, f["others"], f["counts"])] if t["claimed"] else [])
    for x in res:
        print("%-6s %-8s %s" % (x.gate, x.state, "; ".join(x.reasons)))
    if dry or t["state"] != "open":
        return res
    want = {"ready": spec.state == OK, "blocked": spec.state == PENDING, "needs-spec": spec.state == FAIL}
    for lab, on in want.items():
        has = lab in t["labels"]
        if on and not has:
            gh.api("repos/{repo}/issues/%d/labels" % n, "POST", {"labels": [lab]})
        elif has and not on:
            gh.api("repos/{repo}/issues/%d/labels/%s" % (n, lab), "DELETE")
    if spec.state == FAIL and "needs-spec" not in t["labels"]:
        gh.api("repos/{repo}/issues/%d/comments" % n, "POST", {"body": "Spec gate ❌\n\n" + "\n".join("- " + x for x in spec.reasons)})
    if len(res) > 1 and res[1].state == FAIL:
        a = t["claimed"][-1]
        gh.api("repos/{repo}/issues/%d/labels/claimed:%s" % (n, a), "DELETE")
        gh.api("repos/{repo}/issues/%d/comments" % n, "POST",
               {"body": "Claim gate ❌ — removed `claimed:%s`:\n\n%s" % (a, "\n".join("- " + x for x in res[1].reasons))})
    ledger.record_gates("issue#%d" % n, (t["claimed"] or ["-"])[0], "", res)
    return res


def main(argv):
    kind, n = argv[0], int(argv[1])
    apply = "--apply" in argv
    if kind == "pr":
        f = collect_pr(n)
        state = apply_pr(n, evaluate_pr(f), f, dry=not apply)
        return 0 if state != FAIL else 1
    f = collect_issue(n)
    apply_issue(n, f, dry=not apply)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
