package orderedmap

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Nergous/structures/internal/shared/sharedtest"
)

// assertNewEmpty fails the test unless New returns an empty map with no
// preallocated capacity for key type K.
func assertNewEmpty[K comparable, V any](t *testing.T) {
	t.Helper()

	m := New[K, V]()

	if got := m.Len(); got != 0 {
		t.Errorf("Len() = %d, want 0", got)
	}

	if got := m.Cap(); got != 0 {
		t.Errorf("Cap() = %d, want 0", got)
	}
}

// assertInvariants fails the test unless m satisfies every structural
// invariant documented in the Map struct.
func assertInvariants[K comparable, V any](t *testing.T, m *Map[K, V]) {
	t.Helper()

	if got, want := len(m.index), len(m.entries)-m.holes; got != want {
		t.Fatalf("len(index) = %d, want len(entries)-holes = %d "+
			"(len(entries) = %d, holes = %d)",
			got, want, len(m.entries), m.holes)
	}

	alive := 0
	for i, e := range m.entries {
		if !e.alive {
			var zero entry[K, V]
			if any(e) != any(zero) {
				t.Fatalf("entries[%d] = %+v, want zero entry "+
					"(holes must be zeroed)", i, e)
			}

			continue
		}

		alive++
		if got, ok := m.index[e.key]; !ok || got != i {
			t.Fatalf("index[%v] = (%d, %v), want (%d, true)",
				e.key, got, ok, i)
		}
	}

	if alive != len(m.index) {
		t.Fatalf("live entries = %d, want len(index) = %d",
			alive, len(m.index))
	}

	if len(m.entries) == 0 {
		if m.head != 0 {
			t.Fatalf("head on empty = %d, want 0", m.head)
		}

		if m.holes != 0 {
			t.Fatalf("holes on empty = %d, want 0", m.holes)
		}

		return
	}

	if last := len(m.entries) - 1; !m.entries[last].alive {
		t.Fatalf("entries[%d].alive = false, want true "+
			"(last entry must be live)", last)
	}

	if m.head < 0 || m.head >= len(m.entries) || !m.entries[m.head].alive {
		t.Fatalf("head = %d, want index of a live entry in [0, %d)",
			m.head, len(m.entries))
	}

	for i := range m.head {
		if m.entries[i].alive {
			t.Fatalf("entries[%d].alive = true, want false (before head %d)",
				i, m.head)
		}
	}

	// Only removals compact, so Set can grow a slice past minCompact while
	// it still carries up to minCompact-1 holes; the next removal restores
	// the half bound.
	if limit := max(len(m.entries)/2, minCompact-1); m.holes > limit {
		t.Fatalf("holes = %d, want <= %d (%d entries)",
			m.holes, limit, len(m.entries))
	}
}

// assertOrder fails the test unless m holds exactly the keys want, in this
// order, and satisfies its invariants.
func assertOrder[K comparable, V any](t *testing.T, m *Map[K, V], want ...K) {
	t.Helper()

	assertInvariants(t, m)

	if got := m.Len(); got != len(want) {
		t.Errorf("Len() = %d, want %d", got, len(want))
	}

	if got := slices.Collect(m.Keys()); !slices.Equal(got, want) {
		t.Errorf("Keys() = %v, want %v", got, want)
	}
}

// assertTailZeroed fails the test unless every entry slot between Len and
// Cap holds the zero entry.
func assertTailZeroed[K, V comparable](t *testing.T, m *Map[K, V]) {
	t.Helper()

	var zero entry[K, V]
	for i, e := range m.entries[len(m.entries):cap(m.entries)] {
		if e != zero {
			t.Errorf("entries[%d] = %+v, want zero entry",
				len(m.entries)+i, e)
		}
	}
}

// fill returns a map holding the keys 0..n-1 in insertion order, each with
// the value 10*key.
func fill(n int) *Map[int, int] {
	m := New[int, int]()
	for i := range n {
		m.Set(i, i*10)
	}

	return m
}

// ptrMap returns a map of n distinct pointer keys, each mapped to itself.
func ptrMap(n int) (*Map[*int, *int], []*int) {
	m := New[*int, *int]()
	ps := make([]*int, n)
	for i := range ps {
		ps[i] = new(int)
		m.Set(ps[i], ps[i])
	}

	return m, ps
}

func TestMap_New(t *testing.T) {
	t.Run("int", assertNewEmpty[int, int])
	t.Run("string", assertNewEmpty[string, int])
	t.Run("float32", assertNewEmpty[float32, int])
}

func TestMap_NewWithCap(t *testing.T) {
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
			m := NewWithCap[int, int](tt.n)

			if got := m.Len(); got != 0 {
				t.Errorf("Len() = %d, want 0", got)
			}

			if got := m.Cap(); got != tt.want {
				t.Errorf("Cap() = %d, want %d", got, tt.want)
			}

			m.Set(1, 10)

			if got, ok := m.Get(1); !ok || got != 10 {
				t.Errorf("Get(1) after Set(1, 10) = (%d, %v), "+
					"want (10, true)", got, ok)
			}
		})
	}
}

func TestMap_Len(t *testing.T) {
	m := New[int, int]()
	m.Set(1, 10)
	m.Set(2, 20)
	m.Set(3, 30)

	if got := m.Len(); got != 3 {
		t.Errorf("Len() after three Set() = %d, want 3", got)
	}

	m.Remove(1)

	if got := m.Len(); got != 2 {
		t.Errorf("Len() after Remove(1) = %d, want 2", got)
	}
}

