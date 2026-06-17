package util

import "fmt"

// IsIdent reports whether c is a valid identifier character (letter, digit, or underscore).
func IsIdent(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_'
}

// Unique deduplicates a slice while preserving order.
func Unique[T comparable](s []T) []T {
	seen := NewSet[T]()
	r := make([]T, 0, len(s))
	for _, v := range s {
		if !seen.Has(v) {
			seen.Add(v)
			r = append(r, v)
		}
	}
	return r
}

// FmtSize formats a byte count as a human-readable string (B, KiB, MiB, GiB).
func FmtSize(size int64) string {
	switch {
	case size >= 1<<30:
		return fmt.Sprintf("%.1f GiB", float64(size)/(1<<30))
	case size >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(size)/(1<<20))
	case size >= 1<<10:
		return fmt.Sprintf("%.1f KiB", float64(size)/(1<<10))
	default:
		return fmt.Sprintf("%d B", size)
	}
}
