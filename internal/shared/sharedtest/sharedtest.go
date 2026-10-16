// Package sharedtest provides test helpers shared by the tests of the
// container packages and of internal/shared and internal/heapcore.
package sharedtest

import (
	"strings"
	"testing"
)

// Heap fails the test immediately unless data satisfies the heap property
// under compare: compare(data[parent], data[child]) <= 0 for every child.
// Failure messages name the offending pair as data[parent], data[child].
func Heap[T any](t testing.TB, data []T, compare func(a, b T) int) {
	t.Helper()

	for i := range len(data) - 1 {
		child := i + 1
		parent := (child - 1) / 2
		if got := compare(data[parent], data[child]); got > 0 {
			t.Fatalf("compare(data[%d], data[%d]) = %d, want <= 0 "+
				"(heap order broken)", parent, child, got)
		}
	}
}

// Panics runs fn and fails the test unless it panics. When want is not empty,
// the panic value must be a string containing want. name identifies the call
// in failure messages. It reports failures with t.Errorf, so the test goes on.
func Panics(t testing.TB, name, want string, fn func()) {
	t.Helper()

	defer func() {
		t.Helper()

		r := recover()
		if r == nil {
			t.Errorf("%s did not panic, want panic", name)
			return
		}

		if want == "" {
			return
		}

		if msg, _ := r.(string); !strings.Contains(msg, want) {
			t.Errorf("%s panicked with %v, want message containing %q",
				name, r, want)
		}
	}()

	fn()
}
