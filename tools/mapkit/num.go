package mapkit

import (
	"math"
	"strconv"
	"strings"
)

// G formats a number like C's and Python's "%g": six significant digits, trailing zeros dropped, exponent form
// outside 1e-4..1e6. Map coordinates are written with it, so the .map text stays stable across refactors.
func G(v float64) string {
	if v == 0 {
		if math.Signbit(v) {
			return "-0"
		}
		return "0"
	}
	if math.IsInf(v, 0) || math.IsNaN(v) {
		return strings.ToLower(strconv.FormatFloat(v, 'g', -1, 64))
	}
	e := strconv.FormatFloat(v, 'e', 5, 64) // d.ddddde±XX, correctly rounded
	exp, _ := strconv.Atoi(e[strings.IndexByte(e, 'e')+1:])
	if exp < -4 || exp >= 6 {
		mant, ex, _ := strings.Cut(e, "e")
		mant = trimZeros(mant)
		sign := ex[0]
		digits := strings.TrimLeft(ex[1:], "0")
		for len(digits) < 2 {
			digits = "0" + digits
		}
		return mant + "e" + string(sign) + digits
	}
	return trimZeros(strconv.FormatFloat(v, 'f', 5-exp, 64))
}

func trimZeros(s string) string {
	if !strings.Contains(s, ".") {
		return s
	}
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}

// Repr formats a float like Python's repr(): the shortest text that reads back the same, with ".0" on whole
// numbers. Entity keys given as floats are written with it.
func Repr(v float64) string {
	if math.IsInf(v, 0) || math.IsNaN(v) {
		return strings.ToLower(strconv.FormatFloat(v, 'g', -1, 64))
	}
	s := strconv.FormatFloat(v, 'e', -1, 64) // shortest digits; Go also writes two-digit exponents like Python
	_, ex, _ := strings.Cut(s, "e")
	if exp, _ := strconv.Atoi(ex); exp < -4 || exp >= 16 {
		return s
	}
	f := strconv.FormatFloat(v, 'f', -1, 64)
	if !strings.Contains(f, ".") {
		f += ".0"
	}
	return f
}

// Round is Python's round(x): halves go to the even neighbour, and the result is an integer (never -0).
func Round(x float64) float64 { return math.RoundToEven(x) + 0 }

// Round1 is Python's round(x, 1): correctly rounded to one decimal.
func Round1(x float64) float64 {
	v, _ := strconv.ParseFloat(strconv.FormatFloat(x, 'f', 1, 64), 64)
	return v
}

// PyMod is Python's integer %: the result takes the divisor's sign.
func PyMod(a, n int) int { return ((a % n) + n) % n }

// FloorDiv is Python's float // for the cell keys of Chunks.
func FloorDiv(a, b float64) int { return int(math.Floor(a / b)) }

// F6 formats like "%.6f".
func F6(v float64) string { return strconv.FormatFloat(v, 'f', 6, 64) }

// F4 formats like "%.4f".
func F4(v float64) string { return strconv.FormatFloat(v, 'f', 4, 64) }

// Deg converts radians to degrees the way Python's math.degrees does (x * (180/pi), both as doubles).
func Deg(x float64) float64 { return x * degPerRad }

var (
	piVar     = math.Pi
	degPerRad = 180.0 / piVar
)
