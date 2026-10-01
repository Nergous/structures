package orderedmap

import (
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
)

// checkInvariants verifies every structural invariant of m.
func checkInvariants[K comparable, V any](t *testing.T, m *Map[K, V]) {
	t.Helper()
	if len(m.index) != len(m.entries)-m.holes {
		t.Fatalf("len(index)=%d, len(entries)=%d, holes=%d", len(m.index), len(m.entries), m.holes)
	}

	alive := 0
	for i, e := range m.entries {
		if !e.alive {
			var zero entry[K, V]
			if any(e) != any(zero) {
				t.Fatalf("hole at %d is not zeroed: %+v", i, e)
			}
			continue
		}
		alive++
		if got, ok := m.index[e.key]; !ok || got != i {
			t.Fatalf("key %v at %d, index records %d (present=%v)", e.key, i, got, ok)
		}
	}
	if alive != len(m.index) {
		t.Fatalf("%d live entries, index has %d", alive, len(m.index))
	}

	if len(m.entries) == 0 {
		if m.head != 0 || m.holes != 0 {
			t.Fatalf("empty map: head=%d holes=%d", m.head, m.holes)
		}
		return
	}
	if !m.entries[len(m.entries)-1].alive {
		t.Fatal("last entry is a hole")
	}
	if m.head < 0 || m.head >= len(m.entries) || !m.entries[m.head].alive {
		t.Fatalf("head=%d does not point at a live entry", m.head)
	}
	for i := 0; i < m.head; i++ {
		if m.entries[i].alive {
			t.Fatalf("live entry %d before head %d", i, m.head)
		}
	}
	if len(m.entries) >= minCompact && m.holes > len(m.entries)/2 {
		t.Fatalf("holes=%d exceed half of %d entries", m.holes, len(m.entries))
	}
}

func keys[K comparable, V any](m *Map[K, V]) []K {
	return slices.Collect(m.Keys())
}

func expectPanic(t *testing.T, want string, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Fatalf("expected panic containing %q", want)
		}
		if msg, _ := r.(string); !strings.Contains(msg, want) {
			t.Fatalf("panic %v, want it to contain %q", r, want)
		}
	}()
	f()
}

func fill(n int) *Map[int, int] {
	m := New[int, int]()
	for i := 1; i <= n; i++ {
		m.Set(i, i*10)
	}
	return m
}

func TestNewWithCap(t *testing.T) {
	if m := NewWithCap[int, int](16); m.Cap() < 16 || m.Len() != 0 {
		t.Fatalf("Len=%d Cap=%d", m.Len(), m.Cap())
	}
	if m := NewWithCap[int, int](-3); m.Cap() != 0 {
		t.Fatalf("negative capacity: Cap=%d", m.Cap())
	}
}

func TestSetGet_Order(t *testing.T) {
	m := New[string, int]()
	if !m.Set("b", 2) || !m.Set("a", 1) || !m.Set("c", 3) {
		t.Fatal("Set of a new key must report true")
	}
	if m.Set("a", 10) {
		t.Fatal("Set of an existing key must report false")
	}
	if got, want := keys(m), []string{"b", "a", "c"}; !slices.Equal(got, want) {
		t.Fatalf("order %v, want %v", got, want)
	}
	if v, ok := m.Get("a"); !ok || v != 10 {
		t.Fatalf("Get(a) = %v %v", v, ok)
	}
	if v, ok := m.Get("zzz"); ok || v != 0 {
		t.Fatalf("Get(zzz) = %v %v", v, ok)
	}
	if !m.Contains("c") || m.Contains("zzz") {
		t.Fatal("Contains mismatch")
	}
	checkInvariants(t, m)
}

func TestFirstLast(t *testing.T) {
	m := New[int, int]()
	if _, _, ok := m.First(); ok {
		t.Fatal("First on empty map")
	}
	if _, _, ok := m.Last(); ok {
		t.Fatal("Last on empty map")
	}

	m.Set(1, 10)
	k, v, ok := m.First()
	if !ok || k != 1 || v != 10 {
		t.Fatalf("First = %v %v %v", k, v, ok)
	}
	if k, _, ok = m.Last(); !ok || k != 1 {
		t.Fatalf("Last of single = %v %v", k, ok)
	}

	m.Set(2, 20)
	m.Set(3, 30)
	m.Remove(1)
	if k, _, _ = m.First(); k != 2 {
		t.Fatalf("First after removing head = %d, want 2", k)
	}
	m.Remove(3)
	if k, _, _ = m.Last(); k != 2 {
		t.Fatalf("Last after removing tail = %d, want 2", k)
	}
	checkInvariants(t, m)
}

