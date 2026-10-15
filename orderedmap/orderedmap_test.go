package orderedmap

import (
	"maps"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Nergous/structures/internal/shared/sharedtest"
)

// checkInvariants verifies every structural invariant of m.
func checkInvariants[K comparable, V any](t *testing.T, m *Map[K, V]) {
	t.Helper()

	if got, want := len(m.index), len(m.entries)-m.holes; got != want {
		t.Fatalf("len(index) = %d, want len(entries)-holes = %d (len(entries) = %d, holes = %d)",
			got, want, len(m.entries), m.holes)
	}

	alive := 0
	for i, e := range m.entries {
		if !e.alive {
			var zero entry[K, V]
			if any(e) != any(zero) {
				t.Fatalf("entries[%d] = %+v, want zero entry (holes must be zeroed)", i, e)
			}

			continue
		}

		alive++
		if got, ok := m.index[e.key]; !ok || got != i {
			t.Fatalf("index[%v] = (%d, %v), want (%d, true)", e.key, got, ok, i)
		}
	}

	if alive != len(m.index) {
		t.Fatalf("live entries = %d, want len(index) = %d", alive, len(m.index))
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
		t.Fatalf("entries[%d].alive = false, want true (last entry must be live)", last)
	}

	if m.head < 0 || m.head >= len(m.entries) || !m.entries[m.head].alive {
		t.Fatalf("head = %d, want index of a live entry in [0, %d)", m.head, len(m.entries))
	}

	for i := 0; i < m.head; i++ {
		if m.entries[i].alive {
			t.Fatalf("entries[%d].alive = true, want false (before head %d)", i, m.head)
		}
	}

	if len(m.entries) >= minCompact && m.holes > len(m.entries)/2 {
		t.Fatalf("holes = %d, want <= %d (half of %d entries)", m.holes, len(m.entries)/2, len(m.entries))
	}
}

func keys[K comparable, V any](m *Map[K, V]) []K {
	return slices.Collect(m.Keys())
}

func fill(n int) *Map[int, int] {
	m := New[int, int]()
	for i := 1; i <= n; i++ {
		m.Set(i, i*10)
	}

	return m
}

func TestMap_NewWithCap(t *testing.T) {
	m := NewWithCap[int, int](16)

	if got := m.Len(); got != 0 {
		t.Errorf("Len() = %d, want 0", got)
	}

	if got := m.Cap(); got < 16 {
		t.Errorf("Cap() = %d, want >= 16", got)
	}

	if got := NewWithCap[int, int](-3).Cap(); got != 0 {
		t.Errorf("NewWithCap(-3).Cap() = %d, want 0", got)
	}
}

func TestMap_SetGet(t *testing.T) {
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

	if got, want := keys(m), []string{"b", "a", "c"}; !slices.Equal(got, want) {
		t.Errorf("Keys() = %v, want %v", got, want)
	}

	if got, ok := m.Get("a"); !ok || got != 10 {
		t.Errorf("Get(a) = (%d, %v), want (10, true)", got, ok)
	}

	if got, ok := m.Get("zzz"); ok || got != 0 {
		t.Errorf("Get(zzz) = (%d, %v), want (0, false)", got, ok)
	}

	if got := m.Contains("c"); !got {
		t.Errorf("Contains(c) = %v, want true", got)
	}

	if got := m.Contains("zzz"); got {
		t.Errorf("Contains(zzz) = %v, want false", got)
	}

	checkInvariants(t, m)
}

func TestMap_FirstLast(t *testing.T) {
	m := New[int, int]()

	if _, _, ok := m.First(); ok {
		t.Errorf("First() on empty: ok = %v, want false", ok)
	}

	if _, _, ok := m.Last(); ok {
		t.Errorf("Last() on empty: ok = %v, want false", ok)
	}

	m.Set(1, 10)

	if gotKey, gotValue, ok := m.First(); !ok || gotKey != 1 || gotValue != 10 {
		t.Errorf("First() = (%d, %d, %v), want (1, 10, true)", gotKey, gotValue, ok)
	}

	if gotKey, gotValue, ok := m.Last(); !ok || gotKey != 1 || gotValue != 10 {
		t.Errorf("Last() on single = (%d, %d, %v), want (1, 10, true)", gotKey, gotValue, ok)
	}

	m.Set(2, 20)
	m.Set(3, 30)
	m.Remove(1)

	if got, _, _ := m.First(); got != 2 {
		t.Errorf("First() key after Remove(1) = %d, want 2 (head removed)", got)
	}

	m.Remove(3)

	if got, _, _ := m.Last(); got != 2 {
		t.Errorf("Last() key after Remove(3) = %d, want 2 (tail removed)", got)
	}

	checkInvariants(t, m)
}

