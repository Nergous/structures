package set

import (
	"strings"
	"testing"

	"github.com/Nergous/structures/internal/shared/sharedtest"
)

func assertSetElements[T comparable](t *testing.T, s *Set[T], want ...T) {
	t.Helper()

	if got := s.Len(); got != len(want) {
		t.Fatalf("Len() = %d, want %d", got, len(want))
	}

	seen := make(map[T]int, len(want))
	for _, v := range want {
		seen[v]++
	}

	for v := range s.All() {
		seen[v]--
	}

	for v, count := range seen {
		if count != 0 {
			t.Fatalf("element %v count delta = %d, want 0", v, count)
		}
	}
}

func assertSliceElements[T comparable](t *testing.T, got []T, want ...T) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(%v) = %d, want %d", got, len(got), len(want))
	}

	seen := make(map[T]int, len(want))
	for _, v := range want {
		seen[v]++
	}

	for _, v := range got {
		seen[v]--
	}

	for v, count := range seen {
		if count != 0 {
			t.Fatalf("element %v count delta = %d, want 0 (got %v)", v, count, got)
		}
	}
}

func assertNewEmpty[T comparable](t *testing.T) {
	t.Helper()

	s := New[T]()
	if s == nil {
		t.Fatalf("New() = nil, want set")
	}

	if got := s.Len(); got != 0 {
		t.Errorf("Len() = %d, want 0", got)
	}

	if got := s.IsEmpty(); !got {
		t.Errorf("IsEmpty() = %v, want true", got)
	}

	if s.m == nil {
		t.Errorf("m = nil, want allocated map")
	}
}

func TestSet_New(t *testing.T) {
	t.Run("int", assertNewEmpty[int])
	t.Run("string", assertNewEmpty[string])
}

func TestSet_NewWithCap(t *testing.T) {
	tests := []struct {
		name string
		cap  int
	}{
		{
			name: "zero",
			cap:  0,
		},
		{
			name: "positive",
			cap:  8,
		},
		{
			name: "negative clamps",
			cap:  -10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewWithCap[int](tt.cap)

			if got := s.Len(); got != 0 {
				t.Errorf("Len() = %d, want 0", got)
			}

			s.Add(1)

			assertSetElements(t, s, 1)
		})
	}
}

func TestSet_OfFrom(t *testing.T) {
	t.Run("Of de-duplicates values", func(t *testing.T) {
		s := Of(1, 2, 1, 3, 2)

		assertSetElements(t, s, 1, 2, 3)
	})

	t.Run("From builds from a slice", func(t *testing.T) {
		s := From([]string{"go", "go", "structures"})

		assertSetElements(t, s, "go", "structures")
	})
}

func TestSet_AddAddNRemoveContains(t *testing.T) {
	s := New[int]()

	if got := s.Add(1); !got {
		t.Errorf("Add(1) = %v, want true", got)
	}

	if got := s.Add(1); got {
		t.Errorf("Add(1) on existing = %v, want false", got)
	}

	s.AddN(2, 3, 3)

	assertSetElements(t, s, 1, 2, 3)

	if got := s.Contains(2); !got {
		t.Errorf("Contains(2) = %v, want true", got)
	}

	if got := s.Contains(4); got {
		t.Errorf("Contains(4) = %v, want false", got)
	}

	if got := s.Remove(2); !got {
		t.Errorf("Remove(2) = %v, want true", got)
	}

	if got := s.Remove(2); got {
		t.Errorf("Remove(2) on absent = %v, want false", got)
	}

	assertSetElements(t, s, 1, 3)
}

func TestSet_ZeroValue(t *testing.T) {
	var s Set[int]

	if got := s.IsEmpty(); !got {
		t.Errorf("IsEmpty() on zero value = %v, want true", got)
	}

	if got := s.Add(10); !got {
		t.Errorf("Add(10) on zero value = %v, want true", got)
	}

	s.AddN(20, 30)

	assertSetElements(t, &s, 10, 20, 30)

	s.Clear()

	if got := s.Len(); got != 0 {
		t.Errorf("Len() after Clear() = %d, want 0", got)
	}

	s.Add(40)

	assertSetElements(t, &s, 40)
}