func TestMap_Cap(t *testing.T) {
	t.Run("new map has zero cap", func(t *testing.T) {
		m := New[int, int]()

		if got := m.Cap(); got != 0 {
			t.Errorf("Cap() = %d, want 0", got)
		}
	})

	t.Run("never less than Len while growing", func(t *testing.T) {
		m := New[int, int]()
		for i := range 100 {
			m.Set(i, i)
			if c, l := m.Cap(), m.Len(); c < l {
				t.Fatalf("Cap() #%d = %d, want >= Len() = %d", i, c, l)
			}
		}
	})

	t.Run("no reallocation within preallocated capacity", func(t *testing.T) {
		m := NewWithCap[int, int](4)
		for i := range 4 {
			m.Set(i, i)
		}

		if got := m.Cap(); got != 4 {
			t.Errorf("Cap() after 4 Set() = %d, want 4", got)
		}
	})

	t.Run("counts holes left by Remove", func(t *testing.T) {
		m := NewWithCap[int, int](8)
		for i := range 4 {
			m.Set(i, i)
		}

		m.Remove(1)

		if got := m.Cap(); got != 8 {
			t.Errorf("Cap() after Remove(1) = %d, want 8", got)
		}

		if got := len(m.entries); got != 4 {
			t.Errorf("len(entries) after Remove(1) = %d, "+
				"want 4 (hole kept)", got)
		}
	})
}

func TestMap_IsEmpty(t *testing.T) {
	m := New[int, int]()

	if got := m.IsEmpty(); !got {
		t.Errorf("IsEmpty() on empty = %v, want true", got)
	}

	m.Set(1, 10)

	if got := m.IsEmpty(); got {
		t.Errorf("IsEmpty() after Set(1, 10) = %v, want false", got)
	}

	m.Remove(1)

	if got := m.IsEmpty(); !got {
		t.Errorf("IsEmpty() after Remove(1) = %v, want true", got)
	}
}

func TestMap_Get(t *testing.T) {
	m := New[string, int]()
	m.Set("a", 1)
	m.Set("b", 2)

	if got, ok := m.Get("a"); !ok || got != 1 {
		t.Errorf("Get(a) = (%d, %v), want (1, true)", got, ok)
	}

	if got, ok := m.Get("zzz"); ok || got != 0 {
		t.Errorf("Get(zzz) = (%d, %v), want (0, false)", got, ok)
	}

	m.Set("a", 10)

	if got, ok := m.Get("a"); !ok || got != 10 {
		t.Errorf("Get(a) after Set(a, 10) = (%d, %v), want (10, true)",
			got, ok)
	}

	m.Remove("a")

	if got, ok := m.Get("a"); ok || got != 0 {
		t.Errorf("Get(a) after Remove(a) = (%d, %v), want (0, false)",
			got, ok)
	}
}

func TestMap_Contains(t *testing.T) {
	m := New[string, int]()
	m.Set("c", 3)

	if got := m.Contains("c"); !got {
		t.Errorf("Contains(c) = %v, want true", got)
	}

	if got := m.Contains("zzz"); got {
		t.Errorf("Contains(zzz) = %v, want false", got)
	}

	m.Remove("c")

	if got := m.Contains("c"); got {
		t.Errorf("Contains(c) after Remove(c) = %v, want false", got)
	}
}

func TestMap_First(t *testing.T) {
	t.Run("empty returns zero values and false", func(t *testing.T) {
		m := New[int, int]()

		if key, value, ok := m.First(); ok || key != 0 || value != 0 {
			t.Errorf("First() on empty = (%d, %d, %v), "+
				"want (0, 0, false)", key, value, ok)
		}
	})

	t.Run("returns first without removing it", func(t *testing.T) {
		m := fill(3)

		if key, value, ok := m.First(); !ok || key != 0 || value != 0 {
			t.Errorf("First() = (%d, %d, %v), want (0, 0, true)",
				key, value, ok)
		}

		assertOrder(t, m, 0, 1, 2)
	})

	t.Run("follows Remove of the oldest", func(t *testing.T) {
		m := fill(3)
		m.Remove(0)

		if key, value, ok := m.First(); !ok || key != 1 || value != 10 {
			t.Errorf("First() after Remove(0) = (%d, %d, %v), "+
				"want (1, 10, true)", key, value, ok)
		}

		assertInvariants(t, m)
	})
}

func TestMap_Last(t *testing.T) {
	t.Run("empty returns zero values and false", func(t *testing.T) {
		m := New[int, int]()

		if key, value, ok := m.Last(); ok || key != 0 || value != 0 {
			t.Errorf("Last() on empty = (%d, %d, %v), "+
				"want (0, 0, false)", key, value, ok)
		}
	})

	t.Run("returns last without removing it", func(t *testing.T) {
		m := fill(3)

		if key, value, ok := m.Last(); !ok || key != 2 || value != 20 {
			t.Errorf("Last() = (%d, %d, %v), want (2, 20, true)",
				key, value, ok)
		}

		assertOrder(t, m, 0, 1, 2)
	})

	t.Run("follows Remove of the newest", func(t *testing.T) {
		m := fill(3)
		m.Remove(2)

		if key, value, ok := m.Last(); !ok || key != 1 || value != 10 {
			t.Errorf("Last() after Remove(2) = (%d, %d, %v), "+
				"want (1, 10, true)", key, value, ok)
		}

		assertInvariants(t, m)
	})
}