func TestRemove_HeadTailMiddleOnly(t *testing.T) {
	m := fill(6)
	for _, k := range []int{1, 6, 3} { // head, tail, middle
		if v, ok := m.Remove(k); !ok || v != k*10 {
			t.Fatalf("Remove(%d) = %v %v", k, v, ok)
		}
		checkInvariants(t, m)
	}
	if got, want := keys(m), []int{2, 4, 5}; !slices.Equal(got, want) {
		t.Fatalf("order %v, want %v", got, want)
	}
	if v, ok := m.Remove(3); ok || v != 0 {
		t.Fatalf("second Remove(3) = %v %v", v, ok)
	}
	for _, k := range []int{2, 4, 5} {
		m.Remove(k)
		checkInvariants(t, m)
	}
	if !m.IsEmpty() || len(m.entries) != 0 {
		t.Fatalf("map not empty: %d entries", len(m.entries))
	}
	m.Set(9, 90) // usable again
	if k, _, _ := m.First(); k != 9 {
		t.Fatal("First after reuse")
	}
	checkInvariants(t, m)
}

func TestRemove_ZeroesKeyAndValue(t *testing.T) {
	m := New[*int, *int]()
	a, b, c := new(int), new(int), new(int)
	m.Set(a, a)
	m.Set(b, b)
	m.Set(c, c)
	m.Remove(b) // middle: a hole stays in the slice
	m.Remove(c) // tail: trimmed, slot must still be zero
	for i, e := range m.entries[:cap(m.entries)] {
		if i >= 1 && e != (entry[*int, *int]{}) {
			t.Fatalf("slot %d retains %+v", i, e)
		}
	}
}

func TestPopFirstLast(t *testing.T) {
	m := fill(4)
	if k, v, ok := m.PopFirst(); !ok || k != 1 || v != 10 {
		t.Fatalf("PopFirst = %v %v %v", k, v, ok)
	}
	if k, v, ok := m.PopLast(); !ok || k != 4 || v != 40 {
		t.Fatalf("PopLast = %v %v %v", k, v, ok)
	}
	checkInvariants(t, m)
	m.PopFirst()
	m.PopLast()
	checkInvariants(t, m)
	if _, _, ok := m.PopFirst(); ok {
		t.Fatal("PopFirst on empty map")
	}
	if _, _, ok := m.PopLast(); ok {
		t.Fatal("PopLast on empty map")
	}
}

// A queue that never holds more than a few keys must not grow without bound.
func TestQueueUsage_BoundedMemory(t *testing.T) {
	m := New[int, int]()
	for i := range 100_000 {
		m.Set(i, i)
		if m.Len() > 4 {
			m.PopFirst()
		}
	}
	checkInvariants(t, m)
	if len(m.entries) > 2*minCompact {
		t.Fatalf("entries grew to %d with %d live keys", len(m.entries), m.Len())
	}
}

func TestMoveToBack(t *testing.T) {
	m := fill(4)
	if !m.MoveToBack(2) {
		t.Fatal("MoveToBack(2) = false")
	}
	if got, want := keys(m), []int{1, 3, 4, 2}; !slices.Equal(got, want) {
		t.Fatalf("order %v, want %v", got, want)
	}
	if v, _ := m.Get(2); v != 20 {
		t.Fatalf("value changed to %d", v)
	}
	checkInvariants(t, m)

	before := len(m.entries)
	if !m.MoveToBack(2) || len(m.entries) != before { // already newest: no-op
		t.Fatal("MoveToBack of the newest key must be a no-op")
	}
	m.MoveToBack(1) // head
	if got, want := keys(m), []int{3, 4, 2, 1}; !slices.Equal(got, want) {
		t.Fatalf("order %v, want %v", got, want)
	}
	if m.MoveToBack(99) {
		t.Fatal("MoveToBack of a missing key reported true")
	}
	checkInvariants(t, m)

	single := New[int, int]()
	single.Set(1, 1)
	if !single.MoveToBack(1) {
		t.Fatal("single key")
	}
	checkInvariants(t, single)
}