func TestMap_Remove(t *testing.T) {
	t.Run("head tail middle and only", func(t *testing.T) {
		m := fill(6)

		for _, k := range []int{1, 6, 3} { // head, tail, middle
			if got, ok := m.Remove(k); !ok || got != k*10 {
				t.Fatalf("Remove(%d) = (%d, %v), want (%d, true)", k, got, ok, k*10)
			}

			checkInvariants(t, m)
		}

		if got, want := keys(m), []int{2, 4, 5}; !slices.Equal(got, want) {
			t.Errorf("Keys() = %v, want %v", got, want)
		}

		if got, ok := m.Remove(3); ok || got != 0 {
			t.Errorf("Remove(3) on removed = (%d, %v), want (0, false)", got, ok)
		}

		for _, k := range []int{2, 4, 5} {
			m.Remove(k)

			checkInvariants(t, m)
		}

		if got := m.IsEmpty(); !got {
			t.Errorf("IsEmpty() after removing all = %v, want true", got)
		}

		if got := len(m.entries); got != 0 {
			t.Errorf("len(entries) after removing all = %d, want 0", got)
		}

		m.Set(9, 90) // usable again

		if got, _, _ := m.First(); got != 9 {
			t.Errorf("First() key after reuse = %d, want 9", got)
		}

		checkInvariants(t, m)
	})

	t.Run("zeroes key and value", func(t *testing.T) {
		m := New[*int, *int]()
		a, b, c := new(int), new(int), new(int)
		m.Set(a, a)
		m.Set(b, b)
		m.Set(c, c)
		m.Remove(b) // middle: a hole stays in the slice
		m.Remove(c) // tail: trimmed, slot must still be zero

		for i, got := range m.entries[:cap(m.entries)] {
			if i >= 1 && got != (entry[*int, *int]{}) {
				t.Errorf("entries[%d] = %+v, want zero entry (removed slot must not retain references)", i, got)
			}
		}
	})
}

func TestMap_PopFirstLast(t *testing.T) {
	m := fill(4)

	if gotKey, gotValue, ok := m.PopFirst(); !ok || gotKey != 1 || gotValue != 10 {
		t.Errorf("PopFirst() = (%d, %d, %v), want (1, 10, true)", gotKey, gotValue, ok)
	}

	if gotKey, gotValue, ok := m.PopLast(); !ok || gotKey != 4 || gotValue != 40 {
		t.Errorf("PopLast() = (%d, %d, %v), want (4, 40, true)", gotKey, gotValue, ok)
	}

	checkInvariants(t, m)

	m.PopFirst()
	m.PopLast()

	checkInvariants(t, m)

	if _, _, ok := m.PopFirst(); ok {
		t.Errorf("PopFirst() on empty: ok = %v, want false", ok)
	}

	if _, _, ok := m.PopLast(); ok {
		t.Errorf("PopLast() on empty: ok = %v, want false", ok)
	}
}

// A queue that never holds more than a few keys must not grow without bound.
func TestMap_QueueUsageBoundedMemory(t *testing.T) {
	m := New[int, int]()
	for i := range 100_000 {
		m.Set(i, i)
		if m.Len() > 4 {
			m.PopFirst()
		}
	}

	checkInvariants(t, m)

	if got := len(m.entries); got > 2*minCompact {
		t.Errorf("len(entries) = %d, want <= %d (%d live keys)", got, 2*minCompact, m.Len())
	}
}

