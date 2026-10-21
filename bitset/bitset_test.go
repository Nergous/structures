package bitset

import (
	"maps"
	"math"
	"math/bits"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Nergous/structures/internal/shared/sharedtest"
)

func assertValues(t *testing.T, s *Set, want ...int) {
	t.Helper()

	if got := s.Len(); got != len(want) {
		t.Errorf("Len() = %d, want %d", got, len(want))
	}

	got := slices.Sorted(s.All())
	want = slices.Sorted(slices.Values(want))
	if !slices.Equal(got, want) {
		t.Errorf("values = %v, want %v", got, want)
	}
}

// assertInvariants fails the test unless s.n equals the number of set bits in
// s.data and s.data has no trailing empty word.
func assertInvariants(t *testing.T, s *Set) {
	t.Helper()

	var want int
	for _, w := range s.data {
		want += bits.OnesCount64(w)
	}

	if s.n != want {
		t.Errorf("n = %d, want %d set bits in data", s.n, want)
	}

	if l := len(s.data); l > 0 && s.data[l-1] == 0 {
		t.Errorf("data[%d] = 0, want no trailing empty word", l-1)
	}
}

func fill(n int) *Set {
	s := New()
	for i := range n {
		s.Add(i)
	}

	return s
}

// collectBackward drains s.Backward, failing the test instead of hanging if it
// yields more values than the set holds.
func collectBackward(t *testing.T, s *Set) []int {
	t.Helper()

	var got []int
	for v := range s.Backward() {
		got = append(got, v)
		if len(got) > s.Len() {
			t.Fatalf("Backward() yielded %v, more than Len() = %d",
				got, s.Len())
		}
	}

	return got
}

func TestSet_New(t *testing.T) {
	s := New()

	if got := s.Len(); got != 0 {
		t.Errorf("Len() = %d, want 0", got)
	}

	if got := s.Cap(); got != 0 {
		t.Errorf("Cap() = %d, want 0", got)
	}

	if got := s.IsEmpty(); !got {
		t.Errorf("IsEmpty() = %v, want true", got)
	}
}

func TestSet_NewWithCap(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want int
	}{
		{
			name: "negative clamps to zero",
			n:    -1,
			want: 0,
		},
		{
			name: "zero",
			n:    0,
			want: 0,
		},
		{
			name: "one",
			n:    1,
			want: 64,
		},
		{
			name: "one word minus one",
			n:    63,
			want: 64,
		},
		{
			name: "exactly one word",
			n:    64,
			want: 64,
		},
		{
			name: "one word plus one",
			n:    65,
			want: 128,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewWithCap(tt.n)

			if got := s.Len(); got != 0 {
				t.Errorf("Len() = %d, want 0", got)
			}

			if got := s.Cap(); got != tt.want {
				t.Errorf("Cap() = %d, want %d", got, tt.want)
			}

			if tt.n > 0 {
				s.Add(tt.n - 1)

				if got := s.Cap(); got != tt.want {
					t.Errorf("Cap() after Add(%d) = %d, want %d",
						tt.n-1, got, tt.want)
				}
			}

			s.Add(1)

			if got := s.Contains(1); !got {
				t.Errorf("Contains(1) after Add(1) = %v, want true", got)
			}
		})
	}
}

func TestSet_Of(t *testing.T) {
	t.Run("stores each value once", func(t *testing.T) {
		s := Of(3, 1, 3, 2, 1)
		assertValues(t, s, 3, 1, 2)
	})

	t.Run("values across word boundaries ascend", func(t *testing.T) {
		s := Of(128, 0, 64, 63, 127, 64)
		want := []int{0, 63, 64, 127, 128}

		if got := slices.Collect(s.All()); !slices.Equal(got, want) {
			t.Errorf("All() = %v, want %v", got, want)
		}

		if got := s.Len(); got != len(want) {
			t.Errorf("Len() = %d, want %d", got, len(want))
		}
	})

	t.Run("no arguments returns an empty usable set", func(t *testing.T) {
		s := Of()
		assertValues(t, s)

		if got := s.Cap(); got != 0 {
			t.Errorf("Cap() = %d, want 0", got)
		}

		s.Add(1)
		assertValues(t, s, 1)
	})

	t.Run("negative panics", func(t *testing.T) {
		sharedtest.Panics(t, "Of(1, -1)", "bitset: AddN of negative value",
			func() { Of(1, -1) })
	})
}

func TestSet_From(t *testing.T) {
	t.Run("copies the values", func(t *testing.T) {
		s := From([]int{1, 2, 3, 4, 5})
		assertValues(t, s, 1, 2, 3, 4, 5)
	})

	t.Run("never aliases the input", func(t *testing.T) {
		in := []int{1, 2, 3}
		s := From(in)

		in[0], in[1], in[2] = 7, 8, 9
		assertValues(t, s, 1, 2, 3)
	})

	t.Run("stores each value once", func(t *testing.T) {
		s := From([]int{3, 1, 3, 2, 1})
		assertValues(t, s, 3, 1, 2)
	})

	tests := []struct {
		name string
		in   []int
	}{
		{
			name: "nil input returns an empty usable set",
			in:   nil,
		},
		{
			name: "empty input returns an empty usable set",
			in:   []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := From(tt.in)
			assertValues(t, s)

			s.Add(1)
			assertValues(t, s, 1)
		})
	}
}

func TestSet_Len(t *testing.T) {
	s := New()
	s.AddN(1, 2, 2, 3, 64, 130, 130)

	if got := s.Len(); got != 5 {
		t.Errorf("Len() after AddN(1, 2, 2, 3, 64, 130, 130) = %d, want 5",
			got)
	}
	assertInvariants(t, s)

	s.Remove(1)

	if got := s.Len(); got != 4 {
		t.Errorf("Len() after Remove(1) = %d, want 4", got)
	}
	assertInvariants(t, s)

	s.Remove(1)
	s.Remove(500)

	if got := s.Len(); got != 4 {
		t.Errorf("Len() after removing absent values = %d, want 4", got)
	}
	assertInvariants(t, s)
}

func TestSet_Cap(t *testing.T) {
	t.Run("new set has zero cap", func(t *testing.T) {
		s := New()

		if got := s.Cap(); got != 0 {
			t.Errorf("Cap() = %d, want 0", got)
		}
	})

	t.Run("covers every added value while growing", func(t *testing.T) {
		s := New()
		for _, v := range []int{0, 63, 64, 1000, 5, 4096} {
			s.Add(v)

			c := s.Cap()
			if c <= v {
				t.Fatalf("Cap() after Add(%d) = %d, want > %d", v, c, v)
			}

			if c%wordBits != 0 {
				t.Fatalf("Cap() after Add(%d) = %d, want multiple of %d",
					v, c, wordBits)
			}
		}
	})

	t.Run("no reallocation within preallocated capacity", func(t *testing.T) {
		s := NewWithCap(128)
		want := s.Cap()

		for _, v := range []int{0, 127, 64, 1} {
			s.Add(v)
		}

		if got := s.Cap(); got != want {
			t.Errorf("Cap() = %d, want %d", got, want)
		}
	})
}

