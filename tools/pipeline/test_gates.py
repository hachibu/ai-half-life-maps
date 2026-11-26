"""Offline tests for the gates and the scheduler (no network).   python -m tools.pipeline.test_gates"""
from tools.pipeline import gates as G
from tools.pipeline import model as M

TASK_BODY = """### Map
prairieloop

### Paths
maps/prairieloop/

### Goal
Continuous snow

### Acceptance criteria
- no sprite patches

### Depends on
None

### Priority
P1
"""

PR_BODY = """Closes #5

## Summary
Snow everywhere.

## What a player will notice
- snow is continuous

## Screenshots
- `street`: snow over the road

## Changes
| File or zone | What changed and why |
|---|---|
| build.py | sheets |

## Check results
```
routes: 24  failed: 0
```

## Not verified / risks
Not played in game.
"""


def issue(n, body=TASK_BODY, labels=("task",), state="open"):
    return dict(number=n, title="t%d" % n, body=body, labels=[{"name": l} for l in labels], state=state, created_at="2026-01-0%dT00:00:00Z" % (n % 9 + 1))


def pr(n=9, ref="claude/snow", body=PR_BODY, sha="abcdef1234567"):
    return dict(number=n, title="Snow", body=body, head=dict(ref=ref, sha=sha), state="open", draft=False)


def files(*names, adds=5, dels=0):
    return [dict(filename=x, status="modified", additions=adds, deletions=dels, changes=adds + dels) for x in names]


def check(cond, msg):
    assert cond, msg
    print("ok  ", msg)