func TestMap_MoveToBack(t *testing.T) {
	m := fill(4)

	if got := m.MoveToBack(2); !got {
		t.Errorf("MoveToBack(2) = %v, want true", got)
	}

	if got, want := keys(m), []int{1, 3, 4, 2}; !slices.Equal(got, want) {
		t.Errorf("Keys() after MoveToBack(2) = %v, want %v", got, want)
	}

	if got, _ := m.Get(2); got != 20 {
		t.Errorf("Get(2) after MoveToBack(2) = %d, want 20 (value must not change)", got)
	}

	checkInvariants(t, m)

	before := len(m.entries)

	if got := m.MoveToBack(2); !got { // already newest: no-op
		t.Errorf("MoveToBack(2) on newest = %v, want true", got)
	}

	if got := len(m.entries); got != before {
		t.Errorf("len(entries) after MoveToBack(2) on newest = %d, want %d (no-op)", got, before)
	}

	m.MoveToBack(1) // head

	if got, want := keys(m), []int{3, 4, 2, 1}; !slices.Equal(got, want) {
		t.Errorf("Keys() after MoveToBack(1) = %v, want %v", got, want)
	}

	if got := m.MoveToBack(99); got {
		t.Errorf("MoveToBack(99) on missing = %v, want false", got)
	}

	checkInvariants(t, m)

	single := New[int, int]()
	single.Set(1, 1)

	if got := single.MoveToBack(1); !got {
		t.Errorf("MoveToBack(1) on single = %v, want true", got)
	}

	checkInvariants(t, single)
}

func TestMap_RemoveFunc(t *testing.T) {
	m := fill(40)
	m.Remove(5) // leave a hole so RemoveFunc has one to skip
	m.Remove(7)

	seen := 0
	got := m.RemoveFunc(func(k, v int) bool {
		seen++
		if k == 0 {
			t.Fatalf("predicate called with key %d, want live keys only (hole visited)", k)
		}

		return k%2 == 0
	})

	if seen != 38 {
		t.Errorf("predicate calls = %d, want 38", seen)
	}

	if got != 20 {
		t.Errorf("RemoveFunc(even) = %d, want 20", got)
	}

	checkInvariants(t, m)

	if got := m.holes; got != 0 {
		t.Errorf("holes after RemoveFunc() = %d, want 0", got)
	}

	for k := range m.Keys() {
		if k%2 == 0 {
			t.Errorf("Keys() after RemoveFunc(even) yielded %d, want odd keys only", k)
		}
	}

	if got, _, _ := m.First(); got != 1 {
		t.Errorf("First() key after RemoveFunc(even) = %d, want 1", got)
	}

	if got := m.RemoveFunc(func(int, int) bool { return true }); got != 18 {
		t.Errorf("RemoveFunc(all) = %d, want 18", got)
	}

	if got := m.IsEmpty(); !got {
		t.Errorf("IsEmpty() after RemoveFunc(all) = %v, want true", got)
	}

	checkInvariants(t, m)

	if got := m.RemoveFunc(func(int, int) bool { return true }); got != 0 {
		t.Errorf("RemoveFunc(all) on empty = %d, want 0", got)
	}

	sharedtest.Panics(t, "RemoveFunc(nil)", "nil predicate", func() { m.RemoveFunc(nil) })
}

func TestMap_IteratorsSkipHoles(t *testing.T) {
	m := fill(8)
	m.Remove(2)
	m.Remove(5)

	var fwd, bwd []int
	for k, v := range m.All() {
		if v != k*10 {
			t.Errorf("All() yielded (%d, %d), want (%d, %d)", k, v, k, k*10)
		}

		fwd = append(fwd, k)
	}

	for k := range m.Backward() {
		bwd = append(bwd, k)
	}

	want := []int{1, 3, 4, 6, 7, 8}
	if !slices.Equal(fwd, want) {
		t.Errorf("All() keys = %v, want %v", fwd, want)
	}

	slices.Reverse(bwd)
	if !slices.Equal(bwd, want) {
		t.Errorf("Backward() keys reversed = %v, want %v", bwd, want)
	}

	if got, want := slices.Collect(m.Values()), []int{10, 30, 40, 60, 70, 80}; !slices.Equal(got, want) {
		t.Errorf("Values() = %v, want %v", got, want)
	}

	t.Run("early break stops iteration", func(t *testing.T) {
		tests := []struct {
			name  string
			count func() int
		}{
			{
				name: "All",
				count: func() (n int) {
					for range m.All() {
						n++
						break
					}

					return
				},
			},
			{
				name: "Backward",
				count: func() (n int) {
					for range m.Backward() {
						n++
						break
					}

					return
				},
			},
			{
				name: "Keys",
				count: func() (n int) {
					for range m.Keys() {
						n++
						break
					}

					return
				},
			},
			{
				name: "Values",
				count: func() (n int) {
					for range m.Values() {
						n++
						break
					}

					return
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.count(); got != 1 {
					t.Errorf("%s() with early break count = %d, want 1", tt.name, got)
				}
			})
		}
	})
}

