package orderedset

import (
	"iter"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Nergous/structures/internal/shared/sharedtest"
)

// assertOrder verifies that s holds exactly want, in this order.
func assertOrder[T comparable](t *testing.T, s *Set[T], want ...T) {
	t.Helper()

	if got := s.Len(); got != len(want) {
		t.Fatalf("Len() = %d, want %d (Slice() = %v, want %v)", got, len(want), s.Slice(), want)
	}

	if got := s.Slice(); !slices.Equal(got, want) {
		t.Fatalf("Slice() = %v, want %v", got, want)
	}

	if got := slices.Collect(s.All()); !slices.Equal(got, want) {
		t.Fatalf("All() = %v, want %v", got, want)
	}
}

// intRange returns the values lo, lo+1, ..., hi-1.
func intRange(lo, hi int) []int {
	out := make([]int, 0, max(hi-lo, 0))
	for v := lo; v < hi; v++ {
		out = append(out, v)
	}

	return out
}

// fillSet returns a set holding 0, 1, ..., n-1 in this order.
func fillSet(n int) *Set[int] {
	s := New[int]()
	for v := range n {
		s.Add(v)
	}

	return s
}

func TestSet_New(t *testing.T) {
	s := New[int]()
	if s == nil {
		t.Fatalf("New() = nil, want set")
	}

	if got := s.IsEmpty(); !got {
		t.Errorf("IsEmpty() = %v, want true", got)
	}

	s.Add(1)

	assertOrder(t, s, 1)
}

func TestSet_NewWithCap(t *testing.T) {
	tests := []struct {
		name    string
		n       int
		wantCap int
	}{
		{
			name:    "zero",
			n:       0,
			wantCap: 0,
		},
		{
			name:    "positive",
			n:       16,
			wantCap: 16,
		},
		{
			name:    "negative clamps to zero",
			n:       -3,
			wantCap: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewWithCap[int](tt.n)

			if got := s.Len(); got != 0 {
				t.Errorf("Len() = %d, want 0", got)
			}

			if got := s.Cap(); got < tt.wantCap || (tt.wantCap == 0 && got != 0) {
				t.Errorf("Cap() = %d, want %d", got, tt.wantCap)
			}

			if got := s.Slice(); got != nil {
				t.Errorf("Slice() = %v, want nil", got)
			}

			s.Add(1)

			assertOrder(t, s, 1)
		})
	}
}

func TestSet_OfFrom(t *testing.T) {
	t.Run("Of keeps order of first occurrence", func(t *testing.T) {
		assertOrder(t, Of(3, 1, 3, 2, 1), 3, 1, 2)
	})

	t.Run("From keeps order of first occurrence", func(t *testing.T) {
		assertOrder(t, From([]string{"go", "set", "go", "map"}), "go", "set", "map")
	})

	t.Run("From does not retain the input", func(t *testing.T) {
		in := []int{1, 2, 3}
		s := From(in)
		in[0], in[1], in[2] = 7, 8, 9

		assertOrder(t, s, 1, 2, 3)
	})

	t.Run("empty inputs return usable empty sets", func(t *testing.T) {
		tests := []struct {
			name string
			s    *Set[int]
		}{
			{
				name: "Of()",
				s:    Of[int](),
			},
			{
				name: "From(nil)",
				s:    From[int](nil),
			},
			{
				name: "From(empty)",
				s:    From([]int{}),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if tt.s == nil {
					t.Fatalf("%s = nil, want empty set", tt.name)
				}

				if got := tt.s.IsEmpty(); !got {
					t.Errorf("%s.IsEmpty() = %v, want true", tt.name, got)
				}

				tt.s.Add(1)

				assertOrder(t, tt.s, 1)
			})
		}
	})
}

func TestSet_LenCapIsEmpty(t *testing.T) {
	s := New[int]()

	if got := s.IsEmpty(); !got {
		t.Errorf("IsEmpty() on empty = %v, want true", got)
	}

	s.AddN(intRange(0, 20)...)

	if got := s.Len(); got != 20 {
		t.Errorf("Len() = %d, want 20", got)
	}

	if got := s.IsEmpty(); got {
		t.Errorf("IsEmpty() = %v, want false", got)
	}

	before := s.Cap()
	if before < 20 {
		t.Fatalf("Cap() = %d, want >= 20", before)
	}

	s.Remove(5) // middle removals leave holes that Cap still counts
	s.Remove(6)

	if got := s.Len(); got != 18 {
		t.Errorf("Len() after removals = %d, want 18", got)
	}

	if got := s.Cap(); got != before {
		t.Errorf("Cap() after removals = %d, want %d (holes are counted)", got, before)
	}
}

