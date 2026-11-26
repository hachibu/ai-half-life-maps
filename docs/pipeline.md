# The pipeline

The agent (Claude) works when a human gives it a task; from the pull request on, CI takes over and no
human has to be in the loop. Quality is held by **deterministic gates**: code that looks at GitHub data and
the compiled maps and says ok / pending / fail. No gate asks a model for its opinion. Review is optional: an
agent may merge its own PR when every gate is green, and a review that asks for changes on the *exact*
commit blocks it.

## The graph

```
idea ──spec──▶ ready ──claim──▶ claimed ──scope+description──▶ PR ──checks──▶ built+shown
     ──ratchet──▶ ──review──▶ reviewed ──learning──▶ mergeable ══merge══▶ main ──health──▶ healthy
                                                                              └─ red ─▶ incident (P0)
```

| gate | where | passes when |
|---|---|---|
| spec | task issue | Map, Paths, Goal, Acceptance criteria filled; paths inside the repo; map exists (or `new:`); dependencies closed (else `blocked`) |
| claim | task issue | one agent's `claimed:` label; no overlap with an earlier claim's paths; expires after 48 h without a PR |
| scope | PR | `Closes #task` claimed by the PR's agent (from the branch prefix); every file inside the task's paths (+ `docs/lessons/`); no build output; rule files need `human-approved` |
| description | PR | every template section filled; map PRs name at least one view |
| checks | PR | `map-check` green on the head commit: build, limits, placements, wiring, routes, **quality ratchet** (satisfaction + holdout vs main), **visual gate** (changed maps show a changed/new view; no blank shots) |
| ratchet | PR | no map test file shrinks (deleting checks needs `human-approved`) |
| review | PR | nobody but the author asked for changes on this commit: no GitHub "Request changes" review on it, and no `agent-review` block with `verdict: changes`. No approval is needed. `human-approved` overrides; `hold` pauses |
| learning | PR | a PR that ever had a red map-check, or that fixes an incident, adds a valid lesson |
| merge | PR | all of the above ok → CI squash-merges, deletes the branch, dispatches release |
| health | main | release builds and tests every map; green → new quality baseline; red → `incident` issue, fixed before other work |

All gates live in `tools/pipeline/gates.py` (tests: `make pipeline-test`). `.github/workflows/pipeline.yml`
runs them from **main's** code on every PR/issue/review/check event and hourly, so a PR can't weaken its own
judge. Changing the rules (`.github/`, `tools/pipeline/`, `AGENTS.md`, `Makefile`) needs a human's
`human-approved` label.

## How agents are started

Agents run only when a human asks (this keeps AI usage under control). Give an agent a task and it opens a
Task issue, claims it, reads the lessons tagged for it, and takes it through the gates. There is no scheduler
and no unattended worker. Everything after the PR is opened (gates, auto-merge, release, health, ledger)
runs in GitHub Actions, which is free for public repos.

## The ledger

Every gate decision, merge, expired claim and health result is an event file on the `ledger` branch
(append-only, one file per event, written only by CI, so no conflicts). SQLite is the query engine:

```
make ledger                                     # fetch + build build/ledger/ledger.sqlite + LEDGER.md
python -m tools.pipeline.ledger query "SELECT agent, gate, n FROM failures ORDER BY n DESC"
python -m tools.pipeline.ledger query "SELECT * FROM latest WHERE state != 'ok'"
```

Views: `events`, `latest` (last state per item and gate), `merges`, `failures`, `incidents`.
`scores/main.json` on the same branch holds each map's quality baseline. `LEDGER.md` is re-rendered hourly.

## Quality: satisfaction and holdout scenarios

- **Satisfaction** (`tools/checks/satisfy`, Go): thousands of standable points sampled on a jittered grid
  and linked wherever the walker can move; scores weapon rush from spawns, one-way traps, reachable items and
  spawn connectivity (0-100). CI seeds the sample with the `SATISFACTION_SEED` secret. A PR may not lower a
  map's score by more than 2 points.
- **Holdout scenarios** (`tools/checks/holdout.py`, evaluated by `satisfy`): human-written checks the agents can't read, encrypted in
  `holdout/scenarios.enc` with the `HOLDOUT_KEY` secret. CI reports only a pass rate; it may not drop.

## Humans

- Emergency stop: label any PR or task `hold`.
- Rule changes: add `human-approved` to a PR touching `.github/`, `tools/pipeline/`, `AGENTS.md` or `Makefile`.
- Watch: `LEDGER.md` on the `ledger` branch, the Pipeline comment on each PR, incident issues.

## Handy

Per-texture light samples of a compiled map (see the light-budget lesson):

```
python - <<'EOF'
from collections import Counter; from tools.bsp.reader import load
m = load("build/MAP/MAP.bsp"); c = Counter()
for f, e in zip(m.faces, m.fext):
    if e: c[m.tex[m.texinfo[f[4]][8]][0]] += (e[0][1] // 16 + 1) * (e[1][1] // 16 + 1)
print(c.most_common(12))
EOF
```