func TestRemoveFunc(t *testing.T) {
	m := fill(40)
	m.Remove(5) // leave a hole so RemoveFunc has one to skip
	m.Remove(7)

	seen := 0
	n := m.RemoveFunc(func(k, v int) bool {
		seen++
		if k == 0 {
			t.Fatal("predicate called on a hole")
		}
		return k%2 == 0
	})
	if seen != 38 || n != 20 {
		t.Fatalf("seen=%d removed=%d, want 38 and 20", seen, n)
	}
	checkInvariants(t, m)
	if m.holes != 0 {
		t.Fatalf("holes=%d after RemoveFunc", m.holes)
	}
	for k := range m.Keys() {
		if k%2 == 0 {
			t.Fatalf("even key %d survived", k)
		}
	}
	if first, _, _ := m.First(); first != 1 {
		t.Fatalf("First = %d", first)
	}

	if m.RemoveFunc(func(int, int) bool { return true }) != 18 || !m.IsEmpty() {
		t.Fatal("removing everything")
	}
	checkInvariants(t, m)
	if m.RemoveFunc(func(int, int) bool { return true }) != 0 {
		t.Fatal("RemoveFunc on empty map")
	}
	expectPanic(t, "nil predicate", func() { m.RemoveFunc(nil) })
}

func TestIterators_SkipHoles(t *testing.T) {
	m := fill(8)
	m.Remove(2)
	m.Remove(5)

	var fwd, bwd []int
	for k, v := range m.All() {
		if v != k*10 {
			t.Fatalf("pair %d=%d", k, v)
		}
		fwd = append(fwd, k)
	}
	for k := range m.Backward() {
		bwd = append(bwd, k)
	}
	want := []int{1, 3, 4, 6, 7, 8}
	if !slices.Equal(fwd, want) {
		t.Fatalf("All = %v", fwd)
	}
	slices.Reverse(bwd)
	if !slices.Equal(bwd, want) {
		t.Fatalf("Backward reversed = %v", bwd)
	}
	if got := slices.Collect(m.Values()); !slices.Equal(got, []int{10, 30, 40, 60, 70, 80}) {
		t.Fatalf("Values = %v", got)
	}

	for name, seq := range map[string]func() int{
		"All": func() (n int) {
			for range m.All() {
				n++
				break
			}
			return
		},
		"Backward": func() (n int) {
			for range m.Backward() {
				n++
				break
			}
			return
		},
		"Keys": func() (n int) {
			for range m.Keys() {
				n++
				break
			}
			return
		},
		"Values": func() (n int) {
			for range m.Values() {
				n++
				break
			}
			return
		},
	} {
		if seq() != 1 {
			t.Fatalf("%s: early break did not stop iteration", name)
		}
	}
}

func TestCompaction(t *testing.T) {
	m := fill(64)
	c := cap(m.entries)
	for k := 1; k <= 40; k++ {
		m.Remove(k)
		checkInvariants(t, m)
	}
	if m.holes != 0 && m.holes > len(m.entries)/2 {
		t.Fatal("compaction did not run")
	}
	if cap(m.entries) != c {
		t.Fatalf("compaction changed Cap from %d to %d", c, cap(m.entries))
	}
	want := make([]int, 0, 24)
	for k := 41; k <= 64; k++ {
		want = append(want, k)
	}
	if got := keys(m); !slices.Equal(got, want) {
		t.Fatalf("order after compaction %v", got)
	}
	for k := 41; k <= 64; k++ {
		if v, ok := m.Get(k); !ok || v != k*10 {
			t.Fatalf("Get(%d) = %v %v after compaction", k, v, ok)
		}
	}
}

