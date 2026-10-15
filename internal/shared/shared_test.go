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
		name string
		a    int
		b    int
		want int
	}{
		{
			name: "smaller sorts after",
			a:    1,
			b:    2,
			want: 1,
		},
		{
			name: "larger sorts first",
			a:    2,
			b:    1,
			want: -1,
		},
		{
			name: "equal",
			a:    3,
			b:    3,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Greater(tt.a, tt.b); got != tt.want {
				t.Errorf("Greater(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}

	t.Run("sorts descending", func(t *testing.T) {
		s := []int{2, 3, 1}
		slices.SortFunc(s, Greater)

		if want := []int{3, 2, 1}; !slices.Equal(s, want) {
			t.Errorf("SortFunc(s, Greater) = %v, want %v", s, want)
		}
	})
}

func TestShrinkSlice(t *testing.T) {
	t.Run("right-sized slice is returned unchanged", func(t *testing.T) {
		s := []int{1, 2, 3}
		got := ShrinkSlice(s)

		if &got[0] != &s[0] {
			t.Errorf("&ShrinkSlice(s)[0] = %p, want %p (no copy)", &got[0], &s[0])
		}
	})

	t.Run("spare capacity is dropped by copying", func(t *testing.T) {
		s := make([]int, 3, 16)
		copy(s, []int{1, 2, 3})
		got := ShrinkSlice(s)

		if want := []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("ShrinkSlice(s) = %v, want %v", got, want)
		}

		if c := cap(got); c != 3 {
			t.Errorf("cap(ShrinkSlice(s)) = %d, want 3", c)
		}

		if &got[0] == &s[0] {
			t.Errorf("&ShrinkSlice(s)[0] = %p, want new array (not %p)", &got[0], &s[0])
		}
	})

	t.Run("empty slice with capacity", func(t *testing.T) {
		got := ShrinkSlice(make([]int, 0, 8))

		if n := len(got); n != 0 {
			t.Errorf("len(ShrinkSlice(empty)) = %d, want 0", n)
		}

		if c := cap(got); c != 0 {
			t.Errorf("cap(ShrinkSlice(empty)) = %d, want 0", c)
		}
	})
}
