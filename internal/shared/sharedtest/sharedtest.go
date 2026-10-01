// Package sharedtest provides test helpers shared by the container packages.
package sharedtest

import (
	"strings"
	"testing"
)

// Panics runs fn and fails the test unless it panics. When want is not empty,
// the panic value must be a string containing want. name identifies the call
// in failure messages.
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
			t.Errorf("%s panicked with %v, want message containing %q", name, r, want)
		}
	}()
	fn()
}