// TestRandomOps runs a long random mix of operations against a slice-of-keys
// reference model and checks the invariants after every step.
func TestRandomOps(t *testing.T) {
	r := rand.New(rand.NewPCG(21, 22))
	m := New[int, int]()
	var order []int
	vals := map[int]int{}

	del := func(k int) {
		order = slices.DeleteFunc(order, func(x int) bool { return x == k })
		delete(vals, k)
	}

	for step := range 30000 {
		k := 1 + r.IntN(120)
		switch op := r.IntN(14); {
		case op < 5:
			v := r.IntN(1000)
			_, existed := vals[k]
			if isNew := m.Set(k, v); isNew == existed {
				t.Fatalf("Set(%d) new=%v, existed=%v", k, isNew, existed)
			}
			if !existed {
				order = append(order, k)
			}
			vals[k] = v
		case op < 8:
			v, ok := m.Remove(k)
			if want, has := vals[k]; ok != has || (ok && v != want) {
				t.Fatalf("Remove(%d) = %v %v", k, v, ok)
			}
			del(k)
		case op < 9:
			k, v, ok := m.PopFirst()
			if ok != (len(order) > 0) || (ok && (k != order[0] || v != vals[k])) {
				t.Fatalf("PopFirst = %v %v %v", k, v, ok)
			}
			if ok {
				del(k)
			}
		case op < 10:
			k, v, ok := m.PopLast()
			if ok != (len(order) > 0) || (ok && (k != order[len(order)-1] || v != vals[k])) {
				t.Fatalf("PopLast = %v %v %v", k, v, ok)
			}
			if ok {
				del(k)
			}
		case op < 12:
			_, has := vals[k]
			if m.MoveToBack(k) != has {
				t.Fatalf("MoveToBack(%d) mismatch", k)
			}
			if has {
				del2 := slices.Index(order, k)
				order = append(slices.Delete(order, del2, del2+1), k)
			}
		case op < 13 && step%40 == 0:
			mod := 2 + r.IntN(4)
			removed := m.RemoveFunc(func(k, _ int) bool { return k%mod == 0 })
			before := len(order)
			for _, x := range slices.Clone(order) {
				if x%mod == 0 {
					del(x)
				}
			}
			if removed != before-len(order) {
				t.Fatalf("RemoveFunc removed %d, want %d", removed, before-len(order))
			}
		case step%97 == 0:
			m.Shrink()
		}

		checkInvariants(t, m)
		if got := keys(m); !slices.Equal(got, order) {
			t.Fatalf("step %d: order %v, want %v", step, got, order)
		}
		if m.Len() != len(order) {
			t.Fatalf("Len=%d, want %d", m.Len(), len(order))
		}
		for k, v := range vals {
			if got, ok := m.Get(k); !ok || got != v {
				t.Fatalf("Get(%d) = %v %v, want %v", k, got, ok, v)
			}
		}
	}
}

func TestClone(t *testing.T) {
	m := fill(40)
	for k := 1; k <= 10; k++ {
		m.Remove(k)
	}
	c := m.Clone()
	checkInvariants(t, c)
	if c.holes != 0 || c.Cap() != c.Len() {
		t.Fatalf("clone holes=%d Cap=%d Len=%d", c.holes, c.Cap(), c.Len())
	}
	if !slices.Equal(keys(c), keys(m)) {
		t.Fatal("clone order differs")
	}

	c.Set(1000, 1)
	c.Remove(11)
	c.MoveToBack(12)
	checkInvariants(t, c)
	checkInvariants(t, m)
	if m.Contains(1000) || !m.Contains(11) || m.Len() != 30 {
		t.Fatal("mutating the clone changed the original")
	}

	var nilMap *Map[int, int]
	n := nilMap.Clone()
	if n == nil || !n.IsEmpty() {
		t.Fatal("Clone of nil must be an empty map")
	}
	n.Set(1, 1) // usable
	checkInvariants(t, n)
}

func TestShrink(t *testing.T) {
	m := NewWithCap[int, int](100)
	for k := 1; k <= 10; k++ {
		m.Set(k, k)
	}
	m.Remove(2)
	m.Remove(1)
	m.Shrink()
	checkInvariants(t, m)
	if m.Cap() != m.Len() || m.Len() != 8 || m.holes != 0 {
		t.Fatalf("Len=%d Cap=%d holes=%d", m.Len(), m.Cap(), m.holes)
	}
	if got, want := keys(m), []int{3, 4, 5, 6, 7, 8, 9, 10}; !slices.Equal(got, want) {
		t.Fatalf("Shrink lost data: %v", got)
	}
	for k := 3; k <= 10; k++ {
		if v, ok := m.Get(k); !ok || v != k {
			t.Fatalf("Get(%d) = %v %v after Shrink", k, v, ok)
		}
	}
	m.Shrink() // already tight
	checkInvariants(t, m)
	m.Set(99, 99)
	checkInvariants(t, m)
}

