package orderedset

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Nergous/structures/internal/shared/sharedtest"
)

// assertNewEmpty fails the test unless New returns an empty set with no
// preallocated capacity for element type T.
func assertNewEmpty[T comparable](t *testing.T) {
	t.Helper()

	s := New[T]()

	if got := s.Len(); got != 0 {
		t.Errorf("Len() = %d, want 0", got)
	}

	if got := s.Cap(); got != 0 {
		t.Errorf("Cap() = %d, want 0", got)
	}
}

// assertOrder fails the test unless s holds exactly the values in want, in
// this order.
func assertOrder[T comparable](t *testing.T, s *Set[T], want ...T) {
	t.Helper()

	if got := s.Len(); got != len(want) {
		t.Errorf("Len() = %d, want %d", got, len(want))
	}

	if got := s.Slice(); !slices.Equal(got, want) {
		t.Errorf("Slice() = %v, want %v", got, want)
	}

	if got := slices.Collect(s.All()); !slices.Equal(got, want) {
		t.Errorf("All() = %v, want %v", got, want)
	}
}

// fill returns a set holding the values 0..n-1 in insertion order.
func fill(n int) *Set[int] {
	s := New[int]()
	for i := range n {
		s.Add(i)
	}

	return s
}

// intRange returns the values lo, lo+1, ..., hi-1.
func intRange(lo, hi int) []int {
	out := make([]int, 0, max(hi-lo, 0))
	for i := range max(hi-lo, 0) {
		out = append(out, lo+i)
	}

	return out
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
			want: 1,
		},
		{
			name: "ten",
			n:    10,
			want: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewWithCap[int](tt.n)

			if got := s.Len(); got != 0 {
				t.Errorf("Len() = %d, want 0", got)
			}

			if got := s.Cap(); got != tt.want {
				t.Errorf("Cap() = %d, want %d", got, tt.want)
			}

			s.Add(1)

			if got := s.Contains(1); !got {
				t.Errorf("Contains(1) after Add(1) = %v, want true", got)
			}
		})
	}
}

func TestSet_Of(t *testing.T) {
	t.Run("keeps order of first occurrence", func(t *testing.T) {
		s := Of(3, 1, 3, 2, 1)

		assertOrder(t, s, 3, 1, 2)
	})

	t.Run("no arguments returns an empty usable set", func(t *testing.T) {
		s := Of[int]()

		assertOrder(t, s)

		s.Add(1)

		assertOrder(t, s, 1)
	})
}

