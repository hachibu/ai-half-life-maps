"""Offline tests for the gates and the scheduler (no network).   python -m tools.pipeline.test_gates"""
from tools.pipeline import gates as G
from tools.pipeline import model as M
from tools.pipeline import next as N

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

    other = [dict(number=2, agent="codex", paths=["maps/prairieloop/zones/park.py"])]
    check(G.gate_claim(t, other, {"claude": 1}).state == G.FAIL, "overlapping earlier claim fails")
    check(G.gate_claim(t, [dict(number=2, agent="codex", paths=["maps/cagematch/"])], {"claude": 1}).state == G.OK, "disjoint claim passes")
    check(G.gate_claim(t, [], {"claude": 5}).state == G.OK, "human-assigned claims are not WIP-limited")

    tasks = {5: t}
    ok = G.gate_scope(pr(), "claude", files("maps/prairieloop/build.py", "docs/lessons/2026-x.md"), tasks, set())
    check(ok.state == G.OK, "in-scope PR passes")
    check(G.gate_scope(pr(), "claude", files("maps/cagematch/build.py"), tasks, set()).state == G.FAIL, "out-of-scope file fails")
    check(G.gate_scope(pr(body="no link"), "claude", files("maps/prairieloop/a.py"), tasks, set()).state == G.FAIL, "missing Closes fails")
    check(G.gate_scope(pr(), "codex", files("maps/prairieloop/a.py"), tasks, set()).state == G.FAIL, "task claimed by other agent fails")
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
    check(rev(blk("codex", "approve")).state == G.OK, "other agent approval passes")
    check(rev(blk("claude", "approve")).state == G.PENDING, "self-review ignored")
    check(rev(blk("codex", "approve", "1111111")).state == G.PENDING, "approval of an old commit ignored")
    check(rev(blk("codex", "changes")).state == G.FAIL, "changes requested fails")
    codexbot = dict(user=dict(login="chatgpt-codex-connector[bot]"), commit_id="abcdef1234567", body="", created_at="2026-01-02")
    check(rev(reviews=[codexbot]).state == G.OK, "Codex automatic review with no findings passes")
    check(rev(reviews=[codexbot], rc=[codexbot]).state == G.FAIL, "Codex inline finding fails")
    check(G.gate_review("codex", "abcdef1234567", [blk("claude", "approve")], [], [], "2026-01-01", set()).state == G.OK, "Claude approves Codex PR")
    check(G.gate_review("codex", "abcdef1234567", [blk("claude", "approve")], [], [], "2026-01-01", {"hold"}).state == G.PENDING, "hold label pauses")

    check(G.gate_review("claude", "abcdef1234567", [], [], [], "2026-01-01", {"human-approved"}).state == G.OK, "human-approved counts as review")

    lesson = "---\ntitle: X\ntags: [prairieloop, snow]\ngate: checks\nsource: PR #9\n---\n## What happened\na\n## Why\nb\n## Rule\nc\n"
    fl = files("maps/prairieloop/build.py") + [dict(filename="docs/lessons/2026-11-26-x.md", status="added", additions=9, deletions=0, changes=9)]
    failed_runs = [dict(conclusion="failure")]
    check(G.gate_learning(files("maps/prairieloop/build.py"), failed_runs, False, {}).state == G.FAIL, "red run without lesson fails learning")
    check(G.gate_learning(fl, failed_runs, False, {"docs/lessons/2026-11-26-x.md": lesson}).state == G.OK, "red run with lesson passes")
    check(G.gate_learning(fl, [], False, {"docs/lessons/2026-11-26-x.md": "no front matter"}).state == G.FAIL, "malformed lesson fails")
    check(G.gate_learning(files("maps/a/b.py"), [], True, {}).state == G.FAIL, "incident fix needs a lesson")
    check(G.gate_learning(files("maps/a/b.py"), [], False, {}).state == G.OK, "clean PR needs no lesson")

    # scheduler priorities
    st = lambda sha, gates: [dict(body=G.STATUS_MARK + " " + __import__("json").dumps(dict(sha=sha, gates=gates)) + " -->")]
    gok = lambda: dict(state="ok", reasons=[])
    p_codex = pr(10, "codex/thing", sha="c0ffee0000000")
    comments = {10: st("c0ffee0000000", dict(scope=gok(), description=gok(), checks=gok(), ratchet=gok(),
                                                review=dict(state="pending", reasons=[]), learning=gok()))}
    d = N.decide("claude", [issue(5, labels=("task", "ready"))], [p_codex], comments, {})
    check(d["action"] == "review" and d["pr"] == 10, "next: review owed beats claiming")
    d = N.decide("claude", [issue(5, labels=("task", "ready")), issue(6, labels=("task", "incident"))], [p_codex], comments, {})
    check(d["action"] == "fix-incident", "next: incident first")
    d = N.decide("claude", [issue(5, labels=("task", "ready"))], [], {}, {})
    check(d["action"] == "claim" and d["issue"] == 5, "next: claim the ready task")
    d = N.decide("claude", [], [], {}, {})
    check(d["action"] == "propose", "next: propose when the backlog is low")
    mine = pr(11, "claude/x", sha="1234567000000")
    bad = {11: st("1234567000000", dict(scope=dict(state="fail", reasons=["x"])))}
    d = N.decide("claude", [], [mine], bad, {})
    check(d["action"] == "fix-pr" and d["pr"] == 11, "next: fix my failing PR")
    print("all gate tests passed")


if __name__ == "__main__":
    main()
