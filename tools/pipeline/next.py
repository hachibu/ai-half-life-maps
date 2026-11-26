"""What should this agent do next?  Deterministic: same GitHub state -> same answer.

    python -m tools.pipeline.next claude          # human-readable
    python -m tools.pipeline.next codex --json    # for workflows

Priority (first match wins):
  1. fix-incident  an open `incident` issue (main is broken) nobody has claimed, or that you claimed
  2. fix-pr        one of your open PRs has a failing (❌) gate
  3. review        the other agent's PR waits only on your review (checks already green)
  4. continue      a task you claimed has no PR yet
  5. claim         the best ready task: unclaimed, unblocked, no path overlap with claims, within WIP limit
  6. propose       fewer than BACKLOG_MIN ready tasks: write new task issues
  7. idle
Every action comes with the lessons (docs/lessons/) whose tags match the work.
"""
import json
import re
import sys
from pathlib import Path

from tools.pipeline import OTHER
from tools.pipeline import model as M
from tools.pipeline.gates import WIP_LIMIT, STATUS_MARK

BACKLOG_MIN = 3


def pr_status(comments):
    for c in comments:
        b = c.get("body") or ""
        if b.startswith(STATUS_MARK):
            return json.loads(b[len(STATUS_MARK):b.index("-->")].strip())
    return None


def lessons_for(words, root="docs/lessons"):
    words = {w.lower() for w in words if w}
    out = []
    for p in sorted(Path(root).glob("*.md")):
        if p.name == "README.md":
            continue
        L = M.parse_lesson(p.read_text())
        if L and ({t.lower() for t in L["tags"]} & (words | {"general"})):
            out.append(str(p))
    return out


def keywords(task=None, paths=(), gates=()):
    w = set(gates)
    for p in list(paths) + (task["paths"] if task else []):
        w |= set(re.split(r"[/._-]+", p))
    if task:
        w.add(task["map"])
    return w - {"", "maps", "md", "py"}


def decide(agent, issues, prs, pr_comments, pr_files):
    tasks = [M.parse_task(i) for i in issues if "task" in M.labels(i) or "incident" in M.labels(i)]
    open_numbers = {i["number"] for i in issues}
    linked = {n: p for p in prs for n in M.closes(p.get("body"))}

    # 1. incidents
    for t in sorted(tasks, key=lambda t: t["number"]):
        if t["incident"] and (not t["claimed"] or agent in t["claimed"]) and t["number"] not in linked:
            return dict(action="fix-incident", issue=t["number"], title=t["title"], claim=not t["claimed"],
                        why="main is broken; incidents come first",
                        lessons=lessons_for(keywords(t, gates=["health", "build"])))
    # 2. my failing PRs
    mine = [p for p in prs if M.pr_agent(p) == agent and not p.get("draft")]
    for p in sorted(mine, key=lambda p: p["number"]):
        st = pr_status(pr_comments.get(p["number"], []))
        if st and st.get("sha") == p["head"]["sha"]:
            bad = {g: v["reasons"] for g, v in st["gates"].items() if v["state"] == "fail"}
            if bad:
                return dict(action="fix-pr", pr=p["number"], branch=p["head"]["ref"], failing=bad,
                            why="your PR has failing gates",
                            lessons=lessons_for(keywords(paths=[f["filename"] for f in pr_files.get(p["number"], [])], gates=bad)))
    # 3. reviews owed
    for p in sorted(prs, key=lambda p: p["number"]):
        a = M.pr_agent(p)
        if a == agent or (a != "human" and OTHER.get(a) != agent) or p.get("draft"):
            continue
        st = pr_status(pr_comments.get(p["number"], []))
        if not st or st.get("sha") != p["head"]["sha"]:
            continue
        g = st["gates"]
        others_ok = all(v["state"] == "ok" for k, v in g.items() if k not in ("review", "learning"))
        if others_ok and g.get("review", {}).get("state") == "pending":
            return dict(action="review", pr=p["number"], sha=p["head"]["sha"], author=a,
                        why="checks are green and it waits on your review",
                        lessons=lessons_for(keywords(paths=[f["filename"] for f in pr_files.get(p["number"], [])], gates=["review"])))
    # 4. my claimed tasks without a PR
    my_claims = [t for t in tasks if agent in t["claimed"] and not t["incident"]]
    for t in sorted(my_claims, key=lambda t: (M.PRIORITY[t["priority"]], t["number"])):
        if t["number"] not in linked:
            return dict(action="continue", issue=t["number"], title=t["title"], paths=t["paths"],
                        why="you claimed this and there is no PR yet", lessons=lessons_for(keywords(t)))
    # 5. claim
    claimed_paths = [p for t in tasks if t["claimed"] for p in t["paths"]]
    ready = [t for t in tasks if "ready" in t["labels"] and not t["claimed"] and not t["hold"] and not t["incident"]
             and all(d not in open_numbers for d in t["deps"])]
    free = [t for t in ready if not any(M.path_overlap(p, q) for p in t["paths"] for q in claimed_paths)]
    if len(my_claims) < WIP_LIMIT and free:
        t = sorted(free, key=lambda t: (M.PRIORITY[t["priority"]], t["number"]))[0]
        return dict(action="claim", issue=t["number"], title=t["title"], paths=t["paths"],
                    why="best ready task (priority %s, oldest first)" % t["priority"], lessons=lessons_for(keywords(t)))
    # 6. propose
    if len(ready) < BACKLOG_MIN:
        return dict(action="propose", count=BACKLOG_MIN - len(ready),
                    why="only %d ready task(s); the backlog needs at least %d" % (len(ready), BACKLOG_MIN),
                    lessons=lessons_for({"planning"}))
    return dict(action="idle", why="nothing for %s right now" % agent, lessons=[])