func TestSet_From(t *testing.T) {
	t.Run("keeps order of first occurrence", func(t *testing.T) {
		s := From([]string{"go", "set", "go", "map"})

		assertOrder(t, s, "go", "set", "map")
	})

	t.Run("never aliases the input", func(t *testing.T) {
		in := []int{1, 2, 3}
		s := From(in)

		in[0], in[1], in[2] = 7, 8, 9

		assertOrder(t, s, 1, 2, 3)
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

			assertOrder(t, s)

			s.Add(1)

			assertOrder(t, s, 1)
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

func TestSet_Cap(t *testing.T) {
	t.Run("new set has zero cap", func(t *testing.T) {
		s := New[int]()

		if got := s.Cap(); got != 0 {
			t.Errorf("Cap() = %d, want 0", got)
		}
	})

	t.Run("never less than Len while growing", func(t *testing.T) {
		s := New[int]()
		for i := range 100 {
			s.Add(i)
			if c, l := s.Cap(), s.Len(); c < l {
				t.Fatalf("Cap() #%d = %d, want >= Len() = %d", i, c, l)
			}
		}
	})

	t.Run("no reallocation within preallocated capacity", func(t *testing.T) {
		s := NewWithCap[int](4)
		for i := range 4 {
			s.Add(i)
		}

		if got := s.Cap(); got != 4 {
			t.Errorf("Cap() after 4 Add() = %d, want 4", got)
		}
	})

	t.Run("counts holes left by Remove", func(t *testing.T) {
		s := NewWithCap[int](32)
		s.AddN(intRange(0, 20)...)
		capBefore := s.Cap()

		s.Remove(5)
		s.Remove(6)

		if got := s.Cap(); got != capBefore {
			t.Errorf("Cap() after Remove(5), Remove(6) = %d, want %d",
				got, capBefore)
		}
	})
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

func TestSet_First(t *testing.T) {
	t.Run("empty returns zero value and false", func(t *testing.T) {
		s := New[int]()

		if got, ok := s.First(); ok || got != 0 {
			t.Errorf("First() on empty = (%d, %v), want (0, false)", got, ok)
		}
	})

	t.Run("returns first without removing it", func(t *testing.T) {
		s := Of(1, 2, 3)

		if got, ok := s.First(); !ok || got != 1 {
			t.Errorf("First() = (%d, %v), want (1, true)", got, ok)
		}

		assertOrder(t, s, 1, 2, 3)
	})

	t.Run("follows Remove of the oldest", func(t *testing.T) {
		s := Of(1, 2, 3)
		s.Remove(1)

		if got, ok := s.First(); !ok || got != 2 {
			t.Errorf("First() after Remove(1) = (%d, %v), want (2, true)",
				got, ok)
		}
	})
}

func TestSet_Last(t *testing.T) {
	t.Run("empty returns zero value and false", func(t *testing.T) {
		s := New[int]()

		if got, ok := s.Last(); ok || got != 0 {
			t.Errorf("Last() on empty = (%d, %v), want (0, false)", got, ok)
		}
	})

	t.Run("returns last without removing it", func(t *testing.T) {
		s := Of(1, 2, 3)

		if got, ok := s.Last(); !ok || got != 3 {
			t.Errorf("Last() = (%d, %v), want (3, true)", got, ok)
		}

		assertOrder(t, s, 1, 2, 3)
	})

	t.Run("follows Remove of the newest", func(t *testing.T) {
		s := Of(1, 2, 3)
		s.Remove(3)

		if got, ok := s.Last(); !ok || got != 2 {
			t.Errorf("Last() after Remove(3) = (%d, %v), want (2, true)",
				got, ok)
		}
	})
}

func TestSet_All(t *testing.T) {
	t.Run("yields oldest to newest", func(t *testing.T) {
		s := Of(3, 1, 4, 2)
		s.Remove(4)

		got := slices.Collect(s.All())
		if want := []int{3, 1, 2}; !slices.Equal(got, want) {
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

		assertOrder(t, s, 1, 2, 3)
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

		if want := []int{3, 1}; !slices.Equal(got, want) {
			t.Errorf("All() with early break = %v, want %v", got, want)
		}

		if got := s.Len(); got != 3 {
			t.Errorf("Len() after break = %d, want 3", got)
		}
	})
}

func TestSet_Backward(t *testing.T) {
	t.Run("yields newest to oldest", func(t *testing.T) {
		s := Of(3, 1, 4, 2)
		s.Remove(4)

		got := slices.Collect(s.Backward())
		if want := []int{2, 1, 3}; !slices.Equal(got, want) {
			t.Errorf("Backward() = %v, want %v", got, want)
		}
	})

	t.Run("empty set yields nothing", func(t *testing.T) {
		s := New[int]()

		if got := slices.Collect(s.Backward()); len(got) != 0 {
			t.Errorf("Backward() on empty = %v, want no elements", got)
		}
	})

	t.Run("does not consume", func(t *testing.T) {
		s := Of(1, 2, 3)

		_ = slices.Collect(s.Backward())

		assertOrder(t, s, 1, 2, 3)
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

		if want := []int{2, 1}; !slices.Equal(got, want) {
			t.Errorf("Backward() with early break = %v, want %v", got, want)
		}

		if got := s.Len(); got != 3 {
			t.Errorf("Len() after break = %d, want 3", got)
		}
	})
}

func TestSet_Clone(t *testing.T) {
	t.Run("copies all elements in order", func(t *testing.T) {
		s := fill(40)
		for i := range 10 {
			s.Remove(i)
		}

		c := s.Clone()

		assertOrder(t, c, intRange(10, 40)...)
	})

	t.Run("capacity equals length", func(t *testing.T) {
		s := NewWithCap[int](64)
		s.AddN(1, 2, 3, 4, 5)

		c := s.Clone()

		if got, want := c.Cap(), c.Len(); got != want {
			t.Errorf("Cap() of clone = %d, want Len() = %d", got, want)
		}
	})

	t.Run("is independent of the original", func(t *testing.T) {
		s := Of(1, 2, 3)

		c := s.Clone()
		c.Add(4)
		c.Remove(2)
		c.MoveToBack(1)
		s.Remove(1)

		assertOrder(t, c, 3, 4, 1)
		assertOrder(t, s, 2, 3)
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
	t.Run("returns values oldest to newest", func(t *testing.T) {
		s := Of(3, 1, 2)

		got := s.Slice()

		if want := []int{3, 1, 2}; !slices.Equal(got, want) {
			t.Errorf("Slice() = %v, want %v", got, want)
		}
	})

	t.Run("returns an independent copy", func(t *testing.T) {
		s := Of(3, 1, 2)

		out := s.Slice()
		for i := range out {
			out[i] = 99
		}

		assertOrder(t, s, 3, 1, 2)
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
	t.Run("adds every value once", func(t *testing.T) {
		s := Of(1)

		s.AddN(2, 3, 3, 1)

		assertOrder(t, s, 1, 2, 3)
	})

	t.Run("no arguments is a no-op", func(t *testing.T) {
		s := Of(1)

		s.AddN()

		assertOrder(t, s, 1)
	})
}

func TestSet_AddSet(t *testing.T) {
	t.Run("adds the values of other", func(t *testing.T) {
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

	t.Run("zero value receiver stores the values", func(t *testing.T) {
		var s Set[int]

		s.AddSet(Of(2, 1))

		assertOrder(t, &s, 2, 1)
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
	t.Run("deletes the values of other", func(t *testing.T) {
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
		s := fill(32)

		s.RemoveSet(s)

		assertOrder(t, s)

		s.Add(7)

		assertOrder(t, s, 7)
	})
}

func TestSet_RetainSet(t *testing.T) {
	t.Run("keeps the values present in other", func(t *testing.T) {
		s := Of(5, 1, 4, 2, 3)

		s.RetainSet(Of(3, 4, 5, 9))

		assertOrder(t, s, 5, 4, 3)
	})

	t.Run("nil other empties the set", func(t *testing.T) {
		s := Of(1, 2, 3)

		s.RetainSet(nil)

		assertOrder(t, s)

		s.Add(4)

		assertOrder(t, s, 4)
	})

	t.Run("self is a no-op", func(t *testing.T) {
		s := Of(1, 2, 3)

		s.RetainSet(s)

		assertOrder(t, s, 1, 2, 3)
	})

	// Regression: removing more than half of 32+ values triggers compaction
	// in the middle of the pass.
	t.Run("removes more than half", func(t *testing.T) {
		s := fill(32)

		s.RetainSet(Of(31))

		assertOrder(t, s, 31)

		s = fill(40)

		s.RetainSet(Of(39, 0, 20))

		assertOrder(t, s, 0, 20, 39)

		s.Add(100)

		assertOrder(t, s, 0, 20, 39, 100)
	})
}

func TestSet_RemoveFunc(t *testing.T) {
	t.Run("deletes selected values and counts them", func(t *testing.T) {
		s := fill(10)

		got := s.RemoveFunc(func(v int) bool { return v%3 == 0 })

		if got != 4 {
			t.Errorf("RemoveFunc(multiples of 3) = %d, want 4", got)
		}

		assertOrder(t, s, 1, 2, 4, 5, 7, 8)
	})

	t.Run("calls del exactly once per value", func(t *testing.T) {
		s := fill(5)
		var calls []int

		s.RemoveFunc(func(v int) bool {
			calls = append(calls, v)

			return v < 2
		})

		if want := intRange(0, 5); !slices.Equal(calls, want) {
			t.Errorf("del calls = %v, want %v", calls, want)
		}
	})

	t.Run("no match returns 0", func(t *testing.T) {
		s := Of(1, 2)

		if got := s.RemoveFunc(func(int) bool { return false }); got != 0 {
			t.Errorf("RemoveFunc(none) = %d, want 0", got)
		}

		assertOrder(t, s, 1, 2)
	})

	t.Run("nil predicate panics", func(t *testing.T) {
		s := Of(1)
		sharedtest.Panics(t, "RemoveFunc(nil)", "orderedset: nil predicate",
			func() { s.RemoveFunc(nil) })
	})

	t.Run("nil predicate panics on nil receiver", func(t *testing.T) {
		var s *Set[int]
		sharedtest.Panics(t, "nil.RemoveFunc(nil)",
			"orderedset: nil predicate", func() { s.RemoveFunc(nil) })
	})
}

func TestSet_PopFirst(t *testing.T) {
	t.Run("removes oldest", func(t *testing.T) {
		s := Of(1, 2, 3)

		if got, ok := s.PopFirst(); !ok || got != 1 {
			t.Errorf("PopFirst() = (%d, %v), want (1, true)", got, ok)
		}

		assertOrder(t, s, 2, 3)

		s.PopFirst()
		s.PopFirst()

		if got, ok := s.PopFirst(); ok || got != 0 {
			t.Errorf("PopFirst() on empty = (%d, %v), want (0, false)",
				got, ok)
		}

		s.Add(5)

		assertOrder(t, s, 5)
	})

	t.Run("through compaction", func(t *testing.T) {
		s := fill(64)

		for want := range 40 {
			if got, ok := s.PopFirst(); !ok || got != want {
				t.Fatalf("PopFirst() #%d = (%d, %v), want (%d, true)",
					want, got, ok, want)
			}
		}

		assertOrder(t, s, intRange(40, 64)...)
	})

	t.Run("alternating with PopLast through compaction", func(t *testing.T) {
		s := fill(64)

		for range 20 {
			s.PopFirst()
			s.PopLast()
		}

		assertOrder(t, s, intRange(20, 44)...)
	})
}

func TestSet_PopLast(t *testing.T) {
	t.Run("removes newest", func(t *testing.T) {
		s := Of(1, 2, 3)

		if got, ok := s.PopLast(); !ok || got != 3 {
			t.Errorf("PopLast() = (%d, %v), want (3, true)", got, ok)
		}

		assertOrder(t, s, 1, 2)

		s.PopLast()
		s.PopLast()

		if got, ok := s.PopLast(); ok || got != 0 {
			t.Errorf("PopLast() on empty = (%d, %v), want (0, false)",
				got, ok)
		}

		s.Add(5)

		assertOrder(t, s, 5)
	})

	t.Run("through compaction", func(t *testing.T) {
		s := fill(64)
		s.Remove(1) // leave holes near the head
		s.Remove(2)

		for i := range 40 {
			want := 63 - i
			if got, ok := s.PopLast(); !ok || got != want {
				t.Fatalf("PopLast() #%d = (%d, %v), want (%d, true)",
					i, got, ok, want)
			}
		}

		assertOrder(t, s, append([]int{0}, intRange(3, 24)...)...)
	})
}

func TestSet_MoveToBack(t *testing.T) {
	s := Of(1, 2, 3, 4)

	if got := s.MoveToBack(2); !got {
		t.Errorf("MoveToBack(2) = %v, want true", got)
	}

	if got, ok := s.Last(); !ok || got != 2 {
		t.Errorf("Last() after MoveToBack(2) = (%d, %v), want (2, true)",
			got, ok)
	}

	assertOrder(t, s, 1, 3, 4, 2)

	if got := s.MoveToBack(99); got {
		t.Errorf("MoveToBack(99) on absent = %v, want false", got)
	}

	if got := s.Contains(99); got {
		t.Errorf("Contains(99) after MoveToBack(99) = %v, "+
			"want false (must not add)", got)
	}

	assertOrder(t, s, 1, 3, 4, 2)
}

func TestSet_Union(t *testing.T) {
	t.Run("returns values of every operand", func(t *testing.T) {
		s := Of(1, 2, 3)

		u := s.Union(Of(4, 2, 5), Of(6, 1, 7, 4))

		assertOrder(t, u, 1, 2, 3, 4, 5, 6, 7)
		assertOrder(t, s, 1, 2, 3)
	})

	t.Run("nil other is treated as empty", func(t *testing.T) {
		s := Of(2, 1)

		assertOrder(t, s.Union(nil), 2, 1)
	})

	t.Run("no others returns an independent copy", func(t *testing.T) {
		s := Of(2, 1)

		u := s.Union()
		u.Add(3)

		assertOrder(t, u, 2, 1, 3)
		assertOrder(t, s, 2, 1)
	})
}

func TestSet_Intersection(t *testing.T) {
	t.Run("returns values present in every operand", func(t *testing.T) {
		s := Of(5, 1, 4, 2, 3)

		got := s.Intersection(Of(3, 4, 5, 9), Of(4, 5, 3, 1))

		assertOrder(t, got, 5, 4, 3)
		assertOrder(t, s, 5, 1, 4, 2, 3)
	})

	t.Run("nil other is treated as empty", func(t *testing.T) {
		s := Of(1, 2)

		assertOrder(t, s.Intersection(Of(1), nil))
	})

	t.Run("no others returns an independent copy", func(t *testing.T) {
		s := Of(2, 1)

		c := s.Intersection()
		c.Remove(2)

		assertOrder(t, c, 1)
		assertOrder(t, s, 2, 1)
	})

	t.Run("self returns a copy", func(t *testing.T) {
		s := Of(1, 2)

		assertOrder(t, s.Intersection(s), 1, 2)
	})

	// Regression: dropping more than half of 32+ values must not lose or
	// reorder the survivors.
	t.Run("removes more than half", func(t *testing.T) {
		s := fill(32)

		assertOrder(t, s.Intersection(Of(31)), 31)
		assertOrder(t, s.Intersection(Of(31, 0, 16)), 0, 16, 31)
		assertOrder(t, s, intRange(0, 32)...)
	})
}

func TestSet_Difference(t *testing.T) {
	t.Run("returns values absent from every other", func(t *testing.T) {
		s := Of(5, 1, 4, 2, 3)

		got := s.Difference(Of(1), Of(3, 9))

		assertOrder(t, got, 5, 4, 2)
		assertOrder(t, s, 5, 1, 4, 2, 3)
	})

	t.Run("nil other is treated as empty", func(t *testing.T) {
		s := Of(1, 2)

		assertOrder(t, s.Difference(nil), 1, 2)
	})

	t.Run("no others returns an independent copy", func(t *testing.T) {
		s := Of(2, 1)

		c := s.Difference()
		c.Remove(2)

		assertOrder(t, c, 1)
		assertOrder(t, s, 2, 1)
	})

	t.Run("self returns an empty set", func(t *testing.T) {
		s := Of(1, 2)

		assertOrder(t, s.Difference(s))
	})

	// Regression: dropping more than half of 32+ values must not lose or
	// reorder the survivors.
	t.Run("removes more than half", func(t *testing.T) {
		s := fill(32)

		assertOrder(t, s.Difference(From(intRange(0, 31))), 31)
		assertOrder(t, s, intRange(0, 32)...)
	})
}

func TestSet_SymmetricDifference(t *testing.T) {
	t.Run("returns values in exactly one operand", func(t *testing.T) {
		s := Of(1, 2, 3, 4)
		other := Of(6, 3, 5, 1)

		got := s.SymmetricDifference(other)

		assertOrder(t, got, 2, 4, 6, 5)
		assertOrder(t, s, 1, 2, 3, 4)
		assertOrder(t, other, 6, 3, 5, 1)
	})

	t.Run("nil other is treated as empty", func(t *testing.T) {
		s := Of(2, 1)

		assertOrder(t, s.SymmetricDifference(nil), 2, 1)
	})

	t.Run("self returns an empty set", func(t *testing.T) {
		s := fill(32)

		assertOrder(t, s.SymmetricDifference(s))
		assertOrder(t, s, intRange(0, 32)...)
	})

	t.Run("large overlap", func(t *testing.T) {
		s := fill(32)

		assertOrder(t, s.SymmetricDifference(fill(34)), 32, 33)
		assertOrder(t, s.SymmetricDifference(fill(32)))
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

	if got := s.Cap(); got != 0 {
		t.Errorf("Cap() after Clear() = %d, want 0", got)
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
	t.Run("Reset keeps capacity", func(t *testing.T) {
		s := NewWithCap[int](8)
		s.AddN(1, 2, 3)
		s.Remove(2)

		s.Reset()

		if got := s.Len(); got != 0 {
			t.Errorf("Len() after Reset() = %d, want 0", got)
		}

		if got := s.Cap(); got != 8 {
			t.Errorf("Cap() after Reset() = %d, want 8", got)
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

		assertOrder(t, s, 42)
	})
}

func TestSet_Shrink(t *testing.T) {
	t.Run("reduces capacity to length and keeps contents", func(t *testing.T) {
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
	})

	t.Run("no-op when tight", func(t *testing.T) {
		s := fill(3)
		s.Shrink()

		s.Shrink()

		if got, want := s.Cap(), s.Len(); got != want {
			t.Errorf("Cap() after Shrink() = %d, want Len() = %d", got, want)
		}

		assertOrder(t, s, 0, 1, 2)
	})

	t.Run("empty set drops to zero capacity", func(t *testing.T) {
		s := NewWithCap[int](8)

		s.Shrink()

		if got := s.Cap(); got != 0 {
			t.Errorf("Cap() after Shrink() on empty = %d, want 0", got)
		}
	})

	t.Run("usable after Shrink", func(t *testing.T) {
		s := Of(1)
		s.Shrink()

		s.Add(2)

		assertOrder(t, s, 1, 2)
	})
}

func TestSet_Clip(t *testing.T) {
	t.Run("reduces capacity to length and keeps contents", func(t *testing.T) {
		s := NewWithCap[int](64)
		s.AddN(1, 2, 3)

		s.Clip()

		if got, want := s.Cap(), s.Len(); got != want {
			t.Errorf("Cap() after Clip() = %d, want Len() = %d", got, want)
		}

		assertOrder(t, s, 1, 2, 3)
	})

	t.Run("empty set drops to zero capacity", func(t *testing.T) {
		s := NewWithCap[int](8)

		s.Clip()

		if got := s.Cap(); got != 0 {
			t.Errorf("Cap() after Clip() on empty = %d, want 0", got)
		}
	})

	t.Run("usable after Clip", func(t *testing.T) {
		s := NewWithCap[int](16)
		s.Add(1)
		s.Clip()

		s.Add(2)

		assertOrder(t, s, 1, 2)
	})
}

func TestSet_Grow(t *testing.T) {
	t.Run("reserves capacity without changing contents", func(t *testing.T) {
		s := Of(1, 2, 3)

		s.Grow(100)

		if got := s.Cap(); got < 103 {
			t.Errorf("Cap() after Grow(100) = %d, want >= 103", got)
		}

		assertOrder(t, s, 1, 2, 3)
	})

	t.Run("no reallocation within grown capacity", func(t *testing.T) {
		s := New[int]()
		s.Grow(8)
		capBefore := s.Cap()

		for i := range 8 {
			s.Add(i)
		}

		if got := s.Cap(); got != capBefore {
			t.Errorf("Cap() after 8 Add() = %d, want %d", got, capBefore)
		}
	})

	t.Run("non-positive n is a no-op", func(t *testing.T) {
		s := fill(3)
		capBefore := s.Cap()

		s.Grow(0)

		if got := s.Cap(); got != capBefore {
			t.Errorf("Cap() after Grow(0) = %d, want %d", got, capBefore)
		}

		s.Grow(-1)

		if got := s.Cap(); got != capBefore {
			t.Errorf("Cap() after Grow(-1) = %d, want %d", got, capBefore)
		}
	})

	t.Run("sufficient spare capacity is a no-op", func(t *testing.T) {
		s := NewWithCap[int](50)
		s.Add(1)
		capBefore := s.Cap()

		s.Grow(10)

		if got := s.Cap(); got != capBefore {
			t.Errorf("Cap() after Grow(10) = %d, want %d", got, capBefore)
		}
	})
}

// TestSet_RandomOps runs a seeded random sequence of operations against a
// slice model and checks that both agree after every step.
func TestSet_RandomOps(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	s := New[int]()
	var model []int // oldest to newest

	del := func(v int) {
		model = slices.DeleteFunc(model, func(x int) bool { return x == v })
	}

	for i := range 5000 {
		v := r.IntN(80)
		has := slices.Contains(model, v)

		switch op := r.IntN(15); {
		case op < 5:
			if got := s.Add(v); got == has {
				t.Fatalf("step %d: Add(%d) = %v, want %v", i, v, got, !has)
			}

			if !has {
				model = append(model, v)
			}
		case op < 8:
			if got := s.Remove(v); got != has {
				t.Fatalf("step %d: Remove(%d) = %v, want %v",
					i, v, got, has)
			}

			del(v)
		case op < 9:
			got, ok := s.PopFirst()
			if ok != (len(model) > 0) || (ok && got != model[0]) {
				t.Fatalf("step %d: PopFirst() = (%d, %v), "+
					"want oldest of %v", i, got, ok, model)
			}

			if ok {
				del(got)
			}
		case op < 10:
			got, ok := s.PopLast()
			if ok != (len(model) > 0) ||
				(ok && got != model[len(model)-1]) {
				t.Fatalf("step %d: PopLast() = (%d, %v), "+
					"want newest of %v", i, got, ok, model)
			}

			if ok {
				del(got)
			}
		case op < 12:
			if got := s.MoveToBack(v); got != has {
				t.Fatalf("step %d: MoveToBack(%d) = %v, want %v",
					i, v, got, has)
			}

			if has {
				del(v)
				model = append(model, v)
			}
		case op < 13 && i%40 == 0:
			mod := 2 + r.IntN(4)
			lenBefore := len(model)
			model = slices.DeleteFunc(model,
				func(x int) bool { return x%mod == 0 })

			got := s.RemoveFunc(func(x int) bool { return x%mod == 0 })
			if want := lenBefore - len(model); got != want {
				t.Fatalf("step %d: RemoveFunc(multiples of %d) = %d, "+
					"want %d", i, mod, got, want)
			}
		case i%97 == 0:
			switch r.IntN(5) {
			case 0:
				s.Clear()
				model = nil
			case 1:
				s.Reset()
				model = nil
			case 2:
				s.Shrink()
			case 3:
				s.Clip()
			default:
				s.Grow(r.IntN(16))
			}
		}

		if got := s.Slice(); !slices.Equal(got, model) {
			t.Fatalf("step %d: Slice() = %v, want %v", i, got, model)
		}

		if got := s.Len(); got != len(model) {
			t.Fatalf("step %d: Len() = %d, want %d", i, got, len(model))
		}
	}
}

// TestSet_NilReceiver checks that every read-only and removal method treats a
// nil receiver as an empty set.
func TestSet_NilReceiver(t *testing.T) {
	var s *Set[int]

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

	if got, ok := s.First(); ok || got != 0 {
		t.Errorf("nil.First() = (%d, %v), want (0, false)", got, ok)
	}

	if got, ok := s.Last(); ok || got != 0 {
		t.Errorf("nil.Last() = (%d, %v), want (0, false)", got, ok)
	}

	count := 0
	for range s.All() {
		count++
	}

	if count != 0 {
		t.Errorf("nil.All() count = %d, want 0", count)
	}

	count = 0
	for range s.Backward() {
		count++
	}

	if count != 0 {
		t.Errorf("nil.Backward() count = %d, want 0", count)
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

	if got, ok := s.PopFirst(); ok || got != 0 {
		t.Errorf("nil.PopFirst() = (%d, %v), want (0, false)", got, ok)
	}

	if got, ok := s.PopLast(); ok || got != 0 {
		t.Errorf("nil.PopLast() = (%d, %v), want (0, false)", got, ok)
	}

	if got := s.MoveToBack(1); got {
		t.Errorf("nil.MoveToBack(1) = %v, want false", got)
	}

	assertOrder(t, s.Union(Of(1, 2)), 1, 2)
	assertOrder(t, s.Intersection(Of(1)))
	assertOrder(t, s.Difference(Of(1)))
	assertOrder(t, s.SymmetricDifference(Of(1, 2)), 1, 2)

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
		{
			name: "Clip",
			op:   "Clip",
			call: func(s *Set[int]) { s.Clip() },
		},
		{
			name: "Grow",
			op:   "Grow",
			call: func(s *Set[int]) { s.Grow(8) },
		},
		{
			name: "Grow zero",
			op:   "Grow",
			call: func(s *Set[int]) { s.Grow(0) },
		},
		{
			name: "Grow negative",
			op:   "Grow",
			call: func(s *Set[int]) { s.Grow(-1) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s *Set[int]
			sharedtest.Panics(t, tt.name,
				"orderedset: "+tt.op+" on nil receiver",
				func() { tt.call(s) })
		})
	}
}

func TestSet_ZeroValue(t *testing.T) {
	var s Set[int]

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

	if got, ok := s.First(); ok || got != 0 {
		t.Errorf("First() on zero value = (%d, %v), want (0, false)", got, ok)
	}

	if got := s.Slice(); got != nil {
		t.Errorf("Slice() on zero value = %v, want nil", got)
	}

	if got := s.Remove(1); got {
		t.Errorf("Remove(1) on zero value = %v, want false", got)
	}

	if got, ok := s.PopFirst(); ok || got != 0 {
		t.Errorf("PopFirst() on zero value = (%d, %v), want (0, false)",
			got, ok)
	}

	s.Clear()
	s.Reset()
	s.Shrink()
	s.Clip()
	s.Grow(4)

	if got := s.Cap(); got < 4 {
		t.Errorf("Cap() after Grow(4) = %d, want >= 4", got)
	}

	s.AddN(10, 20)

	if got := s.Contains(20); !got {
		t.Errorf("Contains(20) after AddN(10, 20) on zero value = %v, "+
			"want true", got)
	}

	assertOrder(t, &s, 10, 20)
}