func TestMap_All(t *testing.T) {
	t.Run("yields oldest to newest", func(t *testing.T) {
		m := fill(8)
		m.Remove(2) // holes must be skipped
		m.Remove(5)

		var gotKeys, gotValues []int
		for k, v := range m.All() {
			gotKeys = append(gotKeys, k)
			gotValues = append(gotValues, v)
		}

		if want := []int{0, 1, 3, 4, 6, 7}; !slices.Equal(gotKeys, want) {
			t.Errorf("All() keys = %v, want %v", gotKeys, want)
		}

		want := []int{0, 10, 30, 40, 60, 70}
		if !slices.Equal(gotValues, want) {
			t.Errorf("All() values = %v, want %v", gotValues, want)
		}
	})

	t.Run("empty map yields nothing", func(t *testing.T) {
		m := New[int, int]()

		count := 0
		for range m.All() {
			count++
		}

		if count != 0 {
			t.Errorf("All() on empty count = %d, want 0", count)
		}
	})

	t.Run("does not consume", func(t *testing.T) {
		m := fill(3)

		for range m.All() {
		}

		assertOrder(t, m, 0, 1, 2)
	})

	t.Run("break stops iteration early", func(t *testing.T) {
		m := fill(5)

		var got []int
		for k := range m.All() {
			got = append(got, k)
			if len(got) == 2 {
				break
			}
		}

		if want := []int{0, 1}; !slices.Equal(got, want) {
			t.Errorf("All() with early break = %v, want %v", got, want)
		}

		if got := m.Len(); got != 5 {
			t.Errorf("Len() after break = %d, want 5", got)
		}
	})
}

func TestMap_Backward(t *testing.T) {
	t.Run("yields newest to oldest", func(t *testing.T) {
		m := fill(8)
		m.Remove(2) // holes must be skipped
		m.Remove(5)

		var gotKeys, gotValues []int
		for k, v := range m.Backward() {
			gotKeys = append(gotKeys, k)
			gotValues = append(gotValues, v)
		}

		if want := []int{7, 6, 4, 3, 1, 0}; !slices.Equal(gotKeys, want) {
			t.Errorf("Backward() keys = %v, want %v", gotKeys, want)
		}

		want := []int{70, 60, 40, 30, 10, 0}
		if !slices.Equal(gotValues, want) {
			t.Errorf("Backward() values = %v, want %v", gotValues, want)
		}
	})

	t.Run("empty map yields nothing", func(t *testing.T) {
		m := New[int, int]()

		count := 0
		for range m.Backward() {
			count++
		}

		if count != 0 {
			t.Errorf("Backward() on empty count = %d, want 0", count)
		}
	})

	t.Run("does not consume", func(t *testing.T) {
		m := fill(3)

		for range m.Backward() {
		}

		assertOrder(t, m, 0, 1, 2)
	})

	t.Run("break stops iteration early", func(t *testing.T) {
		m := fill(5)

		var got []int
		for k := range m.Backward() {
			got = append(got, k)
			if len(got) == 2 {
				break
			}
		}

		if want := []int{4, 3}; !slices.Equal(got, want) {
			t.Errorf("Backward() with early break = %v, want %v", got, want)
		}

		if got := m.Len(); got != 5 {
			t.Errorf("Len() after break = %d, want 5", got)
		}
	})
}

func TestMap_Keys(t *testing.T) {
	t.Run("yields oldest to newest", func(t *testing.T) {
		m := fill(8)
		m.Remove(2) // holes must be skipped
		m.Remove(5)

		got := slices.Collect(m.Keys())
		if want := []int{0, 1, 3, 4, 6, 7}; !slices.Equal(got, want) {
			t.Errorf("Keys() = %v, want %v", got, want)
		}
	})

	t.Run("empty map yields nothing", func(t *testing.T) {
		m := New[int, int]()

		if got := slices.Collect(m.Keys()); len(got) != 0 {
			t.Errorf("Keys() on empty = %v, want no elements", got)
		}
	})

	t.Run("does not consume", func(t *testing.T) {
		m := fill(3)

		_ = slices.Collect(m.Keys())

		assertOrder(t, m, 0, 1, 2)
	})

	t.Run("break stops iteration early", func(t *testing.T) {
		m := fill(5)

		var got []int
		for k := range m.Keys() {
			got = append(got, k)
			if len(got) == 2 {
				break
			}
		}

		if want := []int{0, 1}; !slices.Equal(got, want) {
			t.Errorf("Keys() with early break = %v, want %v", got, want)
		}

		if got := m.Len(); got != 5 {
			t.Errorf("Len() after break = %d, want 5", got)
		}
	})
}

func TestMap_Values(t *testing.T) {
	t.Run("yields oldest to newest", func(t *testing.T) {
		m := fill(8)
		m.Remove(2) // holes must be skipped
		m.Remove(5)

		got := slices.Collect(m.Values())
		if want := []int{0, 10, 30, 40, 60, 70}; !slices.Equal(got, want) {
			t.Errorf("Values() = %v, want %v", got, want)
		}
	})

	t.Run("empty map yields nothing", func(t *testing.T) {
		m := New[int, int]()

		if got := slices.Collect(m.Values()); len(got) != 0 {
			t.Errorf("Values() on empty = %v, want no elements", got)
		}
	})

	t.Run("does not consume", func(t *testing.T) {
		m := fill(3)

		_ = slices.Collect(m.Values())

		assertOrder(t, m, 0, 1, 2)
	})

	t.Run("break stops iteration early", func(t *testing.T) {
		m := fill(5)

		var got []int
		for v := range m.Values() {
			got = append(got, v)
			if len(got) == 2 {
				break
			}
		}

		if want := []int{0, 10}; !slices.Equal(got, want) {
			t.Errorf("Values() with early break = %v, want %v", got, want)
		}

		if got := m.Len(); got != 5 {
			t.Errorf("Len() after break = %d, want 5", got)
		}
	})
}

