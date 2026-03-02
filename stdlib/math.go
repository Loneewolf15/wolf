package stdlib

import (
	"math"
	"math/rand"
)

// WolfMath implements the Wolf\Math standard library.
type WolfMath struct{}

// Pi is the mathematical constant.
const Pi = math.Pi

// E is Euler's number.
const E = math.E

// Abs returns the absolute value.
func (WolfMath) Abs(x float64) float64 {
	return math.Abs(x)
}

// Ceil rounds up to nearest integer.
func (WolfMath) Ceil(x float64) float64 {
	return math.Ceil(x)
}

// Floor rounds down to nearest integer.
func (WolfMath) Floor(x float64) float64 {
	return math.Floor(x)
}

// Round rounds to nearest integer.
func (WolfMath) Round(x float64) float64 {
	return math.Round(x)
}

// RoundTo rounds to n decimal places.
func (WolfMath) RoundTo(x float64, places int) float64 {
	pow := math.Pow(10, float64(places))
	return math.Round(x*pow) / pow
}

// Sqrt returns the square root.
func (WolfMath) Sqrt(x float64) float64 {
	return math.Sqrt(x)
}

// Pow returns x raised to the power y.
func (WolfMath) Pow(x, y float64) float64 {
	return math.Pow(x, y)
}

// Log returns the natural logarithm.
func (WolfMath) Log(x float64) float64 {
	return math.Log(x)
}

// Log10 returns the base-10 logarithm.
func (WolfMath) Log10(x float64) float64 {
	return math.Log10(x)
}

// Min returns the smaller of two values.
func (WolfMath) Min(a, b float64) float64 {
	return math.Min(a, b)
}

// Max returns the larger of two values.
func (WolfMath) Max(a, b float64) float64 {
	return math.Max(a, b)
}

// Clamp restricts a value between min and max.
func (WolfMath) Clamp(val, min, max float64) float64 {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

// Random returns a random float in [0.0, 1.0).
func (WolfMath) Random() float64 {
	return rand.Float64()
}

// RandomInt returns a random int in [0, max).
func (WolfMath) RandomInt(max int) int {
	return rand.Intn(max)
}

// RandomRange returns a random int in [min, max].
func (WolfMath) RandomRange(min, max int) int {
	return min + rand.Intn(max-min+1)
}

// Sin returns the sine.
func (WolfMath) Sin(x float64) float64 {
	return math.Sin(x)
}

// Cos returns the cosine.
func (WolfMath) Cos(x float64) float64 {
	return math.Cos(x)
}

// Tan returns the tangent.
func (WolfMath) Tan(x float64) float64 {
	return math.Tan(x)
}