func TestSet_IsEmpty(t *testing.T) {
	s := New()

	if got := s.IsEmpty(); !got {
		t.Errorf("IsEmpty() on empty = %v, want true", got)
	}

	s.Add(1)

	if got := s.IsEmpty(); got {
		t.Errorf("IsEmpty() after Add(1) = %v, want false", got)
	}

	s.Remove(1)

	if got := s.IsEmpty(); !got {
		t.Errorf("IsEmpty() after Remove(1) = %v, want true", got)
	}

	t.Run("ignores allocated words", func(t *testing.T) {
		s := NewWithCap(128)

		if got := s.IsEmpty(); !got {
			t.Errorf("IsEmpty() after NewWithCap(128) = %v, want true", got)
		}

		s.Add(1000)
		s.Remove(1000)

		if got := s.IsEmpty(); !got {
			t.Errorf("IsEmpty() after Add/Remove(1000) = %v, want true", got)
		}
	})
}

func TestSet_Contains(t *testing.T) {
	t.Run("present and absent values", func(t *testing.T) {
		s := Of(1, 2, 63, 64, 1000)

		for _, v := range []int{1, 2, 63, 64, 1000} {
			if got := s.Contains(v); !got {
				t.Errorf("Contains(%d) = %v, want true", v, got)
			}
		}

		for _, v := range []int{0, 3, 62, 65, 999, 1001, 1 << 20} {
			if got := s.Contains(v); got {
				t.Errorf("Contains(%d) = %v, want false", v, got)
			}
		}
	})

	t.Run("false after Remove", func(t *testing.T) {
		s := Of(1, 2)
		s.Remove(2)

		if got := s.Contains(2); got {
			t.Errorf("Contains(2) after Remove(2) = %v, want false", got)
		}
	})

	t.Run("negative values are never present", func(t *testing.T) {
		for _, s := range []*Set{New(), Of(0, 63, 64), {}} {
			for _, v := range []int{-1, -64, -65, math.MinInt} {
				if got := s.Contains(v); got {
					t.Errorf("Contains(%d) = %v, want false", v, got)
				}
			}
		}
	})
}

func TestSet_ContainsAll(t *testing.T) {
	s := Of(1, 2, 3, 64, 1000)

	tests := []struct {
		name string
		vs   []int
		want bool
	}{
		{
			name: "no arguments",
			vs:   nil,
			want: true,
		},
		{
			name: "subset in one word",
			vs:   []int{1, 3},
			want: true,
		},
		{
			name: "subset across words",
			vs:   []int{1, 64, 1000},
			want: true,
		},
		{
			name: "duplicates",
			vs:   []int{1, 1, 64},
			want: true,
		},
		{
			name: "one absent",
			vs:   []int{1, 4},
			want: false,
		},
		{
			name: "beyond data",
			vs:   []int{1, 1 << 20},
			want: false,
		},
		{
			name: "negative",
			vs:   []int{1, -1},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.ContainsAll(tt.vs...); got != tt.want {
				t.Errorf("ContainsAll(%v) = %v, want %v", tt.vs, got, tt.want)
			}
		})
	}

	t.Run("false after Remove", func(t *testing.T) {
		s := Of(1, 2, 3)
		s.Remove(3)

		if got := s.ContainsAll(1, 3); got {
			t.Errorf("ContainsAll(1, 3) after Remove(3) = %v, want false", got)
		}
	})
}

func TestSet_ContainsAny(t *testing.T) {
	s := Of(1, 2, 3, 64, 1000)

	tests := []struct {
		name string
		vs   []int
		want bool
	}{
		{
			name: "no arguments",
			vs:   nil,
			want: false,
		},
		{
			name: "one present in one word",
			vs:   []int{4, 2},
			want: true,
		},
		{
			name: "one present in another word",
			vs:   []int{4, 1000},
			want: true,
		},
		{
			name: "match after misses",
			vs:   []int{-1, 1 << 20, 64},
			want: true,
		},
		{
			name: "none present",
			vs:   []int{4, 5, 65},
			want: false,
		},
		{
			name: "only beyond data",
			vs:   []int{1 << 20},
			want: false,
		},
		{
			name: "only negative",
			vs:   []int{-1, -64},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.ContainsAny(tt.vs...); got != tt.want {
				t.Errorf("ContainsAny(%v) = %v, want %v", tt.vs, got, tt.want)
			}
		})
	}

	t.Run("false after Remove", func(t *testing.T) {
		s := Of(1, 2, 3)
		s.Remove(2)

		if got := s.ContainsAny(4, 2); got {
			t.Errorf("ContainsAny(4, 2) after Remove(2) = %v, want false", got)
		}
	})
}

func TestSet_Min(t *testing.T) {
	t.Run("empty returns zero and false", func(t *testing.T) {
		allocated := NewWithCap(128)
		allocated.Add(1000)
		allocated.Remove(1000)

		for name, s := range map[string]*Set{
			"zero value":      {},
			"new":             New(),
			"allocated words": allocated,
		} {
			if got, ok := s.Min(); got != 0 || ok {
				t.Errorf("%s: Min() = (%d, %v), want (0, false)",
					name, got, ok)
			}
		}
	})

	tests := []struct {
		name string
		vs   []int
		want int
	}{
		{
			name: "zero is a value",
			vs:   []int{0, 5},
			want: 0,
		},
		{
			name: "within first word",
			vs:   []int{15, 1, 31, 5},
			want: 1,
		},
		{
			name: "last bit of a word",
			vs:   []int{63, 100},
			want: 63,
		},
		{
			name: "first word empty",
			vs:   []int{200, 64, 130},
			want: 64,
		},
		{
			name: "far value",
			vs:   []int{1 << 16},
			want: 1 << 16,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Of(tt.vs...)

			if got, ok := s.Min(); got != tt.want || !ok {
				t.Errorf("Min() = (%d, %v), want (%d, true)",
					got, ok, tt.want)
			}
		})
	}

	t.Run("follows Remove across words", func(t *testing.T) {
		s := Of(1, 64, 200)

		for _, want := range []int{1, 64, 200} {
			if got, ok := s.Min(); got != want || !ok {
				t.Fatalf("Min() = (%d, %v), want (%d, true)", got, ok, want)
			}

			s.Remove(want)
		}

		if got, ok := s.Min(); got != 0 || ok {
			t.Errorf("Min() after removing all = (%d, %v), want (0, false)",
				got, ok)
		}
	})
}

func TestSet_Max(t *testing.T) {
	t.Run("empty returns zero and false", func(t *testing.T) {
		allocated := NewWithCap(128)
		allocated.Add(1000)
		allocated.Remove(1000)

		for name, s := range map[string]*Set{
			"zero value":      {},
			"new":             New(),
			"allocated words": allocated,
		} {
			if got, ok := s.Max(); got != 0 || ok {
				t.Errorf("%s: Max() = (%d, %v), want (0, false)",
					name, got, ok)
			}
		}
	})

	tests := []struct {
		name string
		vs   []int
		want int
	}{
		{
			name: "zero is a value",
			vs:   []int{0},
			want: 0,
		},
		{
			name: "within first word",
			vs:   []int{15, 1, 31, 5},
			want: 31,
		},
		{
			name: "last bit of a word",
			vs:   []int{1, 63},
			want: 63,
		},
		{
			name: "first bit of a word",
			vs:   []int{1, 63, 64},
			want: 64,
		},
		{
			name: "far value",
			vs:   []int{3, 1 << 16},
			want: 1 << 16,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Of(tt.vs...)

			if got, ok := s.Max(); got != tt.want || !ok {
				t.Errorf("Max() = (%d, %v), want (%d, true)",
					got, ok, tt.want)
			}
		})
	}

	t.Run("skips empty trailing words", func(t *testing.T) {
		s := Of(1, 64, 1000)

		for _, want := range []int{1000, 64, 1} {
			if got, ok := s.Max(); got != want || !ok {
				t.Fatalf("Max() = (%d, %v), want (%d, true)", got, ok, want)
			}

			s.Remove(want)
		}

		if got, ok := s.Max(); got != 0 || ok {
			t.Errorf("Max() after removing all = (%d, %v), want (0, false)",
				got, ok)
		}
	})

	t.Run("ignores spare capacity", func(t *testing.T) {
		s := NewWithCap(128)
		s.Add(5)

		if got, ok := s.Max(); got != 5 || !ok {
			t.Errorf("Max() = (%d, %v), want (5, true)", got, ok)
		}
	})
}