func TestMap_Clone(t *testing.T) {
	t.Run("copies all elements in order", func(t *testing.T) {
		m := fill(40)
		for i := range 10 {
			m.Remove(i)
		}

		c := m.Clone()

		assertOrder(t, c, slices.Collect(m.Keys())...)

		if got := c.holes; got != 0 {
			t.Errorf("holes of clone = %d, want 0", got)
		}

		if got, ok := c.Get(10); !ok || got != 100 {
			t.Errorf("Get(10) of clone = (%d, %v), want (100, true)",
				got, ok)
		}
	})

	t.Run("capacity equals length", func(t *testing.T) {
		m := NewWithCap[int, int](64)
		for i := range 5 {
			m.Set(i, i)
		}

		c := m.Clone()

		if got, want := c.Cap(), c.Len(); got != want {
			t.Errorf("Cap() of clone = %d, want Len() = %d", got, want)
		}
	})

	t.Run("is independent of the original", func(t *testing.T) {
		m := fill(3)

		c := m.Clone()
		c.Set(99, 990)
		c.Remove(0)
		c.MoveToBack(1)
		m.Set(1, 11)

		assertOrder(t, c, 2, 99, 1)
		assertOrder(t, m, 0, 1, 2)

		if got, ok := c.Get(1); !ok || got != 10 {
			t.Errorf("Get(1) of clone = (%d, %v), want (10, true)", got, ok)
		}
	})

	t.Run("clone of empty map is empty and usable", func(t *testing.T) {
		m := New[int, int]()

		c := m.Clone()

		if got := c.Len(); got != 0 {
			t.Errorf("Len() of clone = %d, want 0", got)
		}

		c.Set(1, 10)

		if got, ok := c.Get(1); !ok || got != 10 {
			t.Errorf("Get(1) of clone after Set(1, 10) = (%d, %v), "+
				"want (10, true)", got, ok)
		}
	})
}

func TestMap_Set(t *testing.T) {
	m := New[string, int]()

	if got := m.Set("b", 2); !got {
		t.Errorf("Set(b, 2) = %v, want true", got)
	}

	if got := m.Set("a", 1); !got {
		t.Errorf("Set(a, 1) = %v, want true", got)
	}

	if got := m.Set("c", 3); !got {
		t.Errorf("Set(c, 3) = %v, want true", got)
	}

	if got := m.Set("a", 10); got {
		t.Errorf("Set(a, 10) on existing = %v, want false", got)
	}

	assertOrder(t, m, "b", "a", "c")

	if got, ok := m.Get("a"); !ok || got != 10 {
		t.Errorf("Get(a) after Set(a, 10) = (%d, %v), want (10, true)",
			got, ok)
	}

	m.Remove("b")
	m.Set("b", 20) // removed and set again: counts as new

	assertOrder(t, m, "a", "c", "b")
}

func TestMap_Remove(t *testing.T) {
	t.Run("removes head, tail, and middle", func(t *testing.T) {
		m := fill(6)

		for _, k := range []int{0, 5, 2} { // head, tail, middle
			if got, ok := m.Remove(k); !ok || got != k*10 {
				t.Fatalf("Remove(%d) = (%d, %v), want (%d, true)",
					k, got, ok, k*10)
			}

			assertInvariants(t, m)
		}

		assertOrder(t, m, 1, 3, 4)

		if got, ok := m.Remove(2); ok || got != 0 {
			t.Errorf("Remove(2) on removed = (%d, %v), want (0, false)",
				got, ok)
		}

		for _, k := range []int{1, 3, 4} {
			m.Remove(k)

			assertInvariants(t, m)
		}

		if got := len(m.entries); got != 0 {
			t.Errorf("len(entries) after removing all = %d, want 0", got)
		}

		m.Set(9, 90)

		assertOrder(t, m, 9)
	})

	t.Run("zeroes vacated slot", func(t *testing.T) {
		m, ps := ptrMap(3)

		if _, ok := m.Remove(ps[1]); !ok { // middle: a hole stays
			t.Fatalf("Remove(ps[1]) ok = false, want true")
		}

		if _, ok := m.Remove(ps[2]); !ok { // tail: trimmed
			t.Fatalf("Remove(ps[2]) ok = false, want true")
		}

		for i, got := range m.entries[1:cap(m.entries)] {
			if got != (entry[*int, *int]{}) {
				t.Errorf("entries[%d] after Remove() = %+v, "+
					"want zero entry (Remove must zero the slot)", i+1, got)
			}
		}
	})

	t.Run("compacts when holes dominate", func(t *testing.T) {
		m := fill(64)
		capBefore := cap(m.entries)
		for i := range 40 {
			m.Remove(i)

			assertInvariants(t, m)
		}

		if got := cap(m.entries); got != capBefore {
			t.Errorf("cap(entries) after compaction = %d, want %d "+
				"(compaction must preserve capacity)", got, capBefore)
		}

		want := make([]int, 0, 24)
		for i := range 24 {
			want = append(want, 40+i)
		}

		assertOrder(t, m, want...)

		for _, k := range want {
			if got, ok := m.Get(k); !ok || got != k*10 {
				t.Errorf("Get(%d) after compaction = (%d, %v), "+
					"want (%d, true)", k, got, ok, k*10)
			}
		}
	})
}

