package core

const (
	// OrdEpsilon defines the minimum gap before rebalancing.
	OrdEpsilon = 1e-6
)

// Midpoint returns the midpoint between two ord values.
func Midpoint(a, b float64) float64 {
	return (a + b) / 2
}

// NextOrd returns an ord slightly greater than value.
func NextOrd(value float64) float64 {
	return value + 1
}

// PrevOrd returns an ord slightly less than value.
func PrevOrd(value float64) float64 {
	return value - 1
}