func TestSet_All(t *testing.T) {
	t.Run("yields every value once in ascending order", func(t *testing.T) {
		s := Of(200, 64, 0, 63, 1000, 1, 64)
		s.Remove(1000)

		got := slices.Collect(s.All())
		if want := []int{0, 1, 63, 64, 200}; !slices.Equal(got, want) {
			t.Errorf("All() = %v, want %v", got, want)
		}
	})

	t.Run("empty set yields nothing", func(t *testing.T) {
		allocated := NewWithCap(128)
		allocated.Add(1000)
		allocated.Remove(1000)

		for name, s := range map[string]*Set{
			"zero value":      {},
			"new":             New(),
			"allocated words": allocated,
		} {
			if got := slices.Collect(s.All()); len(got) != 0 {
				t.Errorf("%s: All() = %v, want no elements", name, got)
			}
		}
	})

	t.Run("does not consume", func(t *testing.T) {
		s := Of(1, 2, 3)

		_ = slices.Collect(s.All())

		assertValues(t, s, 1, 2, 3)
	})

	t.Run("break stops iteration early", func(t *testing.T) {
		s := Of(3, 1, 2)

		var got []int
		for v := range s.All() {
			got = append(got, v)
			if len(got) == 2 {
				break
			}
		}

		if want := []int{1, 2}; !slices.Equal(got, want) {
			t.Errorf("All() with early break = %v, want %v", got, want)
		}

		if got := s.Len(); got != 3 {
			t.Errorf("Len() after break = %d, want 3", got)
		}
	})
}

func TestSet_Backward(t *testing.T) {
	t.Run("yields every value once in descending order", func(t *testing.T) {
		s := Of(200, 64, 0, 63, 1000, 1, 64)
		s.Remove(1000)

		got := collectBackward(t, s)
		if want := []int{200, 64, 63, 1, 0}; !slices.Equal(got, want) {
			t.Errorf("Backward() = %v, want %v", got, want)
		}
	})

	t.Run("empty set yields nothing", func(t *testing.T) {
		allocated := NewWithCap(128)
		allocated.Add(1000)
		allocated.Remove(1000)

		for name, s := range map[string]*Set{
			"zero value":      {},
			"new":             New(),
			"allocated words": allocated,
		} {
			if got := collectBackward(t, s); len(got) != 0 {
				t.Errorf("%s: Backward() = %v, want no elements", name, got)
			}
		}
	})

	t.Run("mirrors All", func(t *testing.T) {
		s := Of(5, 0, 63, 64, 127, 128, 4095)

		got := collectBackward(t, s)
		want := slices.Collect(s.All())
		slices.Reverse(want)

		if !slices.Equal(got, want) {
			t.Errorf("Backward() = %v, want reverse of All() %v", got, want)
		}
	})

	t.Run("does not consume", func(t *testing.T) {
		s := Of(1, 2, 3)

		_ = collectBackward(t, s)

		assertValues(t, s, 1, 2, 3)
	})

	t.Run("break stops iteration early", func(t *testing.T) {
		s := Of(3, 1, 2)

		var got []int
		for v := range s.Backward() {
			got = append(got, v)
			if len(got) == 2 {
				break
			}
		}

		if want := []int{3, 2}; !slices.Equal(got, want) {
			t.Errorf("Backward() with early break = %v, want %v", got, want)
		}

		if got := s.Len(); got != 3 {
			t.Errorf("Len() after break = %d, want 3", got)
		}
	})
}

func TestSet_Clone(t *testing.T) {
	t.Run("copies all values", func(t *testing.T) {
		s := Of(0, 5, 63, 64, 300, 1000)
		s.Remove(5)

		c := s.Clone()

		got := slices.Collect(c.All())
		if want := []int{0, 63, 64, 300, 1000}; !slices.Equal(got, want) {
			t.Errorf("All() of clone = %v, want %v", got, want)
		}
		assertInvariants(t, c)
	})

	t.Run("drops spare capacity", func(t *testing.T) {
		s := NewWithCap(4096)
		s.AddN(1, 70)

		c := s.Clone()

		if got, want := c.Cap(), len(s.data)*wordBits; got != want {
			t.Errorf("Cap() of clone = %d, want %d", got, want)
		}
	})

	t.Run("is independent of the original", func(t *testing.T) {
		s := Of(1, 2, 3, 64)

		c := s.Clone()
		c.Add(4)
		c.Remove(2)
		c.Remove(64)
		s.Remove(1)

		assertValues(t, c, 1, 3, 4)
		assertValues(t, s, 2, 3, 64)
		assertInvariants(t, c)
		assertInvariants(t, s)
	})

	t.Run("clone of empty set is empty and usable", func(t *testing.T) {
		c := New().Clone()

		if got := c.Len(); got != 0 {
			t.Errorf("Len() of clone = %d, want 0", got)
		}

		c.Add(1)

		if got := c.Contains(1); !got {
			t.Errorf("Contains(1) of clone after Add(1) = %v, want true", got)
		}
	})
}

func TestSet_Slice(t *testing.T) {
	t.Run("returns every value in ascending order", func(t *testing.T) {
		s := Of(1000, 3, 64, 0, 63, 2)

		got := s.Slice()
		if want := []int{0, 2, 3, 63, 64, 1000}; !slices.Equal(got, want) {
			t.Errorf("Slice() = %v, want %v", got, want)
		}

		if len(got) != cap(got) {
			t.Errorf("cap(Slice()) = %d, want %d", cap(got), len(got))
		}
	})

	t.Run("returns an independent copy", func(t *testing.T) {
		s := Of(3, 1, 2)

		out := s.Slice()
		for i := range out {
			out[i] = 99
		}

		assertValues(t, s, 3, 1, 2)
	})

	t.Run("empty set returns nil", func(t *testing.T) {
		allocated := NewWithCap(128)
		allocated.Add(1000)
		allocated.Remove(1000)

		for name, s := range map[string]*Set{
			"zero value":      {},
			"new":             New(),
			"allocated words": allocated,
		} {
			if got := s.Slice(); got != nil {
				t.Errorf("%s: Slice() = %v, want nil", name, got)
			}
		}
	})
}