func TestMap_RemoveFunc(t *testing.T) {
	t.Run("deletes selected pairs and counts them", func(t *testing.T) {
		m := fill(40)
		m.Remove(5) // leave holes so RemoveFunc has some to skip
		m.Remove(7)
		want := slices.Collect(m.Keys())

		got := m.RemoveFunc(func(k, v int) bool { return k%2 == 0 })

		if got != 20 {
			t.Errorf("RemoveFunc(even) = %d, want 20", got)
		}

		if got := m.holes; got != 0 {
			t.Errorf("holes after RemoveFunc() = %d, want 0", got)
		}

		odd := slices.DeleteFunc(want, func(k int) bool { return k%2 == 0 })
		assertOrder(t, m, odd...)
	})

	t.Run("calls del exactly once per pair", func(t *testing.T) {
		m := fill(5)
		m.Remove(2)

		var calls []int
		m.RemoveFunc(func(k, v int) bool {
			if v != k*10 {
				t.Errorf("del(%d, %d), want del(%d, %d)", k, v, k, k*10)
			}

			calls = append(calls, k)

			return k < 2
		})

		if want := []int{0, 1, 3, 4}; !slices.Equal(calls, want) {
			t.Errorf("del calls = %v, want %v", calls, want)
		}
	})

	t.Run("no match returns 0", func(t *testing.T) {
		m := fill(4)

		if got := m.RemoveFunc(func(int, int) bool { return false }); got != 0 {
			t.Errorf("RemoveFunc(none) = %d, want 0", got)
		}

		assertOrder(t, m, 0, 1, 2, 3)
	})

	t.Run("all removed empties the map", func(t *testing.T) {
		m := fill(4)
		all := func(int, int) bool { return true }

		if got := m.RemoveFunc(all); got != 4 {
			t.Errorf("RemoveFunc(all) = %d, want 4", got)
		}

		assertOrder(t, m)

		if got := m.RemoveFunc(all); got != 0 {
			t.Errorf("RemoveFunc(all) on empty = %d, want 0", got)
		}
	})

	t.Run("nil predicate panics", func(t *testing.T) {
		m := fill(2)
		sharedtest.Panics(t, "RemoveFunc(nil)", "orderedmap: nil predicate",
			func() { m.RemoveFunc(nil) })
	})

	t.Run("nil predicate panics on nil receiver", func(t *testing.T) {
		var m *Map[int, int]
		sharedtest.Panics(t, "nil.RemoveFunc(nil)",
			"orderedmap: nil predicate", func() { m.RemoveFunc(nil) })
	})
}

func TestMap_PopFirst(t *testing.T) {
	t.Run("removes oldest", func(t *testing.T) {
		m := fill(3)

		if key, value, ok := m.PopFirst(); !ok || key != 0 || value != 0 {
			t.Errorf("PopFirst() = (%d, %d, %v), want (0, 0, true)",
				key, value, ok)
		}

		if key, value, ok := m.PopFirst(); !ok || key != 1 || value != 10 {
			t.Errorf("PopFirst() = (%d, %d, %v), want (1, 10, true)",
				key, value, ok)
		}

		assertOrder(t, m, 2)

		m.PopFirst()

		if key, value, ok := m.PopFirst(); ok || key != 0 || value != 0 {
			t.Errorf("PopFirst() on empty = (%d, %d, %v), "+
				"want (0, 0, false)", key, value, ok)
		}

		assertOrder(t, m)
	})

	t.Run("zeroes vacated slot", func(t *testing.T) {
		m, _ := ptrMap(3)

		if _, _, ok := m.PopFirst(); !ok {
			t.Fatalf("PopFirst() ok = false, want true")
		}

		if got := m.entries[0]; got != (entry[*int, *int]{}) {
			t.Errorf("entries[0] after PopFirst() = %+v, "+
				"want zero entry (PopFirst must zero the slot)", got)
		}
	})

	// A queue that never holds more than a few keys must not grow without
	// bound: compaction reclaims the holes PopFirst leaves at the front.
	t.Run("queue usage keeps memory bounded", func(t *testing.T) {
		m := New[int, int]()
		for i := range 100_000 {
			m.Set(i, i)
			if m.Len() > 4 {
				m.PopFirst()
			}
		}

		assertInvariants(t, m)

		if got := len(m.entries); got > 2*minCompact {
			t.Errorf("len(entries) = %d, want <= %d (%d live keys)",
				got, 2*minCompact, m.Len())
		}
	})
}

func TestMap_PopLast(t *testing.T) {
	t.Run("removes newest", func(t *testing.T) {
		m := fill(3)

		if key, value, ok := m.PopLast(); !ok || key != 2 || value != 20 {
			t.Errorf("PopLast() = (%d, %d, %v), want (2, 20, true)",
				key, value, ok)
		}

		assertOrder(t, m, 0, 1)

		m.PopLast()
		m.PopLast()

		if key, value, ok := m.PopLast(); ok || key != 0 || value != 0 {
			t.Errorf("PopLast() on empty = (%d, %d, %v), "+
				"want (0, 0, false)", key, value, ok)
		}

		assertOrder(t, m)
	})

	t.Run("zeroes vacated slot", func(t *testing.T) {
		m, _ := ptrMap(3)

		if _, _, ok := m.PopLast(); !ok {
			t.Fatalf("PopLast() ok = false, want true")
		}

		if got := m.entries[:3][2]; got != (entry[*int, *int]{}) {
			t.Errorf("entries[2] after PopLast() = %+v, "+
				"want zero entry (PopLast must zero the slot)", got)
		}
	})
}

func TestMap_MoveToBack(t *testing.T) {
	m := fill(4)

	if got := m.MoveToBack(1); !got {
		t.Errorf("MoveToBack(1) = %v, want true", got)
	}

	assertOrder(t, m, 0, 2, 3, 1)

	if got, ok := m.Get(1); !ok || got != 10 {
		t.Errorf("Get(1) after MoveToBack(1) = (%d, %v), want (10, true) "+
			"(value must not change)", got, ok)
	}

	lenBefore := len(m.entries)

	if got := m.MoveToBack(1); !got { // already newest: no-op
		t.Errorf("MoveToBack(1) on newest = %v, want true", got)
	}

	if got := len(m.entries); got != lenBefore {
		t.Errorf("len(entries) after MoveToBack(1) on newest = %d, "+
			"want %d (no-op)", got, lenBefore)
	}

	m.MoveToBack(0) // head

	assertOrder(t, m, 2, 3, 1, 0)

	if got := m.MoveToBack(99); got {
		t.Errorf("MoveToBack(99) on absent = %v, want false", got)
	}

	assertOrder(t, m, 2, 3, 1, 0)

	single := New[int, int]()
	single.Set(1, 1)

	if got := single.MoveToBack(1); !got {
		t.Errorf("MoveToBack(1) on single = %v, want true", got)
	}

	assertOrder(t, single, 1)
}