INSTRUCTIONS = {
    "fix-incident": "Claim the incident issue (label claimed:{agent}) if unclaimed. Reproduce with `make build/test`, fix it on a "
                    "branch {agent}/incident-<n>, add a lesson in docs/lessons/, open a PR with `Closes #<n>`.",
    "fix-pr": "Check out the PR branch, fix every failing gate listed (see the PR's Pipeline comment), push. If a check "
              "failed, add or update a lesson in docs/lessons/.",
    "review": "Review the PR at the given sha: read the diff, open the screenshot comment and LOOK at the images, read the "
              "check results. Post a comment ending with the agent-review block (see AGENTS.md). Approve only if you "
              "would be happy to see it in main; otherwise `verdict: changes` plus what to fix.",
    "continue": "Finish the claimed task on branch {agent}/<topic> and open a PR that fills the template and says Closes #<n>.",
    "claim": "Add the label claimed:{agent} to the issue, wait for the claim gate (comment appears only on conflict), "
             "then do the task on branch {agent}/<topic> and open a PR with Closes #<n>.",
    "propose": "Open {count} new task issue(s) with the Task form: concrete, small (one PR each), with paths, acceptance "
               "criteria and dependencies. Prefer improvements suggested by lessons and by the ledger's failure stats.",
    "idle": "Nothing to do. Stop.",
}


def main(argv):
    from tools.pipeline import gh
    agent = argv[0]
    issues = gh.issues("open")
    prs = gh.pulls("open")
    pr_comments = {p["number"]: gh.comments(p["number"]) for p in prs}
    pr_files = {p["number"]: gh.pr_files(p["number"]) for p in prs}
    d = decide(agent, issues, prs, pr_comments, pr_files)
    d["agent"] = agent
    d["instructions"] = INSTRUCTIONS[d["action"]].format(agent=agent, count=d.get("count", 0))
    if "--json" in argv:
        print(json.dumps(d, indent=2))
    else:
        print("NEXT for %s: %s  (%s)" % (agent, d["action"].upper(), d["why"]))
        for k in ("issue", "pr", "title", "branch", "sha", "paths", "failing", "count"):
            if k in d:
                print("  %s: %s" % (k, d[k]))
        print("  do: " + d["instructions"])
        if d["lessons"]:
            print("  read first:\n" + "\n".join("    " + x for x in d["lessons"]))


if __name__ == "__main__":
    main(sys.argv[1:])
