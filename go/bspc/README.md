# bspc: the Go BSP compiler

`make build` compiles maps with this: a Go port of `tools/bsp/compile.py` (kept as the reference,
`make build COMPILER=py`). Standard library only; `go build` in this folder or `make bspc`.

| map | Python | Go | output |
|---|---|---|---|
| _template | 0.4 s | 0.01 s | byte-identical |
| battleaxecombatleague | ~236 s | 31 s | identical except the sign of zero in 28 plane distances (-0.0 vs +0.0: the same number) |
| prairieloop | ~510 s | 4.6 s | byte-identical |

(2-core sandbox; GitHub's runners have more cores, and Go uses all of them.)

## Why it matches Python exactly

The port keeps every algorithm and every loop order. Parallel work only happens where the results are
independent and are put back in the original order: CSG per brush, BSP subtrees, the four hulls, and lighting
per face (each luxel still adds its lights in the same order).

Floating point is matched operation by operation. numpy does not compute a 3-element dot product as
`a0*b0 + a1*b1 + a2*b2`; we measured (4000/4000 random cases each) what each form really does:

| numpy expression | what it computes | Go |
|---|---|---|
| `a @ b` (1-D), `A @ M.T` | `fma(a2,b2, fma(a1,b1, a0*b0))` | `Vec.Dot` |
| `A @ b` (rows times a vector, OpenBLAS gemv) | `fma(a2,b2, fma(a0,b0, a1*b1))` | `Vec.DotM` |
| `einsum` over 3 | `(a0*b0 + a2*b2) + a1*b1` | `Vec.DotE` |
| `norm(x, axis=1)` | `sqrt((x0² + x1²) + x2²)` | inline |

Using plain multiply-adds instead flipped a few near-threshold splits and changed BattleAxe's geometry.
Products are wrapped in `float64()` so the Go compiler never fuses them on arm64.

`make compiler-parity` (run in CI) compiles the template map with both compilers and requires identical bytes.
If you change the algorithm, change both compilers or retire the Python one in the same pull request.