func TestMap_Clear(t *testing.T) {
	m := NewWithCap[int, int](8)
	m.Set(1, 10)
	m.Set(2, 20)
	m.Remove(1)

	m.Clear()

	if got := m.Len(); got != 0 {
		t.Errorf("Len() after Clear() = %d, want 0", got)
	}

	if got := m.Cap(); got != 0 {
		t.Errorf("Cap() after Clear() = %d, want 0", got)
	}

	if got, ok := m.Get(2); ok || got != 0 {
		t.Errorf("Get(2) after Clear() = (%d, %v), want (0, false)", got, ok)
	}

	assertInvariants(t, m)

	m.Set(99, 990)

	if got, ok := m.Get(99); !ok || got != 990 {
		t.Errorf("Get(99) after Clear() and Set(99, 990) = (%d, %v), "+
			"want (990, true)", got, ok)
	}
}

func TestMap_Reset(t *testing.T) {
	t.Run("Reset keeps capacity", func(t *testing.T) {
		m := fill(10)
		m.Remove(1) // leave a hole
		capBefore := m.Cap()

		m.Reset()

		if got := m.Len(); got != 0 {
			t.Errorf("Len() after Reset() = %d, want 0", got)
		}

		if got := m.Cap(); got != capBefore {
			t.Errorf("Cap() after Reset() = %d, want %d", got, capBefore)
		}

		if got := m.Contains(0); got {
			t.Errorf("Contains(0) after Reset() = %v, want false", got)
		}

		assertInvariants(t, m)
	})

	t.Run("zeroes backing array", func(t *testing.T) {
		m, ps := ptrMap(8)
		m.Remove(ps[1])

		m.Reset()

		assertTailZeroed(t, m)
	})

	t.Run("usable after Reset", func(t *testing.T) {
		m := fill(3)
		m.Reset()

		m.Set(42, 420)

		if got, ok := m.Get(42); !ok || got != 420 {
			t.Errorf("Get(42) after Reset() and Set(42, 420) = (%d, %v), "+
				"want (420, true)", got, ok)
		}

		assertOrder(t, m, 42)
	})
}

func TestMap_Shrink(t *testing.T) {
	t.Run("reduces capacity to length and keeps contents", func(t *testing.T) {
		m := NewWithCap[int, int](100)
		for i := range 10 {
			m.Set(i, i)
		}

		m.Remove(1)
		m.Remove(0)

		m.Shrink()

		if got, want := m.Cap(), m.Len(); got != want {
			t.Errorf("Cap() after Shrink() = %d, want Len() = %d", got, want)
		}

		if got := m.holes; got != 0 {
			t.Errorf("holes after Shrink() = %d, want 0", got)
		}

		assertOrder(t, m, 2, 3, 4, 5, 6, 7, 8, 9)

		for i := range 8 {
			k := i + 2
			if got, ok := m.Get(k); !ok || got != k {
				t.Errorf("Get(%d) after Shrink() = (%d, %v), "+
					"want (%d, true)", k, got, ok, k)
			}
		}
	})

	t.Run("allocates a new backing array", func(t *testing.T) {
		m := NewWithCap[int, int](64)
		m.Set(1, 10)
		old := &m.entries[0]

		m.Shrink()

		if got := &m.entries[0]; got == old {
			t.Errorf("&data[0] after Shrink() = %p, want a new backing array",
				got)
		}
	})

	t.Run("no-op when tight", func(t *testing.T) {
		m := fill(3)
		m.Shrink()
		old := &m.entries[0]

		m.Shrink()

		if got := &m.entries[0]; got != old {
			t.Errorf("&data[0] after Shrink() = %p, want %p (no copy)",
				got, old)
		}

		assertInvariants(t, m)
	})

	t.Run("empty map drops to zero capacity", func(t *testing.T) {
		m := NewWithCap[int, int](8)

		m.Shrink()

		if got := m.Cap(); got != 0 {
			t.Errorf("Cap() after Shrink() on empty = %d, want 0", got)
		}
	})

	t.Run("usable after Shrink", func(t *testing.T) {
		m := fill(3)
		m.Shrink()

		m.Set(99, 990)

		assertOrder(t, m, 0, 1, 2, 99)
	})
}

func TestMap_Clip(t *testing.T) {
	t.Run("reduces capacity to length and keeps contents", func(t *testing.T) {
		m := NewWithCap[int, int](64)
		m.Set(1, 10)
		m.Set(2, 20)

		m.Clip()

		if got, want := m.Cap(), m.Len(); got != want {
			t.Errorf("Cap() after Clip() = %d, want Len() = %d", got, want)
		}

		assertOrder(t, m, 1, 2)
	})

	t.Run("does not reallocate", func(t *testing.T) {
		m := NewWithCap[int, int](64)
		m.Set(1, 10)
		m.Set(2, 20)
		old := &m.entries[0]

		m.Clip()

		if got := &m.entries[0]; got != old {
			t.Errorf("&data[0] after Clip() = %p, want %p (no copy)", got, old)
		}
	})

	t.Run("empty map drops to zero capacity", func(t *testing.T) {
		m := NewWithCap[int, int](8)

		m.Clip()

		if got := m.Cap(); got != 0 {
			t.Errorf("Cap() after Clip() on empty = %d, want 0", got)
		}
	})

	t.Run("usable after Clip", func(t *testing.T) {
		m := NewWithCap[int, int](16)
		m.Set(1, 10)
		m.Clip()

		m.Set(2, 20)

		assertOrder(t, m, 1, 2)
	})
}