func TestMap_Compaction(t *testing.T) {
	m := fill(64)
	capBefore := cap(m.entries)
	for k := 1; k <= 40; k++ {
		m.Remove(k)

		checkInvariants(t, m)
	}

	if got := m.holes; got != 0 && got > len(m.entries)/2 {
		t.Errorf("holes = %d, want <= %d (compaction did not run)", got, len(m.entries)/2)
	}

	if got := cap(m.entries); got != capBefore {
		t.Errorf("cap(entries) after compaction = %d, want %d (compaction must preserve capacity)", got, capBefore)
	}

	want := make([]int, 0, 24)
	for k := 41; k <= 64; k++ {
		want = append(want, k)
	}

	if got := keys(m); !slices.Equal(got, want) {
		t.Errorf("Keys() after compaction = %v, want %v", got, want)
	}

	for k := 41; k <= 64; k++ {
		if got, ok := m.Get(k); !ok || got != k*10 {
			t.Errorf("Get(%d) after compaction = (%d, %v), want (%d, true)", k, got, ok, k*10)
		}
	}
}

// TestMap_RandomOps runs a long random mix of operations against a
// slice-of-keys reference model and checks the invariants after every step.
func TestMap_RandomOps(t *testing.T) {
	r := rand.New(rand.NewPCG(21, 22))
	m := New[int, int]()
	var order []int
	vals := map[int]int{}

	del := func(k int) {
		order = slices.DeleteFunc(order, func(x int) bool { return x == k })
		delete(vals, k)
	}

	for i := range 30000 {
		k := 1 + r.IntN(120)
		switch op := r.IntN(14); {
		case op < 5:
			v := r.IntN(1000)
			_, existed := vals[k]
			if got := m.Set(k, v); got == existed {
				t.Fatalf("Set(%d, %d) #%d = %v, want %v", k, v, i, got, !existed)
			}

			if !existed {
				order = append(order, k)
			}

			vals[k] = v
		case op < 8:
			got, ok := m.Remove(k)
			if want, has := vals[k]; ok != has || (ok && got != want) {
				t.Fatalf("Remove(%d) #%d = (%d, %v), want (%d, %v)", k, i, got, ok, want, has)
			}

			del(k)
		case op < 9:
			gotKey, gotValue, ok := m.PopFirst()
			if ok != (len(order) > 0) || (ok && (gotKey != order[0] || gotValue != vals[gotKey])) {
				t.Fatalf("PopFirst() #%d = (%d, %d, %v), want oldest of %v", i, gotKey, gotValue, ok, order)
			}

			if ok {
				del(gotKey)
			}
		case op < 10:
			gotKey, gotValue, ok := m.PopLast()
			if ok != (len(order) > 0) || (ok && (gotKey != order[len(order)-1] || gotValue != vals[gotKey])) {
				t.Fatalf("PopLast() #%d = (%d, %d, %v), want newest of %v", i, gotKey, gotValue, ok, order)
			}

			if ok {
				del(gotKey)
			}
		case op < 12:
			_, has := vals[k]
			if got := m.MoveToBack(k); got != has {
				t.Fatalf("MoveToBack(%d) #%d = %v, want %v", k, i, got, has)
			}

			if has {
				del2 := slices.Index(order, k)
				order = append(slices.Delete(order, del2, del2+1), k)
			}
		case op < 13 && i%40 == 0:
			mod := 2 + r.IntN(4)
			got := m.RemoveFunc(func(k, _ int) bool { return k%mod == 0 })
			before := len(order)
			for _, x := range slices.Clone(order) {
				if x%mod == 0 {
					del(x)
				}
			}

			if want := before - len(order); got != want {
				t.Fatalf("RemoveFunc(multiples of %d) #%d = %d, want %d", mod, i, got, want)
			}
		case i%97 == 0:
			m.Shrink()
		}

		checkInvariants(t, m)

		if got := keys(m); !slices.Equal(got, order) {
			t.Fatalf("Keys() #%d = %v, want %v", i, got, order)
		}

		if got := m.Len(); got != len(order) {
			t.Fatalf("Len() #%d = %d, want %d", i, got, len(order))
		}

		for k, want := range vals {
			if got, ok := m.Get(k); !ok || got != want {
				t.Fatalf("Get(%d) #%d = (%d, %v), want (%d, true)", k, i, got, ok, want)
			}
		}
	}
}

