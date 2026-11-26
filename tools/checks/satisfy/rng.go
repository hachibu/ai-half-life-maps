package main

// numpy.random.default_rng(seed): SeedSequence(seed) -> PCG64 (XSL-RR 128/64) -> next_double.
// Reproduced exactly so the Go scorer samples the same points as the Python one.

import "math/bits"

const (
	initA   = 0x43b0d7e5
	multA   = 0x931e8875
	initB   = 0x8b51f9dd
	multB   = 0x58f38ded
	mixMulL = 0xca01f9dd
	mixMulR = 0x4973f715
)

func hashmix(v uint32, h *uint32) uint32 {
	v ^= *h
	*h *= multA
	v *= *h
	v ^= v >> 16
	return v
}

func mix(x, y uint32) uint32 {
	r := mixMulL*x - mixMulR*y
	r ^= r >> 16
	return r
}

func seedSequenceState(seed uint64, nWords64 int) []uint64 {
	var entropy []uint32
	if seed == 0 {
		entropy = []uint32{0}
	}
	for s := seed; s > 0; s >>= 32 {
		entropy = append(entropy, uint32(s))
	}
	pool := make([]uint32, 4)
	h := uint32(initA)
	for i := range pool {
		if i < len(entropy) {
			pool[i] = hashmix(entropy[i], &h)
		} else {
			pool[i] = hashmix(0, &h)
		}
	}
	for s := range pool {
		for d := range pool {
			if s != d {
				pool[d] = mix(pool[d], hashmix(pool[s], &h))
			}
		}
	}
	for s := len(pool); s < len(entropy); s++ {
		for d := range pool {
			pool[d] = mix(pool[d], hashmix(entropy[s], &h))
		}
	}
	hb := uint32(initB)
	words := make([]uint32, 2*nWords64)
	for i := range words {
		v := pool[i%len(pool)]
		v ^= hb
		hb *= multB
		v *= hb
		v ^= v >> 16
		words[i] = v
	}
	out := make([]uint64, nWords64)
	for i := range out {
		out[i] = uint64(words[2*i]) | uint64(words[2*i+1])<<32
	}
	return out
}

type PCG64 struct{ hi, lo, incHi, incLo uint64 }

const mulHi, mulLo = 0x2360ED051FC65DA4, 0x4385DF649FCCF645

func (r *PCG64) step() {
	h, l := bits.Mul64(r.lo, mulLo)
	h += r.hi*mulLo + r.lo*mulHi
	var c uint64
	l, c = bits.Add64(l, r.incLo, 0)
	h, _ = bits.Add64(h, r.incHi, c)
	r.hi, r.lo = h, l
}

func (r *PCG64) next64() uint64 {
	r.step()
	x := r.hi ^ r.lo
	return bits.RotateLeft64(x, -int(r.hi>>58))
}

func NewRNG(seed uint64) *PCG64 {
	s := seedSequenceState(seed, 4)
	r := &PCG64{}
	r.incHi = s[2]<<1 | s[3]>>63
	r.incLo = s[3]<<1 | 1
	r.step()
	var c uint64
	r.lo, c = bits.Add64(r.lo, s[1], 0)
	r.hi, _ = bits.Add64(r.hi, s[0], c)
	r.step()
	return r
}

func (r *PCG64) Double() float64 { return float64(r.next64()>>11) * (1.0 / 9007199254740992.0) }

func (r *PCG64) Uniform(lo, hi float64) float64 { return lo + (hi-lo)*r.Double() }
