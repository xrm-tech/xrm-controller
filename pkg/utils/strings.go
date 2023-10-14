package utils

import "unsafe"

// UnsafeString returns the string under byte buffer
func UnsafeString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// Clone returns a fresh copy of s.
// It guarantees to make a copy of s into a new allocation,
// which can be important when retaining only a small substring
// of a much larger string. Using Clone can help such programs
// use less memory. Of course, since using Clone makes a copy,
// overuse of Clone can make programs use more memory.
// Clone should typically be used only rarely, and only when
// profiling indicates that it is needed.
// For strings of length zero the string "" will be returned
// and no allocation is made.
func Clone(s string) string {
	if len(s) == 0 {
		return ""
	}
	b := make([]byte, len(s))
	copy(b, s)
	return UnsafeString(b)
}

func StringsReverse(a []string) {
	l := len(a) - 1
	for i := 0; i < len(a)/2; i++ {
		a[i], a[l-i] = a[l-i], a[i]
	}
}
