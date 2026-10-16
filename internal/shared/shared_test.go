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
		want := "pkg: Op on nil receiver"
		sharedtest.Panics(t, "MustNotNil(nil)", want, func() {
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
				t.Errorf("Greater(%d, %d) = %d, want %d",
					tt.a, tt.b, got, tt.want)
			}
		})
	}

	t.Run("sorts descending", func(t *testing.T) {
		got := []int{2, 3, 1}
		slices.SortFunc(got, Greater)

		if want := []int{3, 2, 1}; !slices.Equal(got, want) {
			t.Errorf("SortFunc(s, Greater) = %v, want %v", got, want)
		}
	})
}

func TestShrinkSlice(t *testing.T) {
	t.Run("right-sized slice is returned unchanged", func(t *testing.T) {
		s := []int{1, 2, 3}

		got := ShrinkSlice(s)

		if &got[0] != &s[0] {
			t.Errorf("&ShrinkSlice(s)[0] = %p, want %p (no copy)",
				&got[0], &s[0])
		}
	})

	t.Run("spare capacity is dropped by copying", func(t *testing.T) {
		s := make([]int, 3, 16)
		copy(s, []int{1, 2, 3})

		got := ShrinkSlice(s)

		if want := []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("ShrinkSlice(s) = %v, want %v", got, want)
		}

		if got := cap(got); got != 3 {
			t.Errorf("cap(ShrinkSlice(s)) = %d, want 3", got)
		}

		if &got[0] == &s[0] {
			t.Errorf("&ShrinkSlice(s)[0] = %p, want new array (not %p)",
				&got[0], &s[0])
		}
	})

	t.Run("empty slice with capacity", func(t *testing.T) {
		if got := len(ShrinkSlice(make([]int, 0, 8))); got != 0 {
			t.Errorf("len(ShrinkSlice(empty)) = %d, want 0", got)
		}

		if got := cap(ShrinkSlice(make([]int, 0, 8))); got != 0 {
			t.Errorf("cap(ShrinkSlice(empty)) = %d, want 0", got)
		}
	})
}

func TestGrowSlice(t *testing.T) {
	t.Run("non-positive n returns s unchanged", func(t *testing.T) {
		for _, n := range []int{0, -1} {
			s := make([]int, 3)

			got := GrowSlice(s, n)

			if &got[0] != &s[0] || cap(got) != cap(s) {
				t.Errorf("GrowSlice(s, %d) = %p cap %d, want %p cap %d (s)",
					n, &got[0], cap(got), &s[0], cap(s))
			}
		}
	})

	t.Run("sufficient capacity returns s unchanged", func(t *testing.T) {
		s := make([]int, 3, 8)

		got := GrowSlice(s, 5)

		if &got[0] != &s[0] || cap(got) != 8 {
			t.Errorf("GrowSlice(s, 5) = %p cap %d, want %p cap 8 (s)",
				&got[0], cap(got), &s[0])
		}
	})

	t.Run("insufficient capacity reallocates", func(t *testing.T) {
		s := []int{1, 2, 3}

		got := GrowSlice(s, 10)

		if want := []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("GrowSlice(s, 10) = %v, want %v", got, want)
		}

		if got := cap(got); got < 13 {
			t.Errorf("cap(GrowSlice(s, 10)) = %d, want >= 13", got)
		}

		if &got[0] == &s[0] {
			t.Errorf("&GrowSlice(s, 10)[0] = %p, want new array (not %p)",
				&got[0], &s[0])
		}
	})

	t.Run("nil slice grows", func(t *testing.T) {
		s := GrowSlice[int](nil, 4)

		if got := len(s); got != 0 {
			t.Errorf("len(GrowSlice(nil, 4)) = %d, want 0", got)
		}

		if got := cap(s); got < 4 {
			t.Errorf("cap(GrowSlice(nil, 4)) = %d, want >= 4", got)
		}
	})
}
