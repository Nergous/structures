// Package heapcore holds the binary-heap sift primitives shared by the heap
// containers in this module.
//
// Every function works on a slice kept in heap order under compare: for each
// index i, compare(s[parent(i)], s[i]) <= 0, so the least element sits at
// s[0]. The comparator follows the slices.SortFunc contract.
//
// Functions come in two flavors. The plain ones (Heapify, Up, Down, Fix,
// RemoveAt) only reorder elements and are the hot path for binheap. The Moved
// variants take a moved callback, invoked as moved(s, i) every time an element
// is stored at a new index i, so callers such as pqueue can track element
// positions. Instead of swapping, the Moved variants shift elements into a
// hole and store the sifted element once at its final index, so moved fires
// once per shifted element plus once for the sifted element, about half as
// often as a callback per swap would. The two flavors are kept separate so the
// plain path pays nothing for position tracking.
package heapcore

// Heapify arranges s into heap order in O(n).
func Heapify[T any](s []T, compare func(a, b T) int) {
	for i := len(s)/2 - 1; i >= 0; i-- {
		Down(s, i, compare)
	}
}

// Down sifts the element at index i0 toward the leaves until no child is less
// than it. It reports whether the element moved.
func Down[T any](s []T, i0 int, compare func(a, b T) int) bool {
	i := i0
	n := len(s)
	for {
		left := 2*i + 1
		if left >= n || left < 0 { // left < 0 guards int overflow
			break
		}

		child := left
		if right := left + 1; right < n && compare(s[right], s[left]) < 0 {
			child = right
		}
		if compare(s[child], s[i]) >= 0 {
			break
		}

		s[i], s[child] = s[child], s[i]
		i = child
	}
	return i > i0
}

// Up sifts the element at index i toward the root until its parent is not
// greater than it.
func Up[T any](s []T, i int, compare func(a, b T) int) {
	for i > 0 {
		parent := (i - 1) / 2
		if compare(s[i], s[parent]) >= 0 {
			break
		}

		s[i], s[parent] = s[parent], s[i]
		i = parent
	}
}

// Fix restores heap order after the element at index i changed. The element
// may need to go either down or up; Fix tries down first and falls back to up
// when it did not move. Runs in O(log n).
func Fix[T any](s []T, i int, compare func(a, b T) int) {
	if !Down(s, i, compare) {
		Up(s, i, compare)
	}
}

// RemoveAt removes the element at index i and returns the shortened slice. The
// last element fills the hole, the vacated tail slot is zeroed so it no longer
// retains a reference, and heap order is restored around i. i must be a valid
// index. Runs in O(log n).
func RemoveAt[T any](s []T, i int, compare func(a, b T) int) []T {
	last := len(s) - 1
	if i != last {
		s[i] = s[last]
	}

	var zero T
	s[last] = zero
	s = s[:last]

	if i < last {
		Fix(s, i, compare)
	}
	return s
}

// HeapifyMoved is Heapify that reports relocated elements to moved. Elements
// that stay in place may not be reported; callers that need every index
// recorded must do so before calling it.
func HeapifyMoved[T any](s []T, compare func(a, b T) int, moved func(s []T, i int)) {
	for i := len(s)/2 - 1; i >= 0; i-- {
		DownMoved(s, i, compare, moved)
	}
}

// DownMoved is Down that reports every relocated element to moved. It produces
// the same arrangement as Down. When the element does not move, moved is not
// called.
func DownMoved[T any](s []T, i0 int, compare func(a, b T) int, moved func(s []T, i int)) bool {
	i := i0
	n := len(s)
	x := s[i0]
	for {
		left := 2*i + 1
		if left >= n || left < 0 { // left < 0 guards int overflow
			break
		}

		child := left
		if right := left + 1; right < n && compare(s[right], s[left]) < 0 {
			child = right
		}
		if compare(s[child], x) >= 0 {
			break
		}

		s[i] = s[child]
		moved(s, i)
		i = child
	}
	if i == i0 {
		return false
	}
	s[i] = x
	moved(s, i)
	return true
}

// UpMoved is Up that reports every relocated element to moved. It produces the
// same arrangement as Up. When the element does not move, moved is not called.
func UpMoved[T any](s []T, i int, compare func(a, b T) int, moved func(s []T, i int)) {
	i0 := i
	x := s[i]
	for i > 0 {
		parent := (i - 1) / 2
		if compare(x, s[parent]) >= 0 {
			break
		}

		s[i] = s[parent]
		moved(s, i)
		i = parent
	}
	if i != i0 {
		s[i] = x
		moved(s, i)
	}
}

// FixMoved is Fix that reports relocated elements to moved.
func FixMoved[T any](s []T, i int, compare func(a, b T) int, moved func(s []T, i int)) {
	if !DownMoved(s, i, compare, moved) {
		UpMoved(s, i, compare, moved)
	}
}

// RemoveAtMoved is RemoveAt that reports relocated elements to moved,
// including the tail element placed at index i. The removed element itself is
// not reported; read s[i] before calling if it is needed.
func RemoveAtMoved[T any](s []T, i int, compare func(a, b T) int, moved func(s []T, i int)) []T {
	last := len(s) - 1
	if i != last {
		s[i] = s[last]
	}

	var zero T
	s[last] = zero
	s = s[:last]

	if i < last {
		moved(s, i)
		FixMoved(s, i, compare, moved)
	}
	return s
}