func TestSet_Add(t *testing.T) {
	t.Run("reports whether the value was new", func(t *testing.T) {
		s := Of(1, 2, 3)

		if got := s.Add(2); got {
			t.Errorf("Add(2) on existing = %v, want false", got)
		}

		if got := s.Add(4); !got {
			t.Errorf("Add(4) = %v, want true", got)
		}

		s.Remove(1)

		if got := s.Add(1); !got {
			t.Errorf("Add(1) after Remove(1) = %v, want true", got)
		}

		assertValues(t, s, 1, 2, 3, 4)
		assertInvariants(t, s)
	})

	t.Run("grows across words", func(t *testing.T) {
		s := New()
		for _, v := range []int{0, 63, 64, 1000} {
			if got := s.Add(v); !got {
				t.Errorf("Add(%d) = %v, want true", v, got)
			}

			if got := s.Cap(); got <= v {
				t.Errorf("Cap() after Add(%d) = %d, want > %d", v, got, v)
			}
		}

		for _, v := range []int{1, 62, 65, 500, 999} {
			if got := s.Contains(v); got {
				t.Errorf("Contains(%d) = %v, want false", v, got)
			}
		}

		assertValues(t, s, 0, 63, 64, 1000)
		assertInvariants(t, s)
	})

	t.Run("negative panics and leaves the set unchanged", func(t *testing.T) {
		s := Of(1, 2)
		before := s.Cap()

		sharedtest.Panics(t, "Add(-1)", "negative", func() { s.Add(-1) })

		assertValues(t, s, 1, 2)
		if got := s.Cap(); got != before {
			t.Errorf("Cap() after Add(-1) = %d, want %d", got, before)
		}
	})

	t.Run("clears stale words in spare capacity", func(t *testing.T) {
		s := New()
		s.data = make([]uint64, 1, 4)
		stale := s.data[:4]
		stale[1], stale[2] = ^uint64(0), ^uint64(0)

		s.Add(130)

		assertValues(t, s, 130)
		assertInvariants(t, s)
	})
}

func TestSet_AddN(t *testing.T) {
	t.Run("adds every value once", func(t *testing.T) {
		s := Of(1)

		s.AddN(2, 1000, 3, 3, 64, 1, 1000)

		assertValues(t, s, 1, 2, 3, 64, 1000)
		assertInvariants(t, s)
	})

	t.Run("no arguments is a no-op", func(t *testing.T) {
		s := Of(1)
		before := s.Cap()

		s.AddN()

		assertValues(t, s, 1)
		if got := s.Cap(); got != before {
			t.Errorf("Cap() after AddN() = %d, want %d", got, before)
		}
	})

	t.Run("grows the backing array once", func(t *testing.T) {
		s := New()
		vs := []int{5, 4000, 3, 70, 1000}

		// A single Add of the largest value grows the array exactly once;
		// comparing against it keeps the check valid under -race, which
		// adds its own allocations.
		want := testing.AllocsPerRun(10, func() {
			s.data, s.n = nil, 0
			s.Add(4000)
		})

		got := testing.AllocsPerRun(10, func() {
			s.data, s.n = nil, 0
			s.AddN(vs...)
		})

		if got != want {
			t.Errorf("AddN allocated %v times, want %v (one growth)",
				got, want)
		}
	})

	t.Run("negative panics and leaves the set unchanged", func(t *testing.T) {
		s := Of(1, 2)
		before := s.Cap()

		sharedtest.Panics(t, "AddN(5, -1, 6)", "negative",
			func() { s.AddN(5, -1, 6) })

		assertValues(t, s, 1, 2)
		if got := s.Cap(); got != before {
			t.Errorf("Cap() after failed AddN = %d, want %d", got, before)
		}
	})

	t.Run("zero value receiver stores the values", func(t *testing.T) {
		var s Set

		s.AddN(2, 100, 1)

		assertValues(t, &s, 1, 2, 100)
	})
}

func TestSet_AddSet(t *testing.T) {
	t.Run("adds the values of other", func(t *testing.T) {
		s := Of(1, 2, 3)
		other := Of(5, 2, 4, 1)

		s.AddSet(other)

		assertValues(t, s, 1, 2, 3, 4, 5)
		assertValues(t, other, 1, 2, 4, 5)
		assertInvariants(t, s)
	})

	t.Run("other reaches further words", func(t *testing.T) {
		s := Of(1, 64)
		other := Of(64, 65, 1000)

		s.AddSet(other)

		assertValues(t, s, 1, 64, 65, 1000)
		assertInvariants(t, s)
	})

	t.Run("other is shorter", func(t *testing.T) {
		s := Of(1, 1000)
		other := Of(2, 64)

		s.AddSet(other)

		assertValues(t, s, 1, 2, 64, 1000)
		assertInvariants(t, s)
	})

	t.Run("grows exactly to other", func(t *testing.T) {
		s := New()

		s.AddSet(Of(1))

		if got := len(s.data); got != 1 {
			t.Errorf("len(data) = %d, want 1", got)
		}
		assertInvariants(t, s)
	})

	t.Run("nil other is a no-op", func(t *testing.T) {
		s := Of(1, 2)

		s.AddSet(nil)

		assertValues(t, s, 1, 2)
	})

	t.Run("empty other is a no-op", func(t *testing.T) {
		s := Of(1, 2)
		before := s.Cap()

		s.AddSet(New())

		assertValues(t, s, 1, 2)
		if got := s.Cap(); got != before {
			t.Errorf("Cap() after AddSet(empty) = %d, want %d", got, before)
		}
	})

	t.Run("self is a no-op", func(t *testing.T) {
		s := Of(1, 2, 3)

		s.AddSet(s)

		assertValues(t, s, 1, 2, 3)
		assertInvariants(t, s)
	})

	t.Run("zero value receiver stores the values", func(t *testing.T) {
		var s Set

		s.AddSet(Of(2, 1, 300))

		assertValues(t, &s, 1, 2, 300)
		assertInvariants(t, &s)
	})
}

func TestSet_Remove(t *testing.T) {
	t.Run("reports whether the value was present", func(t *testing.T) {
		s := Of(1, 2, 3)

		if got := s.Remove(2); !got {
			t.Errorf("Remove(2) = %v, want true", got)
		}

		if got := s.Remove(2); got {
			t.Errorf("Remove(2) on absent = %v, want false", got)
		}

		assertValues(t, s, 1, 3)
		assertInvariants(t, s)
	})

	t.Run("absent values report false", func(t *testing.T) {
		s := Of(1, 64)

		for _, v := range []int{-1, math.MinInt, 0, 65, 1 << 20} {
			if got := s.Remove(v); got {
				t.Errorf("Remove(%d) = %v, want false", v, got)
			}
		}

		assertValues(t, s, 1, 64)
		assertInvariants(t, s)
	})

	t.Run("drops trailing empty words", func(t *testing.T) {
		s := Of(1, 64, 1000)
		before := s.Cap()

		s.Remove(1000)

		if got := len(s.data); got != 2 {
			t.Errorf("len(data) after Remove(1000) = %d, want 2", got)
		}

		if got := s.Cap(); got != before {
			t.Errorf("Cap() after Remove(1000) = %d, want %d", got, before)
		}
		assertInvariants(t, s)
	})

	t.Run("keeps words below the largest value", func(t *testing.T) {
		s := Of(1, 64, 1000)

		s.Remove(64)

		if got := len(s.data); got != 16 {
			t.Errorf("len(data) after Remove(64) = %d, want 16", got)
		}
		assertInvariants(t, s)
	})

	t.Run("removing every value empties data", func(t *testing.T) {
		s := Of(1, 64, 1000)

		s.Remove(1000)
		s.Remove(1)
		s.Remove(64)

		if got := len(s.data); got != 0 {
			t.Errorf("len(data) = %d, want 0", got)
		}
		assertValues(t, s)
		assertInvariants(t, s)
	})

	t.Run("usable after trimming", func(t *testing.T) {
		s := Of(1, 64, 1000)
		s.Remove(1000)
		s.Remove(64)

		s.Add(900)

		assertValues(t, s, 1, 900)
		assertInvariants(t, s)
	})
}

