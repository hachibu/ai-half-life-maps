# bspc: the Go BSP compiler

`make build` compiles maps with this. It is the only BSP compiler in the repo: it started as a byte-exact port of
a Python compiler (`tools/bsp/compile.py`, deleted in #24; see git history). Standard library only; `make bspc`
builds `build/bin/bspc` from the repo's `go.mod`.

Speed against the old Python compiler, measured when it was ported:

| map | Python | Go | output |
|---|---|---|---|
| _template | 0.4 s | 0.01 s | byte-identical |
| battleaxecombatleague | ~236 s | 31 s | identical except the sign of zero in 28 plane distances (-0.0 vs +0.0: the same number) |
| prairieloop | ~510 s | 4.6 s | byte-identical |

(2-core sandbox; GitHub's runners have more cores, and Go uses all of them.)

## Golden output

There is no second compiler to compare against any more. Instead `make golden` (run in CI) compiles the template
map and prairieloop and requires their SHA-256 to match `golden.sha256`. Refactoring, linting or speeding up the
compiler must leave that unchanged. If you change the output on purpose (a new feature, a fix, a different
floating-point order), run `make golden-update`, commit `golden.sha256`, and say in the pull request why the
output changed and how you checked the new maps (`make build test views` for every map).

## Floating point

The code still uses the exact operation order of the Python original, because that is what produced today's
golden output. numpy computes small dot products with fused multiply-adds, so the port has several dot forms:

| numpy expression | what it computes | Go |
|---|---|---|
| `a @ b` (1-D), `A @ M.T` | `fma(a2,b2, fma(a1,b1, a0*b0))` | `Vec.Dot` |
| `A @ b` (rows times a vector, OpenBLAS gemv) | `fma(a2,b2, fma(a0,b0, a1*b1))` | `Vec.DotM` |
| `einsum` over 3 | `(a0*b0 + a2*b2) + a1*b1` | `Vec.DotE` |
| `norm(x, axis=1)` | `sqrt((x0² + x1²) + x2²)` | inline |

Changing one of these changes last bits, which can flip a near-threshold split and change a map's geometry:
that's an output change (golden update + map checks), not a refactor. Products are wrapped in `float64()` so
the Go compiler never fuses them on arm64; keep that unless you mean to change the output.