func TestMap_Grow(t *testing.T) {
	t.Run("reserves capacity without changing contents", func(t *testing.T) {
		m := fill(3)

		m.Grow(100)

		if got := m.Cap(); got < 103 {
			t.Errorf("Cap() after Grow(100) = %d, want >= 103", got)
		}

		assertOrder(t, m, 0, 1, 2)
	})

	t.Run("no reallocation within grown capacity", func(t *testing.T) {
		m := New[int, int]()
		m.Grow(8)
		capBefore := m.Cap()

		for i := range 8 {
			m.Set(i, i)
		}

		if got := m.Cap(); got != capBefore {
			t.Errorf("Cap() after 8 Set() = %d, want %d", got, capBefore)
		}
	})

	t.Run("non-positive n is a no-op", func(t *testing.T) {
		m := fill(3)
		capBefore := m.Cap()

		m.Grow(0)

		if got := m.Cap(); got != capBefore {
			t.Errorf("Cap() after Grow(0) = %d, want %d", got, capBefore)
		}

		m.Grow(-1)

		if got := m.Cap(); got != capBefore {
			t.Errorf("Cap() after Grow(-1) = %d, want %d", got, capBefore)
		}
	})

	t.Run("sufficient spare capacity is a no-op", func(t *testing.T) {
		m := NewWithCap[int, int](50)
		m.Set(1, 10)
		capBefore := m.Cap()

		m.Grow(10)

		if got := m.Cap(); got != capBefore {
			t.Errorf("Cap() after Grow(10) = %d, want %d", got, capBefore)
		}
	})
}

// TestMap_RandomOps runs a seeded random sequence of operations against a
// slice model and checks that both agree after every step.
func TestMap_RandomOps(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	m := New[int, int]()
	var model []int // keys, oldest to newest
	vals := map[int]int{}

	del := func(k int) {
		model = slices.DeleteFunc(model, func(x int) bool { return x == k })
		delete(vals, k)
	}

	for i := range 30000 {
		k := 1 + r.IntN(120)
		switch op := r.IntN(14); {
		case op < 5:
			v := r.IntN(1000)
			_, existed := vals[k]
			if got := m.Set(k, v); got == existed {
				t.Fatalf("step %d: Set(%d, %d) = %v, want %v",
					i, k, v, got, !existed)
			}

			if !existed {
				model = append(model, k)
			}

			vals[k] = v
		case op < 8:
			got, ok := m.Remove(k)
			if want, has := vals[k]; ok != has || got != want {
				t.Fatalf("step %d: Remove(%d) = (%d, %v), want (%d, %v)",
					i, k, got, ok, want, has)
			}

			del(k)
		case op < 9:
			key, value, ok := m.PopFirst()
			if ok != (len(model) > 0) ||
				(ok && (key != model[0] || value != vals[key])) {
				t.Fatalf("step %d: PopFirst() = (%d, %d, %v), "+
					"want oldest of %v", i, key, value, ok, model)
			}

			if ok {
				del(key)
			}
		case op < 10:
			key, value, ok := m.PopLast()
			if ok != (len(model) > 0) ||
				(ok && (key != model[len(model)-1] || value != vals[key])) {
				t.Fatalf("step %d: PopLast() = (%d, %d, %v), "+
					"want newest of %v", i, key, value, ok, model)
			}

			if ok {
				del(key)
			}
		case op < 12:
			_, has := vals[k]
			if got := m.MoveToBack(k); got != has {
				t.Fatalf("step %d: MoveToBack(%d) = %v, want %v",
					i, k, got, has)
			}

			if has {
				j := slices.Index(model, k)
				model = append(slices.Delete(model, j, j+1), k)
			}
		case op < 13 && i%40 == 0:
			mod := 2 + r.IntN(4)
			got := m.RemoveFunc(func(k, _ int) bool { return k%mod == 0 })
			lenBefore := len(model)
			for _, x := range slices.Clone(model) {
				if x%mod == 0 {
					del(x)
				}
			}

			if want := lenBefore - len(model); got != want {
				t.Fatalf("step %d: RemoveFunc(multiples of %d) = %d, "+
					"want %d", i, mod, got, want)
			}
		case i%97 == 0:
			switch r.IntN(5) {
			case 0:
				m.Clear()
				model = nil
				clear(vals)
			case 1:
				m.Reset()
				model = nil
				clear(vals)
			case 2:
				m.Shrink()
			case 3:
				m.Clip()
			default:
				m.Grow(r.IntN(16))
			}
		}

		assertInvariants(t, m)

		if got := slices.Collect(m.Keys()); !slices.Equal(got, model) {
			t.Fatalf("step %d: Keys() = %v, want %v", i, got, model)
		}

		if got := m.Len(); got != len(model) {
			t.Fatalf("step %d: Len() = %d, want %d", i, got, len(model))
		}

		for k, want := range vals {
			if got, ok := m.Get(k); !ok || got != want {
				t.Fatalf("step %d: Get(%d) = (%d, %v), want (%d, true)",
					i, k, got, ok, want)
			}
		}
	}
}