func TestSet_RemoveSet(t *testing.T) {
	t.Run("deletes the values of other", func(t *testing.T) {
		s := Of(1, 2, 3, 4, 5)

		s.RemoveSet(Of(4, 9, 2))

		assertValues(t, s, 1, 3, 5)
		assertInvariants(t, s)
	})

	t.Run("other reaches further words", func(t *testing.T) {
		s := Of(1, 64, 100)

		s.RemoveSet(Of(64, 5000))

		assertValues(t, s, 1, 100)
		assertInvariants(t, s)
	})

	t.Run("drops trailing empty words", func(t *testing.T) {
		s := Of(1, 64, 1000)
		before := s.Cap()

		s.RemoveSet(Of(64, 1000))

		assertValues(t, s, 1)
		assertInvariants(t, s)

		if got := s.Cap(); got != before {
			t.Errorf("Cap() = %d, want %d", got, before)
		}
	})

	t.Run("nil other is a no-op", func(t *testing.T) {
		s := Of(1, 2)

		s.RemoveSet(nil)

		assertValues(t, s, 1, 2)
	})

	t.Run("empty receiver is a no-op", func(t *testing.T) {
		s := New()

		s.RemoveSet(Of(1, 2))

		assertValues(t, s)
		assertInvariants(t, s)
	})

	t.Run("self empties the set", func(t *testing.T) {
		s := Of(1, 64, 1000)
		before := s.Cap()

		s.RemoveSet(s)

		assertValues(t, s)
		assertInvariants(t, s)

		if got := s.Cap(); got != before {
			t.Errorf("Cap() after RemoveSet(s) = %d, want %d", got, before)
		}

		s.Add(7)

		assertValues(t, s, 7)
		assertInvariants(t, s)
	})
}

func TestSet_RetainSet(t *testing.T) {
	t.Run("keeps the values present in other", func(t *testing.T) {
		s := Of(5, 1, 4, 2, 3)

		s.RetainSet(Of(3, 4, 5, 9))

		assertValues(t, s, 5, 4, 3)
	})

	t.Run("nil other empties the set", func(t *testing.T) {
		s := Of(1, 2, 3)

		s.RetainSet(nil)

		assertValues(t, s)

		if s.data == nil {
			t.Errorf("data after RetainSet(nil) = nil, want the retained slice")
		}

		s.Add(4)

		assertValues(t, s, 4)
	})

	t.Run("self is a no-op", func(t *testing.T) {
		s := Of(1, 2, 3)

		s.RetainSet(s)

		assertValues(t, s, 1, 2, 3)
	})

	t.Run("shorter other drops the tail", func(t *testing.T) {
		s := Of(1, 2, 64, 1000)
		before := s.Cap()

		s.RetainSet(Of(2, 64))

		assertValues(t, s, 2, 64)
		assertInvariants(t, s)

		if got := s.Cap(); got != before {
			t.Errorf("Cap() = %d, want %d", got, before)
		}
	})

	t.Run("dropped tail is not resurrected", func(t *testing.T) {
		s := Of(1, 900, 1000)

		s.RetainSet(Of(1))
		s.Add(950)

		assertValues(t, s, 1, 950)
		assertInvariants(t, s)
	})
}

func TestSet_RemoveFunc(t *testing.T) {
	t.Run("deletes selected values and counts them", func(t *testing.T) {
		s := fill(10)

		got := s.RemoveFunc(func(v int) bool { return v%3 == 0 })

		if got != 4 {
			t.Errorf("RemoveFunc(multiples of 3) = %d, want 4", got)
		}

		assertValues(t, s, 1, 2, 4, 5, 7, 8)
	})

	t.Run("calls del exactly once per value", func(t *testing.T) {
		s := fill(5)
		calls := make([]int, 5)

		s.RemoveFunc(func(v int) bool {
			calls[v]++

			return v < 2
		})

		want := []int{1, 1, 1, 1, 1}
		if !slices.Equal(calls, want) {
			t.Errorf("del calls = %v, want %v", calls, want)
		}
	})

	t.Run("calls del in ascending order", func(t *testing.T) {
		s := Of(1000, 3, 64, 0, 63, 200)

		var got []int
		s.RemoveFunc(func(v int) bool {
			got = append(got, v)

			return false
		})

		if want := []int{0, 3, 63, 64, 200, 1000}; !slices.Equal(got, want) {
			t.Errorf("del calls = %v, want %v", got, want)
		}
	})

	t.Run("no match returns 0", func(t *testing.T) {
		s := Of(1, 2)

		if got := s.RemoveFunc(func(int) bool { return false }); got != 0 {
			t.Errorf("RemoveFunc(none) = %d, want 0", got)
		}

		assertValues(t, s, 1, 2)
		assertInvariants(t, s)
	})

	t.Run("drops trailing empty words", func(t *testing.T) {
		s := Of(0, 63, 64, 200, 1000)
		before := s.Cap()

		got := s.RemoveFunc(func(v int) bool { return v >= 64 })

		if got != 3 {
			t.Errorf("RemoveFunc(v >= 64) = %d, want 3", got)
		}

		assertValues(t, s, 0, 63)
		assertInvariants(t, s)

		if got := len(s.data); got != 1 {
			t.Errorf("len(data) = %d, want 1", got)
		}

		if got := s.Cap(); got != before {
			t.Errorf("Cap() = %d, want %d", got, before)
		}
	})

	t.Run("removing every value empties data", func(t *testing.T) {
		s := Of(1, 64, 1000)

		if got := s.RemoveFunc(func(int) bool { return true }); got != 3 {
			t.Errorf("RemoveFunc(all) = %d, want 3", got)
		}

		assertValues(t, s)
		assertInvariants(t, s)

		s.Add(5)

		assertValues(t, s, 5)
	})

	t.Run("empty set does not call del", func(t *testing.T) {
		s := Of(1000)
		s.Remove(1000)

		got := s.RemoveFunc(func(int) bool {
			t.Fatal("del called on an empty set")

			return true
		})

		if got != 0 {
			t.Errorf("RemoveFunc() on empty = %d, want 0", got)
		}
	})

	t.Run("panicking del keeps the set consistent", func(t *testing.T) {
		s := Of(1, 2, 64, 65)

		sharedtest.Panics(t, "RemoveFunc(panicking)", "boom", func() {
			s.RemoveFunc(func(v int) bool {
				if v == 64 {
					panic("boom")
				}

				return v == 1 || v == 65
			})
		})

		assertValues(t, s, 2, 64, 65)
		assertInvariants(t, s)
	})

	t.Run("nil predicate panics", func(t *testing.T) {
		s := Of(1)
		sharedtest.Panics(t, "RemoveFunc(nil)", "bitset: nil predicate",
			func() { s.RemoveFunc(nil) })
	})

	t.Run("nil predicate panics on nil receiver", func(t *testing.T) {
		var s *Set
		sharedtest.Panics(t, "nil.RemoveFunc(nil)", "bitset: nil predicate",
			func() { s.RemoveFunc(nil) })
	})
}