func TestMap_Clone(t *testing.T) {
	t.Run("is independent of the original", func(t *testing.T) {
		m := fill(40)
		for k := 1; k <= 10; k++ {
			m.Remove(k)
		}

		c := m.Clone()

		checkInvariants(t, c)

		if got := c.holes; got != 0 {
			t.Errorf("holes of clone = %d, want 0", got)
		}

		if got, want := c.Cap(), c.Len(); got != want {
			t.Errorf("Cap() of clone = %d, want Len() = %d", got, want)
		}

		if got, want := keys(c), keys(m); !slices.Equal(got, want) {
			t.Errorf("Keys() of clone = %v, want %v", got, want)
		}

		c.Set(1000, 1)
		c.Remove(11)
		c.MoveToBack(12)

		checkInvariants(t, c)
		checkInvariants(t, m)

		if got := m.Contains(1000); got {
			t.Errorf("original Contains(1000) = %v, want false (clone must be independent)", got)
		}

		if got := m.Contains(11); !got {
			t.Errorf("original Contains(11) = %v, want true (clone must be independent)", got)
		}

		if got := m.Len(); got != 30 {
			t.Errorf("original Len() = %d, want 30", got)
		}
	})

	t.Run("nil clone returns usable empty map", func(t *testing.T) {
		var m *Map[int, int]

		c := m.Clone()
		if c == nil {
			t.Fatalf("nil.Clone() = nil, want usable empty map")
		}

		if got := c.IsEmpty(); !got {
			t.Errorf("nil.Clone().IsEmpty() = %v, want true", got)
		}

		c.Set(1, 1) // usable

		checkInvariants(t, c)
	})
}

func TestMap_Shrink(t *testing.T) {
	m := NewWithCap[int, int](100)
	for k := 1; k <= 10; k++ {
		m.Set(k, k)
	}

	m.Remove(2)
	m.Remove(1)
	m.Shrink()

	checkInvariants(t, m)

	if got := m.Len(); got != 8 {
		t.Errorf("Len() after Shrink() = %d, want 8", got)
	}

	if got, want := m.Cap(), m.Len(); got != want {
		t.Errorf("Cap() after Shrink() = %d, want Len() = %d", got, want)
	}

	if got := m.holes; got != 0 {
		t.Errorf("holes after Shrink() = %d, want 0", got)
	}

	if got, want := keys(m), []int{3, 4, 5, 6, 7, 8, 9, 10}; !slices.Equal(got, want) {
		t.Errorf("Keys() after Shrink() = %v, want %v", got, want)
	}

	for k := 3; k <= 10; k++ {
		if got, ok := m.Get(k); !ok || got != k {
			t.Errorf("Get(%d) after Shrink() = (%d, %v), want (%d, true)", k, got, ok, k)
		}
	}

	m.Shrink() // already tight

	checkInvariants(t, m)

	m.Set(99, 99)

	checkInvariants(t, m)
}

func TestMap_GrowClip(t *testing.T) {
	m := New[int, int]()
	m.Grow(50)

	if got := m.Cap(); got < 50 {
		t.Errorf("Cap() after Grow(50) = %d, want >= 50", got)
	}

	m.Grow(0)
	m.Grow(-1)
	m.Set(1, 1)
	m.Set(2, 2)
	m.Clip()

	if got := m.Cap(); got != 2 {
		t.Errorf("Cap() after Clip() = %d, want 2", got)
	}

	m.Set(3, 3)

	checkInvariants(t, m)
}

