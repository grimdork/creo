package util

// Set is a generic set type supporting Add, Has, and Remove operations.
type Set[T comparable] map[T]struct{}

// NewSet creates a new empty Set.
func NewSet[T comparable]() Set[T] {
	return make(Set[T])
}

// Add adds an element to the set.
func (s Set[T]) Add(k T) {
	s[k] = struct{}{}
}

// Has reports whether k is in the set.
func (s Set[T]) Has(k T) bool {
	_, ok := s[k]
	return ok
}

// Remove removes an element from the set.
func (s Set[T]) Remove(k T) {
	delete(s, k)
}