func TestSet_Union(t *testing.T) {
	t.Run("returns values of every operand", func(t *testing.T) {
		s := Of(1, 2, 3)

		u := s.Union(Of(4, 2, 1000), Of(64, 1, 7, 4))

		assertValues(t, u, 1, 2, 3, 4, 7, 64, 1000)
		assertValues(t, s, 1, 2, 3)
		assertInvariants(t, u)
	})

	t.Run("receiver reaches further words", func(t *testing.T) {
		s := Of(1, 1000)

		u := s.Union(Of(2, 64))

		assertValues(t, u, 1, 2, 64, 1000)
		assertInvariants(t, u)
	})

	t.Run("nil other is treated as empty", func(t *testing.T) {
		s := Of(2, 1)

		assertValues(t, s.Union(nil), 2, 1)
	})

	t.Run("no others returns an independent copy", func(t *testing.T) {
		s := Of(2, 1)

		u := s.Union()
		u.Add(3)

		assertValues(t, u, 2, 1, 3)
		assertValues(t, s, 2, 1)
	})

	t.Run("self returns a copy", func(t *testing.T) {
		s := Of(1, 64)

		u := s.Union(s)
		u.Remove(1)

		assertValues(t, u, 64)
		assertValues(t, s, 1, 64)
	})
}

func TestSet_Intersection(t *testing.T) {
	t.Run("returns values present in every operand", func(t *testing.T) {
		s := Of(5, 1, 4, 64, 3, 1000)

		got := s.Intersection(Of(3, 4, 64, 9, 1000), Of(4, 64, 3, 1, 1000))

		assertValues(t, got, 3, 4, 64, 1000)
		assertValues(t, s, 5, 1, 4, 64, 3, 1000)
		assertInvariants(t, got)
	})

	t.Run("drops trailing empty words", func(t *testing.T) {
		s := Of(1, 1000)

		got := s.Intersection(Of(1, 2000))

		assertValues(t, got, 1)
		assertInvariants(t, got)
	})

	t.Run("nil other is treated as empty", func(t *testing.T) {
		s := Of(1, 2)

		assertValues(t, s.Intersection(Of(1), nil))
	})

	t.Run("no others returns an independent copy", func(t *testing.T) {
		s := Of(2, 1)

		c := s.Intersection()
		c.Remove(2)

		assertValues(t, c, 1)
		assertValues(t, s, 2, 1)
	})

	t.Run("self returns a copy", func(t *testing.T) {
		s := Of(1, 2)

		assertValues(t, s.Intersection(s), 1, 2)
	})
}

func TestSet_Difference(t *testing.T) {
	t.Run("returns values absent from every other", func(t *testing.T) {
		s := Of(5, 1, 4, 2, 3, 1000)

		got := s.Difference(Of(1), Of(3, 9, 1000))

		assertValues(t, got, 5, 4, 2)
		assertValues(t, s, 5, 1, 4, 2, 3, 1000)
		assertInvariants(t, got)
	})

	t.Run("nil other is treated as empty", func(t *testing.T) {
		s := Of(1, 2)

		assertValues(t, s.Difference(nil), 1, 2)
	})

	t.Run("no others returns an independent copy", func(t *testing.T) {
		s := Of(2, 1)

		c := s.Difference()
		c.Remove(2)

		assertValues(t, c, 1)
		assertValues(t, s, 2, 1)
	})

	t.Run("self returns an empty set", func(t *testing.T) {
		s := Of(1, 2, 1000)

		got := s.Difference(s)

		assertValues(t, got)
		assertInvariants(t, got)
		assertValues(t, s, 1, 2, 1000)
	})
}

func TestSet_SymmetricDifference(t *testing.T) {
	t.Run("returns values in exactly one operand", func(t *testing.T) {
		s := Of(1, 2, 3, 4)
		other := Of(6, 3, 5, 1, 1000)

		got := s.SymmetricDifference(other)

		assertValues(t, got, 2, 4, 6, 5, 1000)
		assertValues(t, s, 1, 2, 3, 4)
		assertValues(t, other, 6, 3, 5, 1, 1000)
		assertInvariants(t, got)
	})

	t.Run("drops trailing empty words", func(t *testing.T) {
		s := Of(1, 1000)

		got := s.SymmetricDifference(Of(1000))

		assertValues(t, got, 1)
		assertInvariants(t, got)
	})

	t.Run("nil other is treated as empty", func(t *testing.T) {
		s := Of(2, 1)

		assertValues(t, s.SymmetricDifference(nil), 2, 1)
	})

	t.Run("self returns an empty set", func(t *testing.T) {
		s := fill(200)

		got := s.SymmetricDifference(s)

		assertValues(t, got)
		assertInvariants(t, got)
	})
}