func TestGrowClip(t *testing.T) {
	m := New[int, int]()
	m.Grow(50)
	if m.Cap() < 50 {
		t.Fatalf("Grow(50): Cap=%d", m.Cap())
	}
	m.Grow(0)
	m.Grow(-1)
	m.Set(1, 1)
	m.Set(2, 2)
	m.Clip()
	if m.Cap() != 2 {
		t.Fatalf("Clip: Cap=%d", m.Cap())
	}
	m.Set(3, 3)
	checkInvariants(t, m)
}

func TestResetClear(t *testing.T) {
	m := NewWithCap[int, int](8)
	m.Set(1, 1)
	m.Set(2, 2)
	m.Remove(1)

	m.Reset()
	if m.Len() != 0 || m.Cap() != 8 || m.Contains(2) {
		t.Fatalf("after Reset: Len=%d Cap=%d", m.Len(), m.Cap())
	}
	checkInvariants(t, m)
	m.Set(3, 3)
	checkInvariants(t, m)

	m.Clear()
	if m.Len() != 0 || m.Cap() != 0 || m.Contains(3) {
		t.Fatalf("after Clear: Len=%d Cap=%d", m.Len(), m.Cap())
	}
	checkInvariants(t, m)
	m.Set(4, 4)
	if k, _, _ := m.First(); k != 4 {
		t.Fatal("map unusable after Clear")
	}
}

func TestNilReceiver(t *testing.T) {
	var m *Map[string, int]

	if m.Len() != 0 || m.Cap() != 0 || !m.IsEmpty() || m.Contains("a") {
		t.Fatal("nil map must read as empty")
	}
	if _, ok := m.Get("a"); ok {
		t.Fatal("Get on nil")
	}
	if _, _, ok := m.First(); ok {
		t.Fatal("First on nil")
	}
	if _, _, ok := m.Last(); ok {
		t.Fatal("Last on nil")
	}
	if _, ok := m.Remove("a"); ok {
		t.Fatal("Remove on nil")
	}
	if m.RemoveFunc(func(string, int) bool { return true }) != 0 {
		t.Fatal("RemoveFunc on nil")
	}
	if _, _, ok := m.PopFirst(); ok {
		t.Fatal("PopFirst on nil")
	}
	if _, _, ok := m.PopLast(); ok {
		t.Fatal("PopLast on nil")
	}
	if m.MoveToBack("a") {
		t.Fatal("MoveToBack on nil")
	}
	for range m.All() {
		t.Fatal("All on nil yielded")
	}
	for range m.Backward() {
		t.Fatal("Backward on nil yielded")
	}
	for range m.Keys() {
		t.Fatal("Keys on nil yielded")
	}
	for range m.Values() {
		t.Fatal("Values on nil yielded")
	}

	for name, f := range map[string]func(){
		"Set":    func() { m.Set("a", 1) },
		"Grow":   func() { m.Grow(1) },
		"Clear":  func() { m.Clear() },
		"Reset":  func() { m.Reset() },
		"Shrink": func() { m.Shrink() },
		"Clip":   func() { m.Clip() },
	} {
		expectPanic(t, name+" on nil receiver", f)
	}
}

func TestZeroValue(t *testing.T) {
	var m Map[string, int]

	if !m.IsEmpty() || m.Contains("a") {
		t.Fatal("zero value must read as empty")
	}
	if _, _, ok := m.First(); ok {
		t.Fatal("First on zero value")
	}
	if _, _, ok := m.PopFirst(); ok {
		t.Fatal("PopFirst on zero value")
	}
	if _, ok := m.Remove("a"); ok {
		t.Fatal("Remove on zero value")
	}
	m.Reset()
	m.Clip()
	m.Shrink()
	m.Grow(4)

	var z Map[string, int]
	if !z.Set("a", 1) { // Set allocates the index
		t.Fatal("Set on zero value")
	}
	if v, ok := z.Get("a"); !ok || v != 1 {
		t.Fatal("Get after Set on zero value")
	}
	checkInvariants(t, &z)
}
