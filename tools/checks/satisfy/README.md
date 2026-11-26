# satisfy: the Go satisfaction scorer

`make quality` (and CI) score maps with this: a Go port of `tools/checks/satisfaction.py`, which stays as
the reference (the holdout scenarios reuse its graph). Same samples, same graph, same score: CI runs
`make compiler-parity`, which requires identical JSON for the template map with a non-default seed.

| map | Python | Go | result |
|---|---|---|---|
| battleaxecombatleague | 35 s | 1.5 s | identical (seed 0 and seed 987654321) |
| prairieloop | 21 s | 2.5 s | identical (seed 0 and seed 987654321) |

What it takes to get the same numbers:

- `rng.go` reproduces `numpy.random.default_rng(seed)`: SeedSequence hashing, PCG64 (XSL-RR 128/64),
  `next_double` and `uniform`. Checked against numpy for seeds 0, 1, 12345, 2^40 and 2^64-1.
- `num.go`: `np.dot` on 3-vectors and `np.linalg.norm` on 2-vectors are FMA chains; Python's `math.hypot` is
  correctly rounded (double-double with an exact big-float fallback, 100,000/100,000 matches); Python's
  float `//` and `round(x, n)` are reproduced exactly; `np.arange` fills `start + i*((start+step)-start)`.
- The point sampling and the navigation graph run on all cores; everything order-dependent (the random
  sequence, Dijkstra's tie-breaking on (distance, node)) stays in Python's order.