func TestMap_ResetClear(t *testing.T) {
	m := NewWithCap[int, int](8)
	m.Set(1, 1)
	m.Set(2, 2)
	m.Remove(1)

	m.Reset()

	if got := m.Len(); got != 0 {
		t.Errorf("Len() after Reset() = %d, want 0", got)
	}

	if got := m.Cap(); got != 8 {
		t.Errorf("Cap() after Reset() = %d, want 8", got)
	}

	if got := m.Contains(2); got {
		t.Errorf("Contains(2) after Reset() = %v, want false", got)
	}

	checkInvariants(t, m)

	m.Set(3, 3)

	checkInvariants(t, m)

	m.Clear()

	if got := m.Len(); got != 0 {
		t.Errorf("Len() after Clear() = %d, want 0", got)
	}

	if got := m.Cap(); got != 0 {
		t.Errorf("Cap() after Clear() = %d, want 0", got)
	}

	if got := m.Contains(3); got {
		t.Errorf("Contains(3) after Clear() = %v, want false", got)
	}

	checkInvariants(t, m)

	m.Set(4, 4)

	if got, _, _ := m.First(); got != 4 {
		t.Errorf("First() key after Clear() and Set(4, 4) = %d, want 4", got)
	}
}

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

	if got := m.Contains("a"); got {
		t.Errorf("nil.Contains(a) = %v, want false", got)
	}

	if _, ok := m.Get("a"); ok {
		t.Errorf("nil.Get(a): ok = %v, want false", ok)
	}

	if _, _, ok := m.First(); ok {
		t.Errorf("nil.First(): ok = %v, want false", ok)
	}

	if _, _, ok := m.Last(); ok {
		t.Errorf("nil.Last(): ok = %v, want false", ok)
	}

	if _, ok := m.Remove("a"); ok {
		t.Errorf("nil.Remove(a): ok = %v, want false", ok)
	}

	if got := m.RemoveFunc(func(string, int) bool { return true }); got != 0 {
		t.Errorf("nil.RemoveFunc(all) = %d, want 0", got)
	}

	if _, _, ok := m.PopFirst(); ok {
		t.Errorf("nil.PopFirst(): ok = %v, want false", ok)
	}

	if _, _, ok := m.PopLast(); ok {
		t.Errorf("nil.PopLast(): ok = %v, want false", ok)
	}

	if got := m.MoveToBack("a"); got {
		t.Errorf("nil.MoveToBack(a) = %v, want false", got)
	}

	if got := len(maps.Collect(m.All())); got != 0 {
		t.Errorf("nil.All() count = %d, want 0", got)
	}

	if got := len(maps.Collect(m.Backward())); got != 0 {
		t.Errorf("nil.Backward() count = %d, want 0", got)
	}

	if got := len(slices.Collect(m.Keys())); got != 0 {
		t.Errorf("nil.Keys() count = %d, want 0", got)
	}

	if got := len(slices.Collect(m.Values())); got != 0 {
		t.Errorf("nil.Values() count = %d, want 0", got)
	}

	t.Run("mutators panic", func(t *testing.T) {
		tests := []struct {
			name string
			call func(m *Map[string, int])
		}{
			{
				name: "Set",
				call: func(m *Map[string, int]) { m.Set("a", 1) },
			},
			{
				name: "Grow",
				call: func(m *Map[string, int]) { m.Grow(1) },
			},
			{
				name: "Clear",
				call: func(m *Map[string, int]) { m.Clear() },
			},
			{
				name: "Reset",
				call: func(m *Map[string, int]) { m.Reset() },
			},
			{
				name: "Shrink",
				call: func(m *Map[string, int]) { m.Shrink() },
			},
			{
				name: "Clip",
				call: func(m *Map[string, int]) { m.Clip() },
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				sharedtest.Panics(t, tt.name, tt.name+" on nil receiver", func() { tt.call(m) })
			})
		}
	})
}

func TestMap_ZeroValue(t *testing.T) {
	var m Map[string, int]

	if got := m.IsEmpty(); !got {
		t.Errorf("IsEmpty() on zero value = %v, want true", got)
	}

	if got := m.Contains("a"); got {
		t.Errorf("Contains(a) on zero value = %v, want false", got)
	}

	if _, _, ok := m.First(); ok {
		t.Errorf("First() on zero value: ok = %v, want false", ok)
	}

	if _, _, ok := m.PopFirst(); ok {
		t.Errorf("PopFirst() on zero value: ok = %v, want false", ok)
	}

	if _, ok := m.Remove("a"); ok {
		t.Errorf("Remove(a) on zero value: ok = %v, want false", ok)
	}

	m.Reset()
	m.Clip()
	m.Shrink()
	m.Grow(4)

	var z Map[string, int]

	if got := z.Set("a", 1); !got { // Set allocates the index
		t.Errorf("Set(a, 1) on zero value = %v, want true", got)
	}

	if got, ok := z.Get("a"); !ok || got != 1 {
		t.Errorf("Get(a) after Set(a, 1) on zero value = (%d, %v), want (1, true)", got, ok)
	}

	checkInvariants(t, &z)
}