func TestSet_ContainsAllContainsAny(t *testing.T) {
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

	if got := s.ContainsAny(); got {
		t.Errorf("ContainsAny() = %v, want false", got)
	}

	if got := s.ContainsAny(4, 2); !got {
		t.Errorf("ContainsAny(4, 2) = %v, want true", got)
	}

	if got := s.ContainsAny(4, 5); got {
		t.Errorf("ContainsAny(4, 5) = %v, want false", got)
	}
}

func TestSet_All(t *testing.T) {
	s := Of(1, 2, 3)

	var got []int
	for v := range s.All() {
		got = append(got, v)
	}

	assertSliceElements(t, got, 1, 2, 3)

	if got := s.Len(); got != 3 {
		t.Errorf("Len() after All() = %d, want 3", got)
	}

	count := 0
	for range s.All() {
		count++
		break
	}

	if count != 1 {
		t.Errorf("All() with early break count = %d, want 1", count)
	}

	if got := s.Len(); got != 3 {
		t.Errorf("Len() after early break = %d, want 3", got)
	}
}

func TestSet_Slice(t *testing.T) {
	t.Run("returns unordered independent copy", func(t *testing.T) {
		s := Of(1, 2, 3)
		sl := s.Slice()

		assertSliceElements(t, sl, 1, 2, 3)

		for i := range sl {
			sl[i] = 99
		}

		assertSetElements(t, s, 1, 2, 3)
	})

	t.Run("empty returns nil", func(t *testing.T) {
		if got := New[int]().Slice(); got != nil {
			t.Errorf("Slice() on empty = %v, want nil", got)
		}
	})
}

func TestSet_Clone(t *testing.T) {
	t.Run("copies contents independently", func(t *testing.T) {
		s := Of(1, 2, 3)
		c := s.Clone()

		assertSetElements(t, c, 1, 2, 3)

		c.Add(4)
		s.Remove(1)

		assertSetElements(t, c, 1, 2, 3, 4)
		assertSetElements(t, s, 2, 3)
	})

	t.Run("nil clone returns usable empty set", func(t *testing.T) {
		var s *Set[int]

		c := s.Clone()
		if c == nil {
			t.Fatalf("nil.Clone() = nil, want usable empty set")
		}

		c.Add(1)

		assertSetElements(t, c, 1)
	})
}

func TestSet_ClearReset(t *testing.T) {
	t.Run("Clear releases map and remains reusable", func(t *testing.T) {
		s := Of(1, 2, 3)
		s.Clear()

		if got := s.Len(); got != 0 {
			t.Errorf("Len() after Clear() = %d, want 0", got)
		}

		if s.m != nil {
			t.Errorf("m after Clear() = %v, want nil", s.m)
		}

		s.Add(4)

		assertSetElements(t, s, 4)
	})

	t.Run("Reset keeps allocated map and remains reusable", func(t *testing.T) {
		s := Of(1, 2, 3)
		s.Reset()

		if got := s.Len(); got != 0 {
			t.Errorf("Len() after Reset() = %d, want 0", got)
		}

		if s.m == nil {
			t.Errorf("m after Reset() = nil, want retained map")
		}

		s.Add(4)

		assertSetElements(t, s, 4)
	})
}

func TestSet_Algebra(t *testing.T) {
	a := Of(1, 2, 3)
	b := Of(3, 4)
	c := Of(2, 3, 5)

	assertSetElements(t, a.Union(b, c), 1, 2, 3, 4, 5)
	assertSetElements(t, a.Intersection(c), 2, 3)
	assertSetElements(t, a.Difference(b, c), 1)
	assertSetElements(t, a.SymmetricDifference(b), 1, 2, 4)

	assertSetElements(t, a, 1, 2, 3)
	assertSetElements(t, b, 3, 4)
}

func TestSet_AlgebraNilAndAlias(t *testing.T) {
	var nilSet *Set[int]

	assertSetElements(t, nilSet.Union(Of(1, 2)), 1, 2)
	assertSetElements(t, Of(1, 2).Union(nil), 1, 2)
	assertSetElements(t, Of(1, 2).Intersection(nil))
	assertSetElements(t, Of(1, 2).Difference(nil), 1, 2)
	assertSetElements(t, nilSet.SymmetricDifference(Of(1, 2)), 1, 2)

	s := Of(1, 2)
	u := s.Union()
	u.Add(3)

	assertSetElements(t, s, 1, 2)
	assertSetElements(t, u, 1, 2, 3)

	assertSetElements(t, s.Intersection(s), 1, 2)
	assertSetElements(t, s.Difference(s))
	assertSetElements(t, s.SymmetricDifference(s))
}

