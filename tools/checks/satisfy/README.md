# satisfy: the satisfaction scorer and holdout checker

`make quality` (and CI) score maps with this, and the holdout scenarios run on its navigation graph
(`satisfy map.bsp --holdout scenarios.json|-`, called by `tools/checks/holdout.py`). It is the only scorer in
the repo: it started as an exact port of `tools/checks/satisfaction.py` (deleted in #24; see git history).

What it measures (score = 100 x the mean of the four):

- `weapon_rush`: spawns that reach a weapon or ammo within 4800 units of running (~15 s)
- `no_traps`: points reachable from a spawn that can still get back to a spawn (no one-way dead ends)
- `items`: items a player can reach from some spawn
- `spawn_links`: spawn pairs where each can reach the other

Standable points come from dropping a player box down a jittered grid of columns, seeded by
`SATISFACTION_SEED` (a CI secret, so agents can't tune a map to CI's exact sample; 0 locally). Two points are
linked when the walker (18-unit steps, free drops) can move between them. A PR may not lower a map's score by
more than 2 points against main with the same seed (`tools/ci/quality.py`).

## Golden output

`make golden` (run in CI) scores the template map and prairieloop with seed 4242 and requires the result to
match `golden.json`. If you change the scoring on purpose, run `make golden-update`, commit `golden.json`, and
say in the pull request why the scores changed.

Speed against the old Python scorer, measured when it was ported:

| map | Python | Go |
|---|---|---|
| battleaxecombatleague | 35 s | 1.5 s |
| prairieloop | 21 s | 2.5 s |

The random numbers reproduce numpy's `default_rng` (PCG64, `rng.go`), and `num.go` reproduces Python's
`math.hypot`, float `//` and `round`, because that is what today's golden output was recorded with. Changing
those changes scores: an output change (golden update), not a refactor.