func TestSet_Contains(t *testing.T) {
	s := Of(1, 2, 3)
	var nilSet *Set[int]

	tests := []struct {
		name string
		got  bool
		want bool
	}{
		{
			name: "Contains(2)",
			got:  s.Contains(2),
			want: true,
		},
		{
			name: "Contains(4)",
			got:  s.Contains(4),
			want: false,
		},
		{
			name: "ContainsAll()",
			got:  s.ContainsAll(),
			want: true,
		},
		{
			name: "nil.ContainsAll()",
			got:  nilSet.ContainsAll(),
			want: true,
		},
		{
			name: "ContainsAll(1, 3)",
			got:  s.ContainsAll(1, 3),
			want: true,
		},
		{
			name: "ContainsAll(1, 4)",
			got:  s.ContainsAll(1, 4),
			want: false,
		},
		{
			name: "ContainsAny()",
			got:  s.ContainsAny(),
			want: false,
		},
		{
			name: "ContainsAny(4, 3)",
			got:  s.ContainsAny(4, 3),
			want: true,
		},
		{
			name: "ContainsAny(4, 5)",
			got:  s.ContainsAny(4, 5),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestSet_All(t *testing.T) {
	s := Of(3, 1, 4, 2)
	s.Remove(4)

	if got, want := slices.Collect(s.All()), []int{3, 1, 2}; !slices.Equal(got, want) {
		t.Errorf("All() = %v, want %v", got, want)
	}

	var seen []int
	for v := range s.All() {
		seen = append(seen, v)
		break
	}

	if want := []int{3}; !slices.Equal(seen, want) {
		t.Errorf("All() with early break = %v, want %v", seen, want)
	}
}

func TestSet_Backward(t *testing.T) {
	s := Of(3, 1, 4, 2)
	s.Remove(4)

	var seq iter.Seq[int] = s.Backward()

	if got, want := slices.Collect(seq), []int{2, 1, 3}; !slices.Equal(got, want) {
		t.Errorf("Backward() = %v, want %v", got, want)
	}

	var seen []int
	for v := range s.Backward() {
		seen = append(seen, v)
		break
	}

	if want := []int{2}; !slices.Equal(seen, want) {
		t.Errorf("Backward() with early break = %v, want %v", seen, want)
	}
}

func TestSet_Slice(t *testing.T) {
	s := Of(3, 1, 2)

	got := s.Slice()
	if want := []int{3, 1, 2}; !slices.Equal(got, want) {
		t.Fatalf("Slice() = %v, want %v", got, want)
	}

	got[0] = 99

	if s.Contains(99) {
		t.Errorf("Contains(99) after mutating Slice() result = true, want false (copy must be independent)")
	}

	assertOrder(t, s, 3, 1, 2)

	if got := New[int]().Slice(); got != nil {
		t.Errorf("Slice() on empty = %v, want nil", got)
	}

	s.RemoveFunc(func(int) bool { return true })

	if got := s.Slice(); got != nil {
		t.Errorf("Slice() after removing all = %v, want nil", got)
	}

	var nilSet *Set[int]

	if got := nilSet.Slice(); got != nil {
		t.Errorf("nil.Slice() = %v, want nil", got)
	}
}

func TestSet_Clone(t *testing.T) {
	t.Run("is independent of the original", func(t *testing.T) {
		s := fillSet(40)
		for v := range 10 {
			s.Remove(v)
		}

		c := s.Clone()

		assertOrder(t, c, intRange(10, 40)...)

		if got, want := c.Cap(), c.Len(); got != want {
			t.Errorf("Cap() of clone = %d, want Len() = %d (no holes)", got, want)
		}

		c.Add(1000)
		c.Remove(10)
		c.MoveToBack(11)

		assertOrder(t, s, intRange(10, 40)...)
	})

	t.Run("nil clone returns usable empty set", func(t *testing.T) {
		var s *Set[int]

		c := s.Clone()
		if c == nil {
			t.Fatalf("nil.Clone() = nil, want usable empty set")
		}

		if got := c.IsEmpty(); !got {
			t.Errorf("nil.Clone().IsEmpty() = %v, want true", got)
		}

		c.Add(1)

		assertOrder(t, c, 1)
	})
}

func TestSet_Add(t *testing.T) {
	s := Of(1, 2, 3)

	if got := s.Add(2); got {
		t.Errorf("Add(2) on existing = %v, want false", got)
	}

	assertOrder(t, s, 1, 2, 3) // re-adding keeps the original position

	if got := s.Add(4); !got {
		t.Errorf("Add(4) = %v, want true", got)
	}

	s.Remove(1)

	if got := s.Add(1); !got {
		t.Errorf("Add(1) after Remove(1) = %v, want true", got)
	}

	assertOrder(t, s, 2, 3, 4, 1)
}

func TestSet_AddN(t *testing.T) {
	s := New[int]()
	s.AddN(3, 1, 3, 2)

	assertOrder(t, s, 3, 1, 2)

	s.AddN()

	assertOrder(t, s, 3, 1, 2)

	s.AddN(1, 4)

	assertOrder(t, s, 3, 1, 2, 4)
}

func TestSet_AddSet(t *testing.T) {
	t.Run("appends new values in order of other", func(t *testing.T) {
		s := Of(1, 2, 3)
		other := Of(5, 2, 4, 1)
		s.AddSet(other)

		assertOrder(t, s, 1, 2, 3, 5, 4)
		assertOrder(t, other, 5, 2, 4, 1)
	})

	t.Run("nil other is a no-op", func(t *testing.T) {
		s := Of(1, 2)
		s.AddSet(nil)

		assertOrder(t, s, 1, 2)
	})

	t.Run("self is a no-op", func(t *testing.T) {
		s := Of(1, 2, 3)
		s.AddSet(s)

		assertOrder(t, s, 1, 2, 3)
	})
}

func TestSet_Remove(t *testing.T) {
	s := Of(1, 2, 3)

	if got := s.Remove(2); !got {
		t.Errorf("Remove(2) = %v, want true", got)
	}

	if got := s.Remove(2); got {
		t.Errorf("Remove(2) on absent = %v, want false", got)
	}

	assertOrder(t, s, 1, 3)

	s.Remove(1)
	s.Remove(3)

	assertOrder(t, s)

	s.Add(4)

	assertOrder(t, s, 4)
}

func TestSet_RemoveSet(t *testing.T) {
	t.Run("removes values of other", func(t *testing.T) {
		s := Of(1, 2, 3, 4, 5)
		s.RemoveSet(Of(4, 9, 2))

		assertOrder(t, s, 1, 3, 5)
	})

	t.Run("nil other is a no-op", func(t *testing.T) {
		s := Of(1, 2)
		s.RemoveSet(nil)

		assertOrder(t, s, 1, 2)
	})

	t.Run("self empties the set", func(t *testing.T) {
		s := fillSet(32)
		s.RemoveSet(s)

		assertOrder(t, s)

		s.Add(7)

		assertOrder(t, s, 7)
	})

	t.Run("nil receiver is a no-op", func(t *testing.T) {
		var s *Set[int]
		s.RemoveSet(Of(1))

		if got := s.Len(); got != 0 {
			t.Errorf("nil.Len() after RemoveSet = %d, want 0", got)
		}
	})
}

func TestSet_RetainAll(t *testing.T) {
	t.Run("keeps order", func(t *testing.T) {
		s := Of(5, 1, 4, 2, 3)
		s.RetainAll(Of(3, 4, 5, 9))

		assertOrder(t, s, 5, 4, 3)
	})

	// Regression: removing more than half of 32+ values triggers compaction
	// in the middle of the pass.
	t.Run("removes more than half", func(t *testing.T) {
		s := fillSet(32)
		s.RetainAll(Of(31))

		assertOrder(t, s, 31)

		s = fillSet(40)
		s.RetainAll(Of(39, 0, 20))

		assertOrder(t, s, 0, 20, 39)

		s.Add(100)

		assertOrder(t, s, 0, 20, 39, 100)
	})

	t.Run("self is a no-op", func(t *testing.T) {
		s := Of(1, 2, 3)
		s.RetainAll(s)

		assertOrder(t, s, 1, 2, 3)
	})

	t.Run("nil other empties the set", func(t *testing.T) {
		s := Of(1, 2, 3)
		s.RetainAll(nil)

		assertOrder(t, s)

		s.Add(4)

		assertOrder(t, s, 4)
	})

	t.Run("nil receiver is a no-op", func(t *testing.T) {
		var s *Set[int]
		s.RetainAll(Of(1))

		if got := s.Len(); got != 0 {
			t.Errorf("nil.Len() after RetainAll = %d, want 0", got)
		}
	})
}

func TestSet_Union(t *testing.T) {
	t.Run("order is s then new values of each other", func(t *testing.T) {
		s := Of(1, 2, 3)
		u := s.Union(Of(4, 2, 5), nil, Of(6, 1, 7, 4))

		assertOrder(t, u, 1, 2, 3, 4, 5, 6, 7)
		assertOrder(t, s, 1, 2, 3)

		u.Add(100)

		if s.Contains(100) {
			t.Errorf("s.Contains(100) after adding to union = true, want false (result must be independent)")
		}
	})

	t.Run("no others returns a copy", func(t *testing.T) {
		s := Of(2, 1)
		u := s.Union()

		assertOrder(t, u, 2, 1)

		u.Remove(2)

		assertOrder(t, s, 2, 1)
	})

	t.Run("nil receiver", func(t *testing.T) {
		var s *Set[int]

		assertOrder(t, s.Union(Of(2, 1), nil), 2, 1)
		assertOrder(t, s.Union())
	})
}

func TestSet_Intersection(t *testing.T) {
	t.Run("order of s", func(t *testing.T) {
		s := Of(5, 1, 4, 2, 3)

		assertOrder(t, s.Intersection(Of(3, 4, 5, 9), Of(4, 5, 3, 1)), 5, 4, 3)
	})

	t.Run("no others returns a copy", func(t *testing.T) {
		s := Of(2, 1)
		c := s.Intersection()

		assertOrder(t, c, 2, 1)

		c.Remove(2)

		assertOrder(t, s, 2, 1)
	})

	t.Run("nil operand is an empty set", func(t *testing.T) {
		var nilSet *Set[int]

		assertOrder(t, Of(1, 2).Intersection(Of(1), nil))
		assertOrder(t, nilSet.Intersection(Of(1)))
	})

	// Regression: dropping more than half of 32+ values must not lose or
	// reorder the survivors.
	t.Run("removes more than half", func(t *testing.T) {
		s := fillSet(32)

		assertOrder(t, s.Intersection(nil))
		assertOrder(t, s.Intersection(Of(31)), 31)
		assertOrder(t, s.Intersection(Of(31, 0, 16)), 0, 16, 31)
		assertOrder(t, s, intRange(0, 32)...)
	})
}

func TestSet_Difference(t *testing.T) {
	t.Run("order of s", func(t *testing.T) {
		s := Of(5, 1, 4, 2, 3)

		assertOrder(t, s.Difference(Of(1), nil, Of(3, 9)), 5, 4, 2)
	})

	t.Run("no others returns a copy", func(t *testing.T) {
		s := Of(2, 1)
		c := s.Difference()

		assertOrder(t, c, 2, 1)

		c.Remove(2)

		assertOrder(t, s, 2, 1)
	})

	t.Run("nil operands are empty sets", func(t *testing.T) {
		var nilSet *Set[int]

		assertOrder(t, Of(1, 2).Difference(nil), 1, 2)
		assertOrder(t, nilSet.Difference(Of(1)))
	})

	// Regression: dropping more than half of 32+ values must not lose or
	// reorder the survivors.
	t.Run("removes more than half", func(t *testing.T) {
		s := fillSet(32)

		assertOrder(t, s.Difference(s))
		assertOrder(t, s.Difference(From(intRange(0, 31))), 31)
		assertOrder(t, s, intRange(0, 32)...)
	})
}

func TestSet_SymmetricDifference(t *testing.T) {
	var nilSet *Set[int]

	tests := []struct {
		name string
		s    *Set[int]
		o    *Set[int]
		want []int
	}{
		{
			name: "only in s then only in other",
			s:    Of(1, 2, 3, 4),
			o:    Of(6, 3, 5, 1),
			want: []int{2, 4, 6, 5},
		},
		{
			name: "nil other",
			s:    Of(2, 1),
			o:    nil,
			want: []int{2, 1},
		},
		{
			name: "nil receiver",
			s:    nilSet,
			o:    Of(2, 1),
			want: []int{2, 1},
		},
		{
			name: "both nil",
			s:    nilSet,
			o:    nil,
			want: nil,
		},
		{
			name: "equal sets",
			s:    fillSet(32),
			o:    fillSet(32),
			want: nil,
		},
		{
			name: "large overlap",
			s:    fillSet(32),
			o:    fillSet(34),
			want: []int{32, 33},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertOrder(t, tt.s.SymmetricDifference(tt.o), tt.want...)
		})
	}

	t.Run("with itself", func(t *testing.T) {
		s := fillSet(32)

		assertOrder(t, s.SymmetricDifference(s))
		assertOrder(t, s, intRange(0, 32)...)
	})
}

// relationCase is one row of the Equal, IsSubset, IsSuperset, and IsDisjoint
// tables.
type relationCase struct {
	name string
	a, b *Set[int]
	want bool
}

func TestSet_Equal(t *testing.T) {
	tests := []relationCase{
		{name: "same order", a: Of(1, 2), b: Of(1, 2), want: true},
		{name: "different order", a: Of(1, 2), b: Of(2, 1), want: true},
		{name: "different length", a: Of(1, 2), b: Of(1, 2, 3), want: false},
		{name: "different values", a: Of(1, 2), b: Of(1, 3), want: false},
		{name: "nil and nil", a: nil, b: nil, want: true},
		{name: "nil and empty", a: nil, b: New[int](), want: true},
		{name: "nil and non-empty", a: nil, b: Of(1), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Equal(tt.b); got != tt.want {
				t.Errorf("a.Equal(b) = %v, want %v", got, tt.want)
			}

			if got := tt.b.Equal(tt.a); got != tt.want {
				t.Errorf("b.Equal(a) = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSet_IsSubset(t *testing.T) {
	tests := []relationCase{
		{name: "proper subset", a: Of(1, 2), b: Of(3, 2, 1), want: true},
		{name: "equal sets", a: Of(1, 2), b: Of(2, 1), want: true},
		{name: "not a subset", a: Of(1, 4), b: Of(1, 2, 3), want: false},
		{name: "larger", a: Of(1, 2, 3), b: Of(1, 2), want: false},
		{name: "nil of non-empty", a: nil, b: Of(1), want: true},
		{name: "nil of nil", a: nil, b: nil, want: true},
		{name: "empty of nil", a: New[int](), b: nil, want: true},
		{name: "non-empty of nil", a: Of(1), b: nil, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.IsSubset(tt.b); got != tt.want {
				t.Errorf("a.IsSubset(b) = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSet_IsSuperset(t *testing.T) {
	tests := []relationCase{
		{name: "proper superset", a: Of(3, 2, 1), b: Of(1, 2), want: true},
		{name: "equal sets", a: Of(2, 1), b: Of(1, 2), want: true},
		{name: "not a superset", a: Of(1, 2, 3), b: Of(1, 4), want: false},
		{name: "smaller", a: Of(1, 2), b: Of(1, 2, 3), want: false},
		{name: "non-empty of nil", a: Of(1), b: nil, want: true},
		{name: "nil of nil", a: nil, b: nil, want: true},
		{name: "nil of non-empty", a: nil, b: Of(1), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.IsSuperset(tt.b); got != tt.want {
				t.Errorf("a.IsSuperset(b) = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSet_IsDisjoint(t *testing.T) {
	tests := []relationCase{
		{name: "disjoint", a: Of(1, 2), b: Of(3, 4), want: true},
		{name: "overlap", a: Of(1, 2), b: Of(2, 3), want: false},
		{name: "small overlaps large", a: Of(1, 2, 3, 4), b: Of(4), want: false},
		{name: "nil and non-empty", a: nil, b: Of(1), want: true},
		{name: "non-empty and nil", a: Of(1), b: nil, want: true},
		{name: "nil and nil", a: nil, b: nil, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.IsDisjoint(tt.b); got != tt.want {
				t.Errorf("a.IsDisjoint(b) = %v, want %v", got, tt.want)
			}

			if got := tt.b.IsDisjoint(tt.a); got != tt.want {
				t.Errorf("b.IsDisjoint(a) = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSet_FirstLast(t *testing.T) {
	s := New[int]()

	if got, ok := s.First(); ok || got != 0 {
		t.Errorf("First() on empty = (%d, %v), want (0, false)", got, ok)
	}

	if got, ok := s.Last(); ok || got != 0 {
		t.Errorf("Last() on empty = (%d, %v), want (0, false)", got, ok)
	}

	s.AddN(1, 2, 3)

	if got, ok := s.First(); !ok || got != 1 {
		t.Errorf("First() = (%d, %v), want (1, true)", got, ok)
	}

	if got, ok := s.Last(); !ok || got != 3 {
		t.Errorf("Last() = (%d, %v), want (3, true)", got, ok)
	}

	assertOrder(t, s, 1, 2, 3) // First and Last do not remove

	s.Remove(1)
	s.Remove(3)

	if got, _ := s.First(); got != 2 {
		t.Errorf("First() after removing head = %d, want 2", got)
	}

	if got, _ := s.Last(); got != 2 {
		t.Errorf("Last() after removing tail = %d, want 2", got)
	}
}

func TestSet_PopFirstPopLast(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		s := Of(1, 2, 3, 4)

		if got, ok := s.PopFirst(); !ok || got != 1 {
			t.Errorf("PopFirst() = (%d, %v), want (1, true)", got, ok)
		}

		if got, ok := s.PopLast(); !ok || got != 4 {
			t.Errorf("PopLast() = (%d, %v), want (4, true)", got, ok)
		}

		assertOrder(t, s, 2, 3)

		s.PopFirst()
		s.PopLast()

		if got, ok := s.PopFirst(); ok || got != 0 {
			t.Errorf("PopFirst() on empty = (%d, %v), want (0, false)", got, ok)
		}

		if got, ok := s.PopLast(); ok || got != 0 {
			t.Errorf("PopLast() on empty = (%d, %v), want (0, false)", got, ok)
		}

		s.Add(5)

		assertOrder(t, s, 5)
	})

	t.Run("PopFirst through compaction", func(t *testing.T) {
		s := fillSet(64)
		for want := range 40 {
			if got, ok := s.PopFirst(); !ok || got != want {
				t.Fatalf("PopFirst() #%d = (%d, %v), want (%d, true)", want, got, ok, want)
			}
		}

		assertOrder(t, s, intRange(40, 64)...)
	})

	t.Run("PopLast through compaction", func(t *testing.T) {
		s := fillSet(64)
		s.Remove(1) // leave holes near the head
		s.Remove(2)
		for i := range 40 {
			want := 63 - i
			if got, ok := s.PopLast(); !ok || got != want {
				t.Fatalf("PopLast() #%d = (%d, %v), want (%d, true)", i, got, ok, want)
			}
		}

		assertOrder(t, s, append([]int{0}, intRange(3, 24)...)...)
	})

	t.Run("alternating through compaction", func(t *testing.T) {
		s := fillSet(64)
		for range 20 {
			s.PopFirst()
			s.PopLast()
		}

		assertOrder(t, s, intRange(20, 44)...)
	})
}

func TestSet_MoveToBack(t *testing.T) {
	s := Of(1, 2, 3, 4)

	if got := s.MoveToBack(2); !got {
		t.Errorf("MoveToBack(2) = %v, want true", got)
	}

	if got, _ := s.Last(); got != 2 {
		t.Errorf("Last() after MoveToBack(2) = %d, want 2", got)
	}

	assertOrder(t, s, 1, 3, 4, 2)

	if got := s.MoveToBack(99); got {
		t.Errorf("MoveToBack(99) on absent = %v, want false", got)
	}

	if s.Contains(99) {
		t.Errorf("Contains(99) after MoveToBack(99) = true, want false (must not add)")
	}

	assertOrder(t, s, 1, 3, 4, 2)

	var nilSet *Set[int]

	if got := nilSet.MoveToBack(1); got {
		t.Errorf("nil.MoveToBack(1) = %v, want false", got)
	}
}

func TestSet_RemoveFunc(t *testing.T) {
	s := fillSet(10)

	var seen []int
	got := s.RemoveFunc(func(v int) bool {
		seen = append(seen, v)
		return v%3 == 0
	})

	if got != 4 {
		t.Errorf("RemoveFunc(multiples of 3) = %d, want 4", got)
	}

	if want := intRange(0, 10); !slices.Equal(seen, want) {
		t.Errorf("predicate calls = %v, want %v (once per value, oldest to newest)", seen, want)
	}

	assertOrder(t, s, 1, 2, 4, 5, 7, 8)

	if got := s.RemoveFunc(func(int) bool { return false }); got != 0 {
		t.Errorf("RemoveFunc(none) = %d, want 0", got)
	}

	var nilSet *Set[int]

	if got := nilSet.RemoveFunc(func(int) bool { return true }); got != 0 {
		t.Errorf("nil.RemoveFunc(all) = %d, want 0", got)
	}

	sharedtest.Panics(t, "RemoveFunc(nil)", "orderedset: nil predicate", func() { s.RemoveFunc(nil) })
	sharedtest.Panics(t, "nil.RemoveFunc(nil)", "orderedset: nil predicate", func() { nilSet.RemoveFunc(nil) })
}

func TestSet_GrowClipShrink(t *testing.T) {
	t.Run("Grow and Clip", func(t *testing.T) {
		s := New[int]()
		s.Grow(50)

		if got := s.Cap(); got < 50 {
			t.Errorf("Cap() after Grow(50) = %d, want >= 50", got)
		}

		s.Grow(0)
		s.Grow(-1)
		s.AddN(1, 2)
		s.Clip()

		if got := s.Cap(); got != 2 {
			t.Errorf("Cap() after Clip() = %d, want 2", got)
		}

		s.Add(3)

		assertOrder(t, s, 1, 2, 3)
	})

	t.Run("Shrink after removals", func(t *testing.T) {
		s := NewWithCap[int](100)
		s.AddN(intRange(1, 11)...)
		s.Remove(1)
		s.Remove(2)
		s.Remove(5)
		s.Shrink()

		if got, want := s.Cap(), s.Len(); got != want {
			t.Errorf("Cap() after Shrink() = %d, want Len() = %d", got, want)
		}

		assertOrder(t, s, 3, 4, 6, 7, 8, 9, 10)

		s.Shrink() // already tight
		s.Add(99)

		assertOrder(t, s, 3, 4, 6, 7, 8, 9, 10, 99)
	})
}

func TestSet_ClearReset(t *testing.T) {
	s := NewWithCap[int](8)
	s.AddN(1, 2, 3)
	s.Remove(2)

	s.Reset()

	assertOrder(t, s)

	if got := s.Cap(); got != 8 {
		t.Errorf("Cap() after Reset() = %d, want 8", got)
	}

	if s.Contains(1) {
		t.Errorf("Contains(1) after Reset() = true, want false")
	}

	s.AddN(4, 5)

	assertOrder(t, s, 4, 5)

	s.Clear()

	assertOrder(t, s)

	if got := s.Cap(); got != 0 {
		t.Errorf("Cap() after Clear() = %d, want 0", got)
	}

	s.AddN(6, 7)

	assertOrder(t, s, 6, 7)
}

func TestSet_ZeroValue(t *testing.T) {
	var s Set[int]

	if got := s.IsEmpty(); !got {
		t.Errorf("IsEmpty() on zero value = %v, want true", got)
	}

	if got := s.Contains(1); got {
		t.Errorf("Contains(1) on zero value = %v, want false", got)
	}

	if _, ok := s.First(); ok {
		t.Errorf("First() on zero value: ok = %v, want false", ok)
	}

	if _, ok := s.PopFirst(); ok {
		t.Errorf("PopFirst() on zero value: ok = %v, want false", ok)
	}

	if got := s.Remove(1); got {
		t.Errorf("Remove(1) on zero value = %v, want false", got)
	}

	assertOrder(t, &s)

	if got := s.Add(10); !got {
		t.Errorf("Add(10) on zero value = %v, want true", got)
	}

	s.AddN(20, 30)
	s.Remove(20)

	assertOrder(t, &s, 10, 30)

	if got, want := slices.Collect(s.Backward()), []int{30, 10}; !slices.Equal(got, want) {
		t.Errorf("Backward() on zero value = %v, want %v", got, want)
	}

	s.Clear()
	s.Add(1)
	s.Reset()

	assertOrder(t, &s)

	var z Set[int]
	z.Reset()
	z.Clip()
	z.Shrink()
	z.Grow(4)

	if got := z.Cap(); got < 4 {
		t.Errorf("Cap() after Grow(4) on zero value = %d, want >= 4", got)
	}

	z.AddN(1, 2)

	assertOrder(t, &z, 1, 2)
}

func TestSet_NilReceiver(t *testing.T) {
	var s *Set[int]

	tests := []struct {
		name string
		got  func() any
		want any
	}{
		{name: "Len", got: func() any { return s.Len() }, want: 0},
		{name: "Cap", got: func() any { return s.Cap() }, want: 0},
		{name: "IsEmpty", got: func() any { return s.IsEmpty() }, want: true},
		{name: "Contains", got: func() any { return s.Contains(1) }, want: false},
		{name: "ContainsAll(1)", got: func() any { return s.ContainsAll(1) }, want: false},
		{name: "ContainsAny(1)", got: func() any { return s.ContainsAny(1) }, want: false},
		{name: "All count", got: func() any { return len(slices.Collect(s.All())) }, want: 0},
		{name: "Backward count", got: func() any { return len(slices.Collect(s.Backward())) }, want: 0},
		{name: "Slice is nil", got: func() any { return s.Slice() == nil }, want: true},
		{name: "Remove", got: func() any { return s.Remove(1) }, want: false},
		{name: "First ok", got: func() any { _, ok := s.First(); return ok }, want: false},
		{name: "Last ok", got: func() any { _, ok := s.Last(); return ok }, want: false},
		{name: "PopFirst ok", got: func() any { _, ok := s.PopFirst(); return ok }, want: false},
		{name: "PopLast ok", got: func() any { _, ok := s.PopLast(); return ok }, want: false},
		{name: "MoveToBack", got: func() any { return s.MoveToBack(1) }, want: false},
		{name: "RemoveFunc", got: func() any { return s.RemoveFunc(func(int) bool { return true }) }, want: 0},
		{name: "Clone is empty", got: func() any { return s.Clone().IsEmpty() }, want: true},
		{name: "Union is empty", got: func() any { return s.Union().IsEmpty() }, want: true},
		{name: "Intersection is empty", got: func() any { return s.Intersection().IsEmpty() }, want: true},
		{name: "Difference is empty", got: func() any { return s.Difference().IsEmpty() }, want: true},
		{name: "SymmetricDifference is empty", got: func() any { return s.SymmetricDifference(nil).IsEmpty() }, want: true},
		{name: "Equal(empty)", got: func() any { return s.Equal(New[int]()) }, want: true},
		{name: "IsSubset(non-empty)", got: func() any { return s.IsSubset(Of(1)) }, want: true},
		{name: "IsSuperset(non-empty)", got: func() any { return s.IsSuperset(Of(1)) }, want: false},
		{name: "IsDisjoint(non-empty)", got: func() any { return s.IsDisjoint(Of(1)) }, want: true},
		{name: "RemoveSet no-op", got: func() any { s.RemoveSet(Of(1)); return s.Len() }, want: 0},
		{name: "RetainAll no-op", got: func() any { s.RetainAll(Of(1)); return s.Len() }, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.got(); got != tt.want {
				t.Errorf("nil.%s = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestSet_NilReceiverMutatorsPanic(t *testing.T) {
	tests := []struct {
		name string
		op   string
		call func(s *Set[int])
	}{
		{
			name: "Add",
			op:   "Add",
			call: func(s *Set[int]) { s.Add(1) },
		},
		{
			name: "AddN",
			op:   "AddN",
			call: func(s *Set[int]) { s.AddN(1, 2) },
		},
		{
			name: "AddN with no arguments",
			op:   "AddN",
			call: func(s *Set[int]) { s.AddN() },
		},
		{
			name: "AddSet",
			op:   "AddSet",
			call: func(s *Set[int]) { s.AddSet(Of(1)) },
		},
		{
			name: "Clear",
			op:   "Clear",
			call: func(s *Set[int]) { s.Clear() },
		},
		{
			name: "Reset",
			op:   "Reset",
			call: func(s *Set[int]) { s.Reset() },
		},
		{
			name: "Grow",
			op:   "Grow",
			call: func(s *Set[int]) { s.Grow(1) },
		},
		{
			name: "Clip",
			op:   "Clip",
			call: func(s *Set[int]) { s.Clip() },
		},
		{
			name: "Shrink",
			op:   "Shrink",
			call: func(s *Set[int]) { s.Shrink() },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s *Set[int]
			sharedtest.Panics(t, tt.name, "orderedset: "+tt.op+" on nil receiver", func() { tt.call(s) })
		})
	}
}

// TestSet_RandomOps runs a random mix of operations against a slice model and
// checks the order after every step.
func TestSet_RandomOps(t *testing.T) {
	r := rand.New(rand.NewPCG(31, 32))
	s := New[int]()
	var order []int

	del := func(v int) {
		order = slices.DeleteFunc(order, func(x int) bool { return x == v })
	}

	for i := range 5000 {
		v := r.IntN(80)
		switch op := r.IntN(14); {
		case op < 5:
			has := slices.Contains(order, v)
			if got := s.Add(v); got == has {
				t.Fatalf("Add(%d) #%d = %v, want %v", v, i, got, !has)
			}

			if !has {
				order = append(order, v)
			}
		case op < 8:
			has := slices.Contains(order, v)
			if got := s.Remove(v); got != has {
				t.Fatalf("Remove(%d) #%d = %v, want %v", v, i, got, has)
			}

			del(v)
		case op < 9:
			got, ok := s.PopFirst()
			if ok != (len(order) > 0) || (ok && got != order[0]) {
				t.Fatalf("PopFirst() #%d = (%d, %v), want oldest of %v", i, got, ok, order)
			}

			if ok {
				del(got)
			}
		case op < 10:
			got, ok := s.PopLast()
			if ok != (len(order) > 0) || (ok && got != order[len(order)-1]) {
				t.Fatalf("PopLast() #%d = (%d, %v), want newest of %v", i, got, ok, order)
			}

			if ok {
				del(got)
			}
		case op < 12:
			has := slices.Contains(order, v)
			if got := s.MoveToBack(v); got != has {
				t.Fatalf("MoveToBack(%d) #%d = %v, want %v", v, i, got, has)
			}

			if has {
				del(v)
				order = append(order, v)
			}
		case op < 13 && i%40 == 0:
			mod := 2 + r.IntN(4)
			got := s.RemoveFunc(func(x int) bool { return x%mod == 0 })
			before := len(order)
			order = slices.DeleteFunc(order, func(x int) bool { return x%mod == 0 })

			if want := before - len(order); got != want {
				t.Fatalf("RemoveFunc(multiples of %d) #%d = %d, want %d", mod, i, got, want)
			}
		}

		if got := s.Slice(); !slices.Equal(got, order) {
			t.Fatalf("Slice() #%d = %v, want %v", i, got, order)
		}

		if got := s.Len(); got != len(order) {
			t.Fatalf("Len() #%d = %d, want %d", i, got, len(order))
		}
	}
}
