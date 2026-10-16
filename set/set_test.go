package set

import (
	"cmp"
	"maps"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Nergous/structures/internal/shared/sharedtest"
)

// assertNewEmpty fails the test unless New returns an empty set for element
// type T.
func assertNewEmpty[T comparable](t *testing.T) {
	t.Helper()

	s := New[T]()

	if got := s.Len(); got != 0 {
		t.Errorf("Len() = %d, want 0", got)
	}

	if got := s.IsEmpty(); !got {
		t.Errorf("IsEmpty() = %v, want true", got)
	}
}

// assertValues fails the test unless s holds exactly the values in want, in
// any order. want must not contain duplicates.
func assertValues[T cmp.Ordered](t *testing.T, s *Set[T], want ...T) {
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

// fill returns a set holding the values 0..n-1.
func fill(n int) *Set[int] {
	s := New[int]()
	for i := range n {
		s.Add(i)
	}

	return s
}

func TestSet_New(t *testing.T) {
	t.Run("int", assertNewEmpty[int])
	t.Run("string", assertNewEmpty[string])
	t.Run("float32", assertNewEmpty[float32])
}

func TestSet_NewWithCap(t *testing.T) {
	tests := []struct {
		name string
		n    int
	}{
		{
			name: "negative clamps to zero",
			n:    -1,
		},
		{
			name: "zero",
			n:    0,
		},
		{
			name: "one",
			n:    1,
		},
		{
			name: "ten",
			n:    10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewWithCap[int](tt.n)

			if got := s.Len(); got != 0 {
				t.Errorf("Len() = %d, want 0", got)
			}

			if s.m == nil {
				t.Errorf("m = nil, want an allocated map")
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

	t.Run("no arguments returns an empty usable set", func(t *testing.T) {
		s := Of[int]()

		assertValues(t, s)

		s.Add(1)

		assertValues(t, s, 1)
	})
}

func TestSet_From(t *testing.T) {
	t.Run("stores each value once", func(t *testing.T) {
		s := From([]string{"go", "set", "go", "map"})

		assertValues(t, s, "go", "set", "map")
	})

	t.Run("never aliases the input", func(t *testing.T) {
		in := []int{1, 2, 3}
		s := From(in)

		in[0], in[1], in[2] = 7, 8, 9

		assertValues(t, s, 1, 2, 3)
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
	s := New[int]()
	s.AddN(1, 2, 2, 3)

	if got := s.Len(); got != 3 {
		t.Errorf("Len() after AddN(1, 2, 2, 3) = %d, want 3", got)
	}

	s.Remove(1)

	if got := s.Len(); got != 2 {
		t.Errorf("Len() after Remove(1) = %d, want 2", got)
	}
}

func TestSet_IsEmpty(t *testing.T) {
	s := New[int]()

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
}

func TestSet_Contains(t *testing.T) {
	s := Of(1, 2)

	if got := s.Contains(2); !got {
		t.Errorf("Contains(2) = %v, want true", got)
	}

	if got := s.Contains(3); got {
		t.Errorf("Contains(3) = %v, want false", got)
	}

	s.Remove(2)

	if got := s.Contains(2); got {
		t.Errorf("Contains(2) after Remove(2) = %v, want false", got)
	}
}

func TestSet_ContainsAll(t *testing.T) {
	s := Of(1, 2, 3)

	if got := s.ContainsAll(); !got {
		t.Errorf("ContainsAll() = %v, want true", got)
	}

	if got := s.ContainsAll(1, 3); !got {
		t.Errorf("ContainsAll(1, 3) = %v, want true", got)
	}

	if got := s.ContainsAll(1, 4); got {
		t.Errorf("ContainsAll(1, 4) = %v, want false", got)
	}

	s.Remove(3)

	if got := s.ContainsAll(1, 3); got {
		t.Errorf("ContainsAll(1, 3) after Remove(3) = %v, want false", got)
	}
}

func TestSet_ContainsAny(t *testing.T) {
	s := Of(1, 2, 3)

	if got := s.ContainsAny(); got {
		t.Errorf("ContainsAny() = %v, want false", got)
	}

	if got := s.ContainsAny(4, 2); !got {
		t.Errorf("ContainsAny(4, 2) = %v, want true", got)
	}

	if got := s.ContainsAny(4, 5); got {
		t.Errorf("ContainsAny(4, 5) = %v, want false", got)
	}

	s.Remove(2)

	if got := s.ContainsAny(4, 2); got {
		t.Errorf("ContainsAny(4, 2) after Remove(2) = %v, want false", got)
	}
}

func TestSet_All(t *testing.T) {
	t.Run("yields every value once", func(t *testing.T) {
		s := Of(3, 1, 4, 2)
		s.Remove(4)

		got := slices.Sorted(s.All())
		if want := []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("All() = %v, want %v", got, want)
		}
	})

	t.Run("empty set yields nothing", func(t *testing.T) {
		s := New[int]()

		if got := slices.Collect(s.All()); len(got) != 0 {
			t.Errorf("All() on empty = %v, want no elements", got)
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

		if len(got) != 2 || !s.ContainsAll(got...) {
			t.Errorf("All() with early break = %v, want 2 set values", got)
		}

		if got := s.Len(); got != 3 {
			t.Errorf("Len() after break = %d, want 3", got)
		}
	})
}

func TestSet_Clone(t *testing.T) {
	t.Run("copies all elements", func(t *testing.T) {
		s := fill(40)
		for i := range 10 {
			s.Remove(i)
		}

		c := s.Clone()

		assertValues(t, c, slices.Collect(s.All())...)
	})

	t.Run("is independent of the original", func(t *testing.T) {
		s := Of(1, 2, 3)

		c := s.Clone()
		c.Add(4)
		c.Remove(2)
		s.Remove(1)

		assertValues(t, c, 1, 3, 4)
		assertValues(t, s, 2, 3)
	})

	t.Run("clone of empty set is empty and usable", func(t *testing.T) {
		s := New[int]()

		c := s.Clone()

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
	t.Run("returns every value", func(t *testing.T) {
		s := Of(3, 1, 2)

		got := s.Slice()
		slices.Sort(got)

		if want := []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("Slice() = %v, want %v", got, want)
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
		if got := New[int]().Slice(); got != nil {
			t.Errorf("Slice() on empty = %v, want nil", got)
		}

		s := Of(1)
		s.Remove(1)

		if got := s.Slice(); got != nil {
			t.Errorf("Slice() after Remove(1) = %v, want nil", got)
		}
	})
}

func TestSet_Add(t *testing.T) {
	s := Of(1, 2, 3)

	if got := s.Add(2); got {
		t.Errorf("Add(2) on existing = %v, want false", got)
	}

	assertValues(t, s, 1, 2, 3)

	if got := s.Add(4); !got {
		t.Errorf("Add(4) = %v, want true", got)
	}

	s.Remove(1)

	if got := s.Add(1); !got {
		t.Errorf("Add(1) after Remove(1) = %v, want true", got)
	}

	assertValues(t, s, 2, 3, 4, 1)
}

func TestSet_AddN(t *testing.T) {
	t.Run("adds every value once", func(t *testing.T) {
		s := Of(1)

		s.AddN(2, 3, 3, 1)

		assertValues(t, s, 1, 2, 3)
	})

	t.Run("no arguments is a no-op", func(t *testing.T) {
		s := Of(1)

		s.AddN()

		assertValues(t, s, 1)
	})
}

func TestSet_AddSet(t *testing.T) {
	t.Run("adds the values of other", func(t *testing.T) {
		s := Of(1, 2, 3)
		other := Of(5, 2, 4, 1)

		s.AddSet(other)

		assertValues(t, s, 1, 2, 3, 5, 4)
		assertValues(t, other, 5, 2, 4, 1)
	})

	t.Run("nil other is a no-op", func(t *testing.T) {
		s := Of(1, 2)

		s.AddSet(nil)

		assertValues(t, s, 1, 2)
	})

	t.Run("self is a no-op", func(t *testing.T) {
		s := Of(1, 2, 3)

		s.AddSet(s)

		assertValues(t, s, 1, 2, 3)
	})

	t.Run("zero value receiver stores the values", func(t *testing.T) {
		var s Set[int]

		s.AddSet(Of(2, 1))

		assertValues(t, &s, 2, 1)
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

	assertValues(t, s, 1, 3)

	s.Remove(1)
	s.Remove(3)

	assertValues(t, s)

	s.Add(4)

	assertValues(t, s, 4)
}

func TestSet_RemoveSet(t *testing.T) {
	t.Run("deletes the values of other", func(t *testing.T) {
		s := Of(1, 2, 3, 4, 5)

		s.RemoveSet(Of(4, 9, 2))

		assertValues(t, s, 1, 3, 5)
	})

	t.Run("nil other is a no-op", func(t *testing.T) {
		s := Of(1, 2)

		s.RemoveSet(nil)

		assertValues(t, s, 1, 2)
	})

	t.Run("self empties the set", func(t *testing.T) {
		s := fill(32)

		s.RemoveSet(s)

		assertValues(t, s)

		if s.m == nil {
			t.Errorf("m after RemoveSet(s) = nil, want the retained map")
		}

		s.Add(7)

		assertValues(t, s, 7)
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

		if s.m == nil {
			t.Errorf("m after RetainSet(nil) = nil, want the retained map")
		}

		s.Add(4)

		assertValues(t, s, 4)
	})

	t.Run("self is a no-op", func(t *testing.T) {
		s := Of(1, 2, 3)

		s.RetainSet(s)

		assertValues(t, s, 1, 2, 3)
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
		calls := make(map[int]int)

		s.RemoveFunc(func(v int) bool {
			calls[v]++

			return v < 2
		})

		want := map[int]int{0: 1, 1: 1, 2: 1, 3: 1, 4: 1}
		if !maps.Equal(calls, want) {
			t.Errorf("del calls = %v, want %v", calls, want)
		}
	})

	t.Run("no match returns 0", func(t *testing.T) {
		s := Of(1, 2)

		if got := s.RemoveFunc(func(int) bool { return false }); got != 0 {
			t.Errorf("RemoveFunc(none) = %d, want 0", got)
		}

		assertValues(t, s, 1, 2)
	})

	t.Run("nil predicate panics", func(t *testing.T) {
		s := Of(1)
		sharedtest.Panics(t, "RemoveFunc(nil)", "set: nil predicate",
			func() { s.RemoveFunc(nil) })
	})

	t.Run("nil predicate panics on nil receiver", func(t *testing.T) {
		var s *Set[int]
		sharedtest.Panics(t, "nil.RemoveFunc(nil)", "set: nil predicate",
			func() { s.RemoveFunc(nil) })
	})
}

func TestSet_Union(t *testing.T) {
	t.Run("returns values of every operand", func(t *testing.T) {
		s := Of(1, 2, 3)

		u := s.Union(Of(4, 2, 5), Of(6, 1, 7, 4))

		assertValues(t, u, 1, 2, 3, 4, 5, 6, 7)
		assertValues(t, s, 1, 2, 3)
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
}

func TestSet_Intersection(t *testing.T) {
	t.Run("returns values present in every operand", func(t *testing.T) {
		s := Of(5, 1, 4, 2, 3)

		got := s.Intersection(Of(3, 4, 5, 9), Of(4, 5, 3, 1))

		assertValues(t, got, 5, 4, 3)
		assertValues(t, s, 5, 1, 4, 2, 3)
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
		s := Of(5, 1, 4, 2, 3)

		got := s.Difference(Of(1), Of(3, 9))

		assertValues(t, got, 5, 4, 2)
		assertValues(t, s, 5, 1, 4, 2, 3)
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
		s := Of(1, 2)

		assertValues(t, s.Difference(s))
	})
}

func TestSet_SymmetricDifference(t *testing.T) {
	t.Run("returns values in exactly one operand", func(t *testing.T) {
		s := Of(1, 2, 3, 4)
		other := Of(6, 3, 5, 1)

		got := s.SymmetricDifference(other)

		assertValues(t, got, 2, 4, 6, 5)
		assertValues(t, s, 1, 2, 3, 4)
		assertValues(t, other, 6, 3, 5, 1)
	})

	t.Run("nil other is treated as empty", func(t *testing.T) {
		s := Of(2, 1)

		assertValues(t, s.SymmetricDifference(nil), 2, 1)
	})

	t.Run("self returns an empty set", func(t *testing.T) {
		s := fill(32)

		assertValues(t, s.SymmetricDifference(s))
	})
}

func TestSet_Equal(t *testing.T) {
	tests := []struct {
		name  string
		other *Set[int]
		want  bool
	}{
		{
			name:  "same values",
			other: Of(1, 2),
			want:  true,
		},
		{
			name:  "same values in another order",
			other: Of(2, 1),
			want:  true,
		},
		{
			name:  "superset",
			other: Of(1, 2, 3),
			want:  false,
		},
		{
			name:  "same size, different values",
			other: Of(1, 3),
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
			if got := Of(1, 2).Equal(tt.other); got != tt.want {
				t.Errorf("Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSet_IsSubset(t *testing.T) {
	tests := []struct {
		name  string
		other *Set[int]
		want  bool
	}{
		{
			name:  "superset",
			other: Of(3, 2, 1),
			want:  true,
		},
		{
			name:  "equal",
			other: Of(2, 1),
			want:  true,
		},
		{
			name:  "missing a value",
			other: Of(1, 3, 4),
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
			if got := Of(1, 2).IsSubset(tt.other); got != tt.want {
				t.Errorf("IsSubset() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSet_IsSuperset(t *testing.T) {
	tests := []struct {
		name  string
		other *Set[int]
		want  bool
	}{
		{
			name:  "subset",
			other: Of(1),
			want:  true,
		},
		{
			name:  "equal",
			other: Of(2, 1),
			want:  true,
		},
		{
			name:  "missing a value",
			other: Of(1, 3),
			want:  false,
		},
		{
			name:  "larger",
			other: Of(1, 2, 3),
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
			if got := Of(1, 2).IsSuperset(tt.other); got != tt.want {
				t.Errorf("IsSuperset() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSet_IsDisjoint(t *testing.T) {
	tests := []struct {
		name  string
		other *Set[int]
		want  bool
	}{
		{
			name:  "no common values",
			other: Of(3, 4, 5, 6),
			want:  true,
		},
		{
			name:  "overlapping smaller other",
			other: Of(2),
			want:  false,
		},
		{
			name:  "overlapping larger other",
			other: Of(5, 6, 7, 1),
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
			if got := Of(1, 2).IsDisjoint(tt.other); got != tt.want {
				t.Errorf("IsDisjoint() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSet_Clear(t *testing.T) {
	s := NewWithCap[int](8)
	s.AddN(1, 2, 3)
	s.Remove(2)

	s.Clear()

	if got := s.Len(); got != 0 {
		t.Errorf("Len() after Clear() = %d, want 0", got)
	}

	if s.m != nil {
		t.Errorf("m after Clear() = %v, want nil", s.m)
	}

	if got := s.Contains(1); got {
		t.Errorf("Contains(1) after Clear() = %v, want false", got)
	}

	s.Add(99)

	if got := s.Contains(99); !got {
		t.Errorf("Contains(99) after Clear() and Add(99) = %v, want true",
			got)
	}
}

func TestSet_Reset(t *testing.T) {
	t.Run("Reset keeps the backing map", func(t *testing.T) {
		s := NewWithCap[int](8)
		s.AddN(1, 2, 3)
		s.Remove(2)

		s.Reset()

		if got := s.Len(); got != 0 {
			t.Errorf("Len() after Reset() = %d, want 0", got)
		}

		if s.m == nil {
			t.Errorf("m after Reset() = nil, want the retained map")
		}

		if got := s.Contains(1); got {
			t.Errorf("Contains(1) after Reset() = %v, want false", got)
		}
	})

	t.Run("usable after Reset", func(t *testing.T) {
		s := fill(3)
		s.Reset()

		s.Add(42)

		if got := s.Contains(42); !got {
			t.Errorf("Contains(42) after Reset() and Add(42) = %v, "+
				"want true", got)
		}

		assertValues(t, s, 42)
	})
}

func TestSet_Shrink(t *testing.T) {
	t.Run("keeps contents in a new backing map", func(t *testing.T) {
		s := fill(1024)
		s.RemoveFunc(func(v int) bool { return v >= 3 })
		old := s.m

		s.Shrink()

		assertValues(t, s, 0, 1, 2)

		old[99] = struct{}{}

		if got := s.Contains(99); got {
			t.Errorf("Contains(99) after Shrink() = %v, "+
				"want false (Shrink must copy)", got)
		}
	})

	t.Run("empty set releases the backing map", func(t *testing.T) {
		s := fill(8)
		s.Reset()

		s.Shrink()

		if s.m != nil {
			t.Errorf("m after Shrink() on empty = %v, want nil", s.m)
		}
	})

	t.Run("usable after Shrink", func(t *testing.T) {
		s := Of(1)
		s.Shrink()

		s.Add(2)

		assertValues(t, s, 1, 2)
	})
}

// TestSet_RandomOps runs a seeded random sequence of operations against a map
// model and checks that both agree after every step.
func TestSet_RandomOps(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	s := New[int]()
	model := make(map[int]struct{})

	for i := range 5000 {
		v := r.IntN(64)
		_, has := model[v]

		switch op := r.IntN(20); {
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
		case op < 18:
			if got := s.Contains(v); got != has {
				t.Fatalf("step %d: Contains(%d) = %v, want %v",
					i, v, got, has)
			}
		case op < 19:
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
		default:
			switch r.IntN(3) {
			case 0:
				s.Clear()
				clear(model)
			case 1:
				s.Reset()
				clear(model)
			default:
				s.Shrink()
			}
		}

		if got := s.Len(); got != len(model) {
			t.Fatalf("step %d: Len() = %d, want %d", i, got, len(model))
		}
	}

	assertValues(t, s, slices.Collect(maps.Keys(model))...)
}

// TestSet_NilReceiver checks that every read-only and removal method treats a
// nil receiver as an empty set.
func TestSet_NilReceiver(t *testing.T) {
	var s *Set[int]

	if got := s.Len(); got != 0 {
		t.Errorf("nil.Len() = %d, want 0", got)
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

	count := 0
	for range s.All() {
		count++
	}

	if count != 0 {
		t.Errorf("nil.All() count = %d, want 0", count)
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

	if got := s.Equal(New[int]()); !got {
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
		call func(*Set[int])
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
			name: "AddSet with nil other",
			op:   "AddSet",
			call: func(s *Set[int]) { s.AddSet(nil) },
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
			name: "Shrink",
			op:   "Shrink",
			call: func(s *Set[int]) { s.Shrink() },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s *Set[int]
			sharedtest.Panics(t, tt.name, "set: "+tt.op+" on nil receiver",
				func() { tt.call(s) })
		})
	}
}

func TestSet_ZeroValue(t *testing.T) {
	var s Set[int]

	if got := s.Len(); got != 0 {
		t.Errorf("Len() on zero value = %d, want 0", got)
	}

	if got := s.IsEmpty(); !got {
		t.Errorf("IsEmpty() on zero value = %v, want true", got)
	}

	if got := s.Contains(1); got {
		t.Errorf("Contains(1) on zero value = %v, want false", got)
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

	s.AddN(10, 20)

	if got := s.Contains(20); !got {
		t.Errorf("Contains(20) after AddN(10, 20) on zero value = %v, "+
			"want true", got)
	}

	assertValues(t, &s, 10, 20)
}
