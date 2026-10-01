package shared

import (
	"slices"
	"testing"

	"github.com/Nergous/structures/internal/shared/sharedtest"
)

func TestMustNotNil(t *testing.T) {
	t.Run("non-nil does not panic", func(t *testing.T) {
		v := 1
		MustNotNil(&v, "pkg", "Op")
	})

	t.Run("nil panics with package and operation", func(t *testing.T) {
		var p *int
		sharedtest.Panics(t, "MustNotNil(nil)", "pkg: Op on nil receiver", func() {
			MustNotNil(p, "pkg", "Op")
		})
	})
}

func TestGreater(t *testing.T) {
	tests := []struct {
		a, b int
		want int
	}{
		{1, 2, 1},
		{2, 1, -1},
		{3, 3, 0},
	}
	for _, tt := range tests {
		if got := Greater(tt.a, tt.b); got != tt.want {
			t.Errorf("Greater(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}

	s := []int{2, 3, 1}
	slices.SortFunc(s, Greater)
	if !slices.Equal(s, []int{3, 2, 1}) {
		t.Errorf("SortFunc(Greater) = %v, want [3 2 1]", s)
	}
}

func TestShrinkSlice(t *testing.T) {
	t.Run("right-sized slice is returned unchanged", func(t *testing.T) {
		s := []int{1, 2, 3}
		got := ShrinkSlice(s)
		if &got[0] != &s[0] {
			t.Errorf("ShrinkSlice copied a right-sized slice")
		}
	})

	t.Run("spare capacity is dropped by copying", func(t *testing.T) {
		s := make([]int, 3, 16)
		copy(s, []int{1, 2, 3})
		got := ShrinkSlice(s)
		if cap(got) != 3 || !slices.Equal(got, []int{1, 2, 3}) {
			t.Errorf("ShrinkSlice = %v (cap %d), want [1 2 3] (cap 3)", got, cap(got))
		}
		if &got[0] == &s[0] {
			t.Errorf("ShrinkSlice reused the old array")
		}
	})

	t.Run("empty slice with capacity", func(t *testing.T) {
		got := ShrinkSlice(make([]int, 0, 8))
		if len(got) != 0 || cap(got) != 0 {
			t.Errorf("ShrinkSlice(empty) len=%d cap=%d, want 0, 0", len(got), cap(got))
		}
	})
}