// TestMap_NilReceiver checks that every read-only and removal method treats a
// nil receiver as an empty map.
func TestMap_NilReceiver(t *testing.T) {
	var m *Map[string, int]

	if got := m.Len(); got != 0 {
		t.Errorf("nil.Len() = %d, want 0", got)
	}

	if got := m.Cap(); got != 0 {
		t.Errorf("nil.Cap() = %d, want 0", got)
	}

	if got := m.IsEmpty(); !got {
		t.Errorf("nil.IsEmpty() = %v, want true", got)
	}

	if got, ok := m.Get("a"); ok || got != 0 {
		t.Errorf("nil.Get(a) = (%d, %v), want (0, false)", got, ok)
	}

	if got := m.Contains("a"); got {
		t.Errorf("nil.Contains(a) = %v, want false", got)
	}

	if key, value, ok := m.First(); ok || key != "" || value != 0 {
		t.Errorf("nil.First() = (%q, %d, %v), want (\"\", 0, false)",
			key, value, ok)
	}

	if key, value, ok := m.Last(); ok || key != "" || value != 0 {
		t.Errorf("nil.Last() = (%q, %d, %v), want (\"\", 0, false)",
			key, value, ok)
	}

	count := 0
	for range m.All() {
		count++
	}

	if count != 0 {
		t.Errorf("nil.All() count = %d, want 0", count)
	}

	count = 0
	for range m.Backward() {
		count++
	}

	if count != 0 {
		t.Errorf("nil.Backward() count = %d, want 0", count)
	}

	count = 0
	for range m.Keys() {
		count++
	}

	if count != 0 {
		t.Errorf("nil.Keys() count = %d, want 0", count)
	}

	count = 0
	for range m.Values() {
		count++
	}

	if count != 0 {
		t.Errorf("nil.Values() count = %d, want 0", count)
	}

	c := m.Clone()
	if c == nil {
		t.Fatalf("nil.Clone() = nil, want a usable empty map")
	}

	if got := c.Len(); got != 0 {
		t.Errorf("nil.Clone().Len() = %d, want 0", got)
	}

	c.Set("a", 42)

	if got, ok := c.Get("a"); !ok || got != 42 {
		t.Errorf("Get(a) after Set(a, 42) on nil.Clone() = (%d, %v), "+
			"want (42, true)", got, ok)
	}

	if got, ok := m.Remove("a"); ok || got != 0 {
		t.Errorf("nil.Remove(a) = (%d, %v), want (0, false)", got, ok)
	}

	all := func(string, int) bool { return true }
	if got := m.RemoveFunc(all); got != 0 {
		t.Errorf("nil.RemoveFunc(all) = %d, want 0", got)
	}

	if key, value, ok := m.PopFirst(); ok || key != "" || value != 0 {
		t.Errorf("nil.PopFirst() = (%q, %d, %v), want (\"\", 0, false)",
			key, value, ok)
	}

	if key, value, ok := m.PopLast(); ok || key != "" || value != 0 {
		t.Errorf("nil.PopLast() = (%q, %d, %v), want (\"\", 0, false)",
			key, value, ok)
	}

	if got := m.MoveToBack("a"); got {
		t.Errorf("nil.MoveToBack(a) = %v, want false", got)
	}
}

// TestMap_NilReceiverMutatorsPanic checks that every method that stores,
// clears, resets, or resizes data panics on a nil receiver.
func TestMap_NilReceiverMutatorsPanic(t *testing.T) {
	tests := []struct {
		name string
		op   string
		call func(*Map[int, int])
	}{
		{
			name: "Set",
			op:   "Set",
			call: func(m *Map[int, int]) { m.Set(1, 1) },
		},
		{
			name: "Clear",
			op:   "Clear",
			call: func(m *Map[int, int]) { m.Clear() },
		},
		{
			name: "Reset",
			op:   "Reset",
			call: func(m *Map[int, int]) { m.Reset() },
		},
		{
			name: "Shrink",
			op:   "Shrink",
			call: func(m *Map[int, int]) { m.Shrink() },
		},
		{
			name: "Clip",
			op:   "Clip",
			call: func(m *Map[int, int]) { m.Clip() },
		},
		{
			name: "Grow",
			op:   "Grow",
			call: func(m *Map[int, int]) { m.Grow(8) },
		},
		{
			name: "Grow zero",
			op:   "Grow",
			call: func(m *Map[int, int]) { m.Grow(0) },
		},
		{
			name: "Grow negative",
			op:   "Grow",
			call: func(m *Map[int, int]) { m.Grow(-1) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m *Map[int, int]
			sharedtest.Panics(t, tt.name,
				"orderedmap: "+tt.op+" on nil receiver",
				func() { tt.call(m) })
		})
	}
}

func TestMap_ZeroValue(t *testing.T) {
	var m Map[string, int]

	if got := m.Len(); got != 0 {
		t.Errorf("Len() on zero value = %d, want 0", got)
	}

	if got := m.Cap(); got != 0 {
		t.Errorf("Cap() on zero value = %d, want 0", got)
	}

	if got := m.IsEmpty(); !got {
		t.Errorf("IsEmpty() on zero value = %v, want true", got)
	}

	if got, ok := m.Get("a"); ok || got != 0 {
		t.Errorf("Get(a) on zero value = (%d, %v), want (0, false)", got, ok)
	}

	if got := m.Contains("a"); got {
		t.Errorf("Contains(a) on zero value = %v, want false", got)
	}

	if key, value, ok := m.First(); ok || key != "" || value != 0 {
		t.Errorf("First() on zero value = (%q, %d, %v), "+
			"want (\"\", 0, false)", key, value, ok)
	}

	if key, value, ok := m.PopFirst(); ok || key != "" || value != 0 {
		t.Errorf("PopFirst() on zero value = (%q, %d, %v), "+
			"want (\"\", 0, false)", key, value, ok)
	}

	if got, ok := m.Remove("a"); ok || got != 0 {
		t.Errorf("Remove(a) on zero value = (%d, %v), want (0, false)",
			got, ok)
	}

	m.Clear()
	m.Reset()
	m.Shrink()
	m.Clip()
	m.Grow(4)

	if got := m.Cap(); got < 4 {
		t.Errorf("Cap() after Grow(4) = %d, want >= 4", got)
	}

	if got := m.Set("a", 1); !got { // Set allocates the index
		t.Errorf("Set(a, 1) on zero value = %v, want true", got)
	}

	if got, ok := m.Get("a"); !ok || got != 1 {
		t.Errorf("Get(a) after Set(a, 1) on zero value = (%d, %v), "+
			"want (1, true)", got, ok)
	}

	assertOrder(t, &m, "a")
}