func TestSet_Equal(t *testing.T) {
	spare := NewWithCap(4096)
	spare.AddN(1, 64)

	trimmed := Of(1, 64, 1000)
	trimmed.Remove(1000)

	tests := []struct {
		name  string
		other *Set
		want  bool
	}{
		{
			name:  "same values",
			other: Of(1, 64),
			want:  true,
		},
		{
			name:  "same values with spare capacity",
			other: spare,
			want:  true,
		},
		{
			name:  "same values after trimming",
			other: trimmed,
			want:  true,
		},
		{
			name:  "superset",
			other: Of(1, 64, 1000),
			want:  false,
		},
		{
			name:  "same size, different values",
			other: Of(1, 65),
			want:  false,
		},
		{
			name:  "nil other is treated as empty",
			other: nil,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Of(1, 64).Equal(tt.other); got != tt.want {
				t.Errorf("Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSet_IsSubset(t *testing.T) {
	tests := []struct {
		name  string
		other *Set
		want  bool
	}{
		{
			name:  "superset",
			other: Of(3, 64, 1, 1000),
			want:  true,
		},
		{
			name:  "equal",
			other: Of(64, 1),
			want:  true,
		},
		{
			name:  "missing a value",
			other: Of(1, 3, 4),
			want:  false,
		},
		{
			name:  "larger but shorter",
			other: Of(1, 2, 3),
			want:  false,
		},
		{
			name:  "smaller",
			other: Of(1),
			want:  false,
		},
		{
			name:  "nil other is treated as empty",
			other: nil,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Of(1, 64).IsSubset(tt.other); got != tt.want {
				t.Errorf("IsSubset() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("empty set is a subset of any set", func(t *testing.T) {
		if got := New().IsSubset(Of(5)); !got {
			t.Errorf("IsSubset() = %v, want true", got)
		}
	})
}

func TestSet_IsSuperset(t *testing.T) {
	tests := []struct {
		name  string
		other *Set
		want  bool
	}{
		{
			name:  "subset",
			other: Of(64),
			want:  true,
		},
		{
			name:  "equal",
			other: Of(64, 1),
			want:  true,
		},
		{
			name:  "missing a value",
			other: Of(1, 3),
			want:  false,
		},
		{
			name:  "larger",
			other: Of(1, 64, 1000),
			want:  false,
		},
		{
			name:  "nil other is treated as empty",
			other: nil,
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Of(1, 64).IsSuperset(tt.other); got != tt.want {
				t.Errorf("IsSuperset() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSet_IsDisjoint(t *testing.T) {
	tests := []struct {
		name  string
		other *Set
		want  bool
	}{
		{
			name:  "no common values",
			other: Of(3, 4, 65, 1000),
			want:  true,
		},
		{
			name:  "overlap in the first word",
			other: Of(2),
			want:  false,
		},
		{
			name:  "overlap in a later word",
			other: Of(5, 64, 1000),
			want:  false,
		},
		{
			name:  "other only beyond the receiver",
			other: Of(1000, 2000),
			want:  true,
		},
		{
			name:  "nil other is treated as empty",
			other: nil,
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Of(2, 64).IsDisjoint(tt.other); got != tt.want {
				t.Errorf("IsDisjoint() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSet_Clear(t *testing.T) {
	s := NewWithCap(4096)
	s.AddN(1, 64, 1000)

	s.Clear()

	assertValues(t, s)
	assertInvariants(t, s)

	if s.data != nil {
		t.Errorf("data after Clear() = %v, want nil", s.data)
	}

	if got := s.Cap(); got != 0 {
		t.Errorf("Cap() after Clear() = %d, want 0", got)
	}

	s.Add(99)

	assertValues(t, s, 99)
}

func TestSet_Reset(t *testing.T) {
	t.Run("keeps the backing array", func(t *testing.T) {
		s := NewWithCap(4096)
		s.AddN(1, 64, 1000)
		before := s.Cap()

		s.Reset()

		assertValues(t, s)
		assertInvariants(t, s)

		if got := s.Cap(); got != before {
			t.Errorf("Cap() after Reset() = %d, want %d", got, before)
		}

		if got := s.Contains(1); got {
			t.Errorf("Contains(1) after Reset() = %v, want false", got)
		}
	})

	t.Run("old values are not resurrected", func(t *testing.T) {
		s := Of(1, 64, 900, 1000)
		s.Reset()

		s.Add(950)

		assertValues(t, s, 950)
		assertInvariants(t, s)
	})

	t.Run("reuses the backing array", func(t *testing.T) {
		s := fill(1024)
		s.Reset()

		allocs := testing.AllocsPerRun(10, func() {
			s.Add(1023)
		})

		if allocs != 0 {
			t.Errorf("Add after Reset allocated %v times, want 0", allocs)
		}
	})
}

func TestSet_Shrink(t *testing.T) {
	t.Run("copies into a right-sized array", func(t *testing.T) {
		s := Of(1, 64, 4000)
		s.Remove(4000)
		old := s.data[:cap(s.data)]

		s.Shrink()

		assertValues(t, s, 1, 64)
		assertInvariants(t, s)

		if got := s.Cap(); got != 2*wordBits {
			t.Errorf("Cap() after Shrink() = %d, want %d", got, 2*wordBits)
		}

		old[0] = ^uint64(0)

		if got := s.Contains(0); got {
			t.Errorf("Contains(0) after Shrink() = %v, "+
				"want false (Shrink must copy)", got)
		}
	})

	t.Run("right-sized array is kept", func(t *testing.T) {
		s := Of(1, 64)
		s.Clip()
		old := &s.data[0]

		s.Shrink()

		if &s.data[0] != old {
			t.Errorf("Shrink() reallocated a right-sized array")
		}
	})

	t.Run("empty set releases the backing array", func(t *testing.T) {
		s := fill(200)
		s.Reset()

		s.Shrink()

		if s.data != nil {
			t.Errorf("data after Shrink() on empty = %v, want nil", s.data)
		}
	})

	t.Run("usable after Shrink", func(t *testing.T) {
		s := Of(1)
		s.Shrink()

		s.Add(2000)

		assertValues(t, s, 1, 2000)
		assertInvariants(t, s)
	})
}

func TestSet_Clip(t *testing.T) {
	t.Run("trims spare capacity without copying", func(t *testing.T) {
		s := NewWithCap(4096)
		s.AddN(1, 64)
		old := &s.data[0]

		s.Clip()

		assertValues(t, s, 1, 64)

		if got := s.Cap(); got != 2*wordBits {
			t.Errorf("Cap() after Clip() = %d, want %d", got, 2*wordBits)
		}

		if &s.data[0] != old {
			t.Errorf("Clip() reallocated the backing array")
		}
	})

	t.Run("usable after Clip", func(t *testing.T) {
		s := Of(1, 64, 1000)
		s.Remove(1000)
		s.Clip()

		s.Add(500)

		assertValues(t, s, 1, 64, 500)
		assertInvariants(t, s)
	})
}

func TestSet_Grow(t *testing.T) {
	t.Run("covers values below n", func(t *testing.T) {
		s := Of(1)

		s.Grow(1000)

		if got := s.Cap(); got < 1000 {
			t.Fatalf("Cap() after Grow(1000) = %d, want >= 1000", got)
		}

		assertValues(t, s, 1)
		assertInvariants(t, s)

		allocs := testing.AllocsPerRun(10, func() {
			s.Add(999)
		})

		if allocs != 0 {
			t.Errorf("Add(999) after Grow(1000) allocated %v times, want 0",
				allocs)
		}
	})

	t.Run("already covered is a no-op", func(t *testing.T) {
		s := NewWithCap(1024)
		s.Add(1)
		before := s.Cap()

		s.Grow(512)

		if got := s.Cap(); got != before {
			t.Errorf("Cap() after Grow(512) = %d, want %d", got, before)
		}
	})

	t.Run("non-positive n is a no-op", func(t *testing.T) {
		s := Of(1)
		before := s.Cap()

		s.Grow(0)
		s.Grow(-10)

		if got := s.Cap(); got != before {
			t.Errorf("Cap() after Grow(<= 0) = %d, want %d", got, before)
		}
	})

	t.Run("zero value receiver", func(t *testing.T) {
		var s Set

		s.Grow(65)

		if got := s.Cap(); got < 128 {
			t.Errorf("Cap() after Grow(65) = %d, want >= 128", got)
		}

		assertValues(t, &s)
		assertInvariants(t, &s)
	})
}

// TestSet_RandomOps runs a seeded random sequence of operations against a map
// model and checks that both agree after every step.
func TestSet_RandomOps(t *testing.T) {
	const span = 300

	r := rand.New(rand.NewPCG(1, 2))
	s := New()
	model := make(map[int]struct{})

	for i := range 5000 {
		v := r.IntN(span)
		_, has := model[v]

		switch op := r.IntN(24); {
		case op < 8:
			if got := s.Add(v); got == has {
				t.Fatalf("step %d: Add(%d) = %v, want %v", i, v, got, !has)
			}

			model[v] = struct{}{}
		case op < 14:
			if got := s.Remove(v); got != has {
				t.Fatalf("step %d: Remove(%d) = %v, want %v",
					i, v, got, has)
			}

			delete(model, v)
		case op < 17:
			if got := s.Contains(v); got != has {
				t.Fatalf("step %d: Contains(%d) = %v, want %v",
					i, v, got, has)
			}
		case op < 18:
			mod := 2 + r.IntN(4)
			want := 0
			for k := range model {
				if k%mod == 0 {
					delete(model, k)
					want++
				}
			}

			got := s.RemoveFunc(func(x int) bool { return x%mod == 0 })
			if got != want {
				t.Fatalf("step %d: RemoveFunc(multiples of %d) = %d, "+
					"want %d", i, mod, got, want)
			}
		case op < 21:
			other := New()
			for range r.IntN(20) {
				other.Add(r.IntN(span))
			}

			switch r.IntN(3) {
			case 0:
				s.AddSet(other)
				for k := range other.All() {
					model[k] = struct{}{}
				}
			case 1:
				s.RemoveSet(other)
				for k := range other.All() {
					delete(model, k)
				}
			default:
				s.RetainSet(other)
				for k := range model {
					if !other.Contains(k) {
						delete(model, k)
					}
				}
			}
		case op < 22:
			checkBounds(t, i, s, model)
		default:
			switch r.IntN(4) {
			case 0:
				s.Clear()
				clear(model)
			case 1:
				s.Reset()
				clear(model)
			case 2:
				s.Shrink()
			default:
				s.Clip()
			}
		}

		if got := s.Len(); got != len(model) {
			t.Fatalf("step %d: Len() = %d, want %d", i, got, len(model))
		}

		assertInvariants(t, s)
		if t.Failed() {
			t.Fatalf("step %d: invariants broken", i)
		}
	}

	assertValues(t, s, slices.Collect(maps.Keys(model))...)
}

// checkBounds fails the test unless Min and Max of s match the model.
func checkBounds(t *testing.T, step int, s *Set, model map[int]struct{}) {
	t.Helper()

	wantMin, wantMax, ok := 0, 0, len(model) > 0
	if ok {
		wantMin = slices.Min(slices.Collect(maps.Keys(model)))
		wantMax = slices.Max(slices.Collect(maps.Keys(model)))
	}

	if got, gotOK := s.Min(); got != wantMin || gotOK != ok {
		t.Fatalf("step %d: Min() = (%d, %v), want (%d, %v)",
			step, got, gotOK, wantMin, ok)
	}

	if got, gotOK := s.Max(); got != wantMax || gotOK != ok {
		t.Fatalf("step %d: Max() = (%d, %v), want (%d, %v)",
			step, got, gotOK, wantMax, ok)
	}
}

// TestSet_NilReceiver checks that every read-only and removal method treats a
// nil receiver as an empty set.
func TestSet_NilReceiver(t *testing.T) {
	var s *Set

	if got := s.Len(); got != 0 {
		t.Errorf("nil.Len() = %d, want 0", got)
	}

	if got := s.Cap(); got != 0 {
		t.Errorf("nil.Cap() = %d, want 0", got)
	}

	if got := s.IsEmpty(); !got {
		t.Errorf("nil.IsEmpty() = %v, want true", got)
	}

	if got := s.Contains(1); got {
		t.Errorf("nil.Contains(1) = %v, want false", got)
	}

	if got := s.ContainsAll(); !got {
		t.Errorf("nil.ContainsAll() = %v, want true", got)
	}

	if got := s.ContainsAll(1); got {
		t.Errorf("nil.ContainsAll(1) = %v, want false", got)
	}

	if got := s.ContainsAny(1); got {
		t.Errorf("nil.ContainsAny(1) = %v, want false", got)
	}

	if got, ok := s.Min(); got != 0 || ok {
		t.Errorf("nil.Min() = (%d, %v), want (0, false)", got, ok)
	}

	if got, ok := s.Max(); got != 0 || ok {
		t.Errorf("nil.Max() = (%d, %v), want (0, false)", got, ok)
	}

	count := 0
	for range s.All() {
		count++
	}

	for range s.Backward() {
		count++
	}

	if count != 0 {
		t.Errorf("nil.All() and nil.Backward() count = %d, want 0", count)
	}

	c := s.Clone()
	if c == nil {
		t.Fatalf("nil.Clone() = nil, want a usable empty set")
	}

	if got := c.Len(); got != 0 {
		t.Errorf("nil.Clone().Len() = %d, want 0", got)
	}

	c.Add(42)

	if got := c.Contains(42); !got {
		t.Errorf("Contains(42) after Add(42) on nil.Clone() = %v, "+
			"want true", got)
	}

	if got := s.Slice(); got != nil {
		t.Errorf("nil.Slice() = %v, want nil", got)
	}

	if got := s.Remove(1); got {
		t.Errorf("nil.Remove(1) = %v, want false", got)
	}

	s.RemoveSet(Of(1))

	if got := s.Len(); got != 0 {
		t.Errorf("nil.Len() after RemoveSet() = %d, want 0", got)
	}

	s.RetainSet(Of(1))

	if got := s.Len(); got != 0 {
		t.Errorf("nil.Len() after RetainSet() = %d, want 0", got)
	}

	if got := s.RemoveFunc(func(int) bool { return true }); got != 0 {
		t.Errorf("nil.RemoveFunc(all) = %d, want 0", got)
	}

	assertValues(t, s.Union(Of(1, 2)), 1, 2)
	assertValues(t, s.Intersection(Of(1)))
	assertValues(t, s.Difference(Of(1)))
	assertValues(t, s.SymmetricDifference(Of(1, 2)), 1, 2)

	if got := s.Equal(New()); !got {
		t.Errorf("nil.Equal(empty) = %v, want true", got)
	}

	if got := s.IsSubset(Of(1)); !got {
		t.Errorf("nil.IsSubset(non-empty) = %v, want true", got)
	}

	if got := s.IsSuperset(Of(1)); got {
		t.Errorf("nil.IsSuperset(non-empty) = %v, want false", got)
	}

	if got := s.IsDisjoint(Of(1)); !got {
		t.Errorf("nil.IsDisjoint(non-empty) = %v, want true", got)
	}
}

// TestSet_NilReceiverMutatorsPanic checks that every method that stores,
// clears, resets, or resizes data panics on a nil receiver.
func TestSet_NilReceiverMutatorsPanic(t *testing.T) {
	tests := []struct {
		name string
		op   string
		call func(*Set)
	}{
		{
			name: "Add",
			op:   "Add",
			call: func(s *Set) { s.Add(1) },
		},
		{
			name: "AddN",
			op:   "AddN",
			call: func(s *Set) { s.AddN(1, 2) },
		},
		{
			name: "AddN with no arguments",
			op:   "AddN",
			call: func(s *Set) { s.AddN() },
		},
		{
			name: "AddSet",
			op:   "AddSet",
			call: func(s *Set) { s.AddSet(Of(1)) },
		},
		{
			name: "AddSet with nil other",
			op:   "AddSet",
			call: func(s *Set) { s.AddSet(nil) },
		},
		{
			name: "Clear",
			op:   "Clear",
			call: func(s *Set) { s.Clear() },
		},
		{
			name: "Reset",
			op:   "Reset",
			call: func(s *Set) { s.Reset() },
		},
		{
			name: "Shrink",
			op:   "Shrink",
			call: func(s *Set) { s.Shrink() },
		},
		{
			name: "Clip",
			op:   "Clip",
			call: func(s *Set) { s.Clip() },
		},
		{
			name: "Grow",
			op:   "Grow",
			call: func(s *Set) { s.Grow(64) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s *Set
			sharedtest.Panics(t, tt.name,
				"bitset: "+tt.op+" on nil receiver",
				func() { tt.call(s) })
		})
	}
}

func TestSet_ZeroValue(t *testing.T) {
	var s Set

	if got := s.Len(); got != 0 {
		t.Errorf("Len() on zero value = %d, want 0", got)
	}

	if got := s.Cap(); got != 0 {
		t.Errorf("Cap() on zero value = %d, want 0", got)
	}

	if got := s.IsEmpty(); !got {
		t.Errorf("IsEmpty() on zero value = %v, want true", got)
	}

	if got := s.Contains(1); got {
		t.Errorf("Contains(1) on zero value = %v, want false", got)
	}

	if got, ok := s.Max(); got != 0 || ok {
		t.Errorf("Max() on zero value = (%d, %v), want (0, false)", got, ok)
	}

	if got := s.Slice(); got != nil {
		t.Errorf("Slice() on zero value = %v, want nil", got)
	}

	if got := s.Remove(1); got {
		t.Errorf("Remove(1) on zero value = %v, want false", got)
	}

	s.Clear()
	s.Reset()
	s.Shrink()
	s.Clip()

	s.AddN(10, 200)

	if got := s.Contains(200); !got {
		t.Errorf("Contains(200) after AddN(10, 200) on zero value = %v, "+
			"want true", got)
	}

	assertValues(t, &s, 10, 200)
	assertInvariants(t, &s)
}
