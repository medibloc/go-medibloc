package math

// XOR operation
func XOR(a, b bool) bool {
	return (a || b) && !(a && b)
}

// TernaryXOR operation
func TernaryXOR(a, b, c bool) bool {
	return (a || b || c) && !(a && b) && !(b && c) && !(a && c)
}