func TestSet_Predicates(t *testing.T) {
	a := Of(1, 2)
	b := Of(1, 2, 3)
	c := Of(4)

	if got := a.Equal(Of(2, 1)); !got {
		t.Errorf("a.Equal(Of(2, 1)) = %v, want true", got)
	}

	if got := a.Equal(b); got {
		t.Errorf("a.Equal(b) = %v, want false (different sizes)", got)
	}

	if got := a.IsSubset(b); !got {
		t.Errorf("a.IsSubset(b) = %v, want true", got)
	}

	if got := b.IsSuperset(a); !got {
		t.Errorf("b.IsSuperset(a) = %v, want true", got)
	}

	if got := a.IsDisjoint(c); !got {
		t.Errorf("a.IsDisjoint(c) = %v, want true", got)
	}

	if got := a.IsDisjoint(b); got {
		t.Errorf("a.IsDisjoint(b) = %v, want false (overlapping)", got)
	}
}

func TestSet_PredicatesNilAsEmpty(t *testing.T) {
	var nilSet *Set[int]
	empty := New[int]()
	nonEmpty := Of(1)

	if got := nilSet.Equal(empty); !got {
		t.Errorf("nil.Equal(empty) = %v, want true", got)
	}

	if got := nilSet.IsSubset(nonEmpty); !got {
		t.Errorf("nil.IsSubset(nonEmpty) = %v, want true", got)
	}

	if got := nonEmpty.IsSuperset(nilSet); !got {
		t.Errorf("nonEmpty.IsSuperset(nil) = %v, want true", got)
	}

	if got := nilSet.IsDisjoint(nonEmpty); !got {
		t.Errorf("nil.IsDisjoint(nonEmpty) = %v, want true", got)
	}
}

func TestSet_InPlaceMutators(t *testing.T) {
	t.Run("AddSet", func(t *testing.T) {
		s := Of(1)
		s.AddSet(Of(2, 3))
		s.AddSet(nil)

		assertSetElements(t, s, 1, 2, 3)
	})

	t.Run("RemoveSet", func(t *testing.T) {
		s := Of(1, 2, 3)
		s.RemoveSet(Of(2, 4))
		s.RemoveSet(nil)

		assertSetElements(t, s, 1, 3)
	})

	t.Run("RetainAll", func(t *testing.T) {
		s := Of(1, 2, 3)
		s.RetainAll(Of(2, 4))

		assertSetElements(t, s, 2)

		s.RetainAll(nil)

		assertSetElements(t, s)
	})
}

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

	if got := s.Remove(1); got {
		t.Errorf("nil.Remove(1) = %v, want false", got)
	}

	if got := s.Slice(); got != nil {
		t.Errorf("nil.Slice() = %v, want nil", got)
	}

	count := 0
	for range s.All() {
		count++
	}

	if count != 0 {
		t.Errorf("nil.All() count = %d, want 0", count)
	}

	s.RemoveSet(Of(1))
	s.RetainAll(Of(1))
}

func TestSet_NilReceiverMutatorsPanic(t *testing.T) {
	tests := []struct {
		name string
		call func(s *Set[int])
	}{
		{
			name: "Add",
			call: func(s *Set[int]) { s.Add(1) },
		},
		{
			name: "AddN",
			call: func(s *Set[int]) { s.AddN(1, 2) },
		},
		{
			name: "AddN with no arguments",
			call: func(s *Set[int]) { s.AddN() },
		},
		{
			name: "AddSet",
			call: func(s *Set[int]) { s.AddSet(Of(1)) },
		},
		{
			name: "Clear",
			call: func(s *Set[int]) { s.Clear() },
		},
		{
			name: "Reset",
			call: func(s *Set[int]) { s.Reset() },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s *Set[int]
			sharedtest.Panics(t, tt.name, "set: "+strings.Fields(tt.name)[0]+" on nil receiver", func() { tt.call(s) })
		})
	}
}