def main():
    t = M.parse_task(issue(5, labels=("task", "claimed:claude")))
    check(t["paths"] == ["maps/prairieloop/"] and t["priority"] == "P1" and t["claimed"] == ["claude"], "task parses")
    check(G.gate_spec(t, set(), {"prairieloop"}).state == G.OK, "complete task passes spec")
    bad = M.parse_task(issue(6, body=TASK_BODY.replace("Continuous snow", "_No response_")))
    check(G.gate_spec(bad, set(), {"prairieloop"}).state == G.FAIL, "empty goal fails spec")
    nomap = M.parse_task(issue(9, body=TASK_BODY.replace("prairieloop\n\n###", "none\n\n###", 1)))
    check(nomap["map"] == "none" and G.gate_spec(nomap, set(), {"prairieloop"}).state == G.OK, "Map: none passes spec")
    empty = M.parse_task(issue(9, body=TASK_BODY.replace("prairieloop\n\n###", "_No response_\n\n###", 1)))
    check(G.gate_spec(empty, set(), {"prairieloop"}).state == G.FAIL, "empty Map fails spec")
    dep = M.parse_task(issue(7, body=TASK_BODY.replace("None", "#3")))
    check(G.gate_spec(dep, set(), {"prairieloop"}).state == G.PENDING, "open dependency blocks")
    check(G.gate_spec(dep, {3}, {"prairieloop"}).state == G.OK, "closed dependency unblocks")
    check(G.gate_spec(M.parse_task(issue(8, body=TASK_BODY.replace("prairieloop\n\n###", "nosuch\n\n###", 1))), set(), {"prairieloop"}).state == G.FAIL,
          "unknown map fails spec")

    other = [dict(number=2, agent="claude", paths=["maps/prairieloop/zones/park.py"])]
    check(G.gate_claim(t, other, {"claude": 1}).state == G.FAIL, "overlapping earlier claim fails")
    check(G.gate_claim(t, [dict(number=2, agent="claude", paths=["maps/cagematch/"])], {"claude": 1}).state == G.OK, "disjoint claim passes")
    check(G.gate_claim(t, [], {"claude": 5}).state == G.OK, "human-assigned claims are not WIP-limited")

    tasks = {5: t}
    ok = G.gate_scope(pr(), "claude", files("maps/prairieloop/build.py", "docs/lessons/2026-x.md"), tasks, set())
    check(ok.state == G.OK, "in-scope PR passes")
    check(G.gate_scope(pr(), "claude", files("maps/cagematch/build.py"), tasks, set()).state == G.FAIL, "out-of-scope file fails")
    check(G.gate_scope(pr(body="no link"), "claude", files("maps/prairieloop/a.py"), tasks, set()).state == G.FAIL, "missing Closes fails")
    check(G.gate_scope(pr(), "claude", files("maps/prairieloop/a.py"), {5: dict(t, claimed=[])}, set()).state == G.FAIL, "unclaimed task fails")
    t2 = dict(t, paths=["maps/prairieloop/", "tools/pipeline/"])
    check(G.gate_scope(pr(), "claude", files("tools/pipeline/gates.py"), {5: t2}, set()).state == G.PENDING, "protected path waits for human-approved")
    check(G.gate_scope(pr(), "claude", files("tools/pipeline/gates.py"), {5: t2}, {"human-approved"}).state == G.OK, "human-approved unlocks protected path")
    check(G.gate_scope(pr(), "claude", files("build/x.bsp"), tasks, set()).state == G.FAIL, "build output fails")

    check(G.gate_description(pr(), files("maps/prairieloop/build.py")).state == G.OK, "filled template passes")
    empty = PR_BODY.replace("Not played in game.", "")
    check(G.gate_description(pr(body=empty), files("maps/prairieloop/build.py")).state == G.FAIL, "empty section fails")
    noview = PR_BODY.replace("- `street`: snow over the road", "- looks great")
    check(G.gate_description(pr(body=noview), files("maps/prairieloop/build.py")).state == G.FAIL, "map PR must name a view")

    runs = [dict(id=1, head_sha="abcdef1234567", status="completed", conclusion="success", run_number=3)]
    check(G.gate_checks(runs, {}, "abcdef1234567").state == G.OK, "green map-check passes")
    check(G.gate_checks([dict(runs[0], status="in_progress")], {}, "abcdef1234567").state == G.PENDING, "running map-check waits")
    check(G.gate_checks([dict(runs[0], conclusion="failure")], {1: [dict(name="quality", conclusion="failure")]}, "abcdef1234567").state == G.FAIL,
          "red map-check fails with job name")
    check(G.gate_checks(runs, {}, "0000000").state == G.PENDING, "no run for head waits")

    check(G.gate_tests_ratchet(files("maps/prairieloop/tests/routes.py", adds=1, dels=9), set()).state == G.FAIL, "shrinking routes fails")
    check(G.gate_tests_ratchet(files("maps/prairieloop/tests/routes.py", adds=9, dels=1), set()).state == G.OK, "growing routes passes")

    blk = lambda who, verdict, sha="abcdef1": dict(body="looks good\n<!-- agent-review -->\nreviewer: %s\nverdict: %s\nsha: %s" % (who, verdict, sha),
                                                    user=dict(login="hachibu"), created_at="2026-01-02")
    rev = lambda *c, reviews=(), rc=(): G.gate_review("claude", "abcdef1234567", list(c), list(reviews), list(rc), "2026-01-01", set())
    check(rev().state == G.OK, "no review needed: an agent can merge its own PR")
    check(rev(blk("hachibu", "approve")).state == G.OK, "an approval passes")
    check(rev(blk("claude", "approve")).state == G.OK, "self-approval is harmless")
    check(rev(blk("hachibu", "changes")).state == G.FAIL, "changes requested by anyone but the author fails")
    check(rev(blk("claude", "changes")).state == G.OK, "the author's own `changes` block is ignored")
    check(rev(blk("hachibu", "changes", "1111111")).state == G.OK, "changes requested on an older commit are ignored")
    check(rev(blk("hachibu", "changes"), blk("hachibu", "approve")).state == G.FAIL, "an approval doesn't cancel requested changes")
    gh_rev = lambda state, sha="abcdef1234567": dict(user=dict(login="hachibu"), state=state, commit_id=sha, body="")
    check(rev(reviews=[gh_rev("CHANGES_REQUESTED")]).state == G.FAIL, "GitHub 'Request changes' on the head commit fails")
    check(rev(reviews=[gh_rev("CHANGES_REQUESTED", "1111111000000")]).state == G.OK, "GitHub 'Request changes' on an older commit is ignored")
    check(rev(reviews=[gh_rev("APPROVED")]).state == G.OK, "GitHub approval passes")
    check(G.gate_review("claude", "abcdef1234567", [blk("hachibu", "approve")], [], [], "2026-01-01", {"hold"}).state == G.PENDING, "hold label pauses")
    check(G.gate_review("claude", "abcdef1234567", [], [], [], "2026-01-01", {"hold"}).state == G.PENDING, "hold pauses even with no review")
    check(G.gate_review("claude", "abcdef1234567", [blk("hachibu", "changes")], [], [], "2026-01-01", {"human-approved"}).state == G.OK,
          "human-approved overrides requested changes")
    check(G.gate_review("claude", "abcdef1234567", [], [], [], "2026-01-01", {"human-approved"}).state == G.OK, "human-approved counts as review")

    lesson = "---\ntitle: X\ntags: [prairieloop, snow]\ngate: checks\nsource: PR #9\n---\n## What happened\na\n## Why\nb\n## Rule\nc\n"
    fl = files("maps/prairieloop/build.py") + [dict(filename="docs/lessons/2026-11-26-x.md", status="added", additions=9, deletions=0, changes=9)]
    failed_runs = [dict(conclusion="failure")]
    check(G.gate_learning(files("maps/prairieloop/build.py"), failed_runs, False, {}).state == G.FAIL, "red run without lesson fails learning")
    check(G.gate_learning(fl, failed_runs, False, {"docs/lessons/2026-11-26-x.md": lesson}).state == G.OK, "red run with lesson passes")
    check(G.gate_learning(fl, [], False, {"docs/lessons/2026-11-26-x.md": "no front matter"}).state == G.FAIL, "malformed lesson fails")
    check(G.gate_learning(files("maps/a/b.py"), [], True, {}).state == G.FAIL, "incident fix needs a lesson")
    check(G.gate_learning(files("maps/a/b.py"), [], False, {}).state == G.OK, "clean PR needs no lesson")

    print("all gate tests passed")


if __name__ == "__main__":
    main()
