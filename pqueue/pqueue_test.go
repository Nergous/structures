package pqueue

import (
	"cmp"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
)

// checkInvariants verifies heap order, that every key's recorded position
// matches the backing array, and that the index holds no stale keys.
func checkInvariants[K comparable, P any](t *testing.T, q *PriorityQueue[K, P]) {
	t.Helper()
	for i := 1; i < len(q.data); i++ {
		if p := (i - 1) / 2; q.compare(q.data[p], q.data[i]) > 0 {
			t.Fatalf("heap order broken: parent %d > child %d", p, i)
		}
	}
	if len(q.pos) != len(q.data) {
		t.Fatalf("index has %d keys, heap has %d", len(q.pos), len(q.data))
	}
	for i, e := range q.data {
		if got, ok := q.pos[e.key]; !ok || got != i {
			t.Fatalf("key %v at index %d, index records %d (present=%v)", e.key, i, got, ok)
		}
	}
}

func popAll[K comparable, P any](q *PriorityQueue[K, P]) []Item[K, P] {
	var out []Item[K, P]
	for {
		k, p, ok := q.Pop()
		if !ok {
			return out
		}
		out = append(out, Item[K, P]{Key: k, Priority: p})
	}
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

func TestNew_MinOrder(t *testing.T) {
	q := New[string, int]()
	q.Push("c", 3)
	q.Push("a", 1)
	q.Push("b", 2)
	checkInvariants(t, q)

	got := popAll(q)
	want := []Item[string, int]{{"a", 1}, {"b", 2}, {"c", 3}}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestNewMax_MaxOrder(t *testing.T) {
	q := NewMax[string, int]()
	q.Push("low", 1)
	q.Push("high", 9)
	q.Push("mid", 5)

	k, p, ok := q.Peek()
	if !ok || k != "high" || p != 9 {
		t.Fatalf("Peek = %v %v %v, want high 9 true", k, p, ok)
	}
	got := popAll(q)
	want := []Item[string, int]{{"high", 9}, {"mid", 5}, {"low", 1}}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestNewFunc_CustomComparator(t *testing.T) {
	type deadline struct{ day, hour int }
	q := NewFunc[string](func(a, b deadline) int {
		if c := cmp.Compare(a.day, b.day); c != 0 {
			return c
		}
		return cmp.Compare(a.hour, b.hour)
	})
	q.Push("late", deadline{2, 9})
	q.Push("early", deadline{1, 18})
	q.Push("earliest", deadline{1, 8})

	var keys []string
	for _, it := range popAll(q) {
		keys = append(keys, it.Key)
	}
	if want := []string{"earliest", "early", "late"}; !slices.Equal(keys, want) {
		t.Fatalf("got %v, want %v", keys, want)
	}
}

func TestNewFunc_NilComparatorPanics(t *testing.T) {
	expectPanic(t, "nil comparator", func() { NewFunc[string, int](nil) })
	expectPanic(t, "nil comparator", func() { NewFuncWithCap[string, int](4, nil) })
}

func TestNewWithCap(t *testing.T) {
	for _, q := range []*PriorityQueue[int, int]{NewWithCap[int, int](16), NewMaxWithCap[int, int](16), NewFuncWithCap[int](16, cmp.Compare[int])} {
		if q.Cap() < 16 || q.Len() != 0 {
			t.Fatalf("Len=%d Cap=%d, want 0 and >= 16", q.Len(), q.Cap())
		}
	}
	if q := NewWithCap[int, int](-3); q.Cap() != 0 {
		t.Fatalf("negative capacity: Cap=%d, want 0", q.Cap())
	}
}

func TestPush_ReportsNewKey(t *testing.T) {
	q := New[string, int]()
	if !q.Push("a", 5) {
		t.Fatal("first Push reported existing key")
	}
	if q.Push("a", 1) {
		t.Fatal("second Push reported new key")
	}
	if q.Len() != 1 {
		t.Fatalf("Len = %d, want 1", q.Len())
	}
	if p, ok := q.Priority("a"); !ok || p != 1 {
		t.Fatalf("Priority = %v %v, want 1 true", p, ok)
	}
}

func TestPush_UpdateMovesBothWays(t *testing.T) {
	q := New[string, int]()
	for i, k := range []string{"a", "b", "c", "d", "e"} {
		q.Push(k, (i+1)*10)
	}

	q.Push("e", 1) // decrease: e goes to the top
	checkInvariants(t, q)
	if k, _, _ := q.Peek(); k != "e" {
		t.Fatalf("after decrease, top = %q, want e", k)
	}

	q.Push("e", 100) // increase: e sinks to the bottom
	checkInvariants(t, q)
	got := popAll(q)
	if got[len(got)-1] != (Item[string, int]{"e", 100}) {
		t.Fatalf("after increase, last = %v, want e 100", got[len(got)-1])
	}
}

func TestStable_EqualPrioritiesFIFO(t *testing.T) {
	for _, q := range []*PriorityQueue[int, int]{New[int, int](), NewMax[int, int]()} {
		for k := range 50 {
			q.Push(k, k%3)
		}
		prev := map[int]int{} // priority -> last key seen
		for _, it := range popAll(q) {
			if last, ok := prev[it.Priority]; ok && it.Key < last {
				t.Fatalf("priority %d: key %d came after %d", it.Priority, it.Key, last)
			}
			prev[it.Priority] = it.Key
		}
	}
}

func TestStable_UpdateKeepsPlace(t *testing.T) {
	q := New[string, int]()
	q.Push("first", 5)
	q.Push("second", 5)
	q.Push("third", 9)

	q.Push("first", 5) // same priority: must stay ahead of second
	q.Push("third", 5) // joins the 5s, but was pushed last
	q.Push("first", 7) // leave the 5s ...
	q.Push("first", 5) // ... and come back: original place is kept

	var keys []string
	for _, it := range popAll(q) {
		keys = append(keys, it.Key)
	}
	if want := []string{"first", "second", "third"}; !slices.Equal(keys, want) {
		t.Fatalf("got %v, want %v", keys, want)
	}
}

func TestPop_Empty(t *testing.T) {
	q := New[string, int]()
	if k, p, ok := q.Pop(); ok || k != "" || p != 0 {
		t.Fatalf("Pop on empty = %q %v %v", k, p, ok)
	}
	if k, p, ok := q.Peek(); ok || k != "" || p != 0 {
		t.Fatalf("Peek on empty = %q %v %v", k, p, ok)
	}
}

func TestPop_ZeroesVacatedSlot(t *testing.T) {
	q := New[*int, int]()
	a, b := new(int), new(int)
	q.Push(a, 1)
	q.Push(b, 2)
	q.Pop()
	q.Pop()
	full := q.data[:cap(q.data)]
	for i, e := range full {
		if e != (entry[*int, int]{}) {
			t.Fatalf("slot %d not zeroed: %+v", i, e)
		}
	}
}

func TestPopN(t *testing.T) {
	q := New[int, int]()
	for k := range 5 {
		q.Push(k, 10-k)
	}
	got := q.PopN(2)
	want := []Item[int, int]{{4, 6}, {3, 7}}
	if !slices.Equal(got, want) {
		t.Fatalf("PopN(2) = %v, want %v", got, want)
	}
	checkInvariants(t, q)
	if got := q.PopN(10); len(got) != 3 || !q.IsEmpty() {
		t.Fatalf("PopN(10) = %v, Len after = %d", got, q.Len())
	}
	if q.PopN(1) != nil || q.PopN(0) != nil || q.PopN(-1) != nil {
		t.Fatal("PopN on empty or n <= 0 must return nil")
	}
}

func TestRemove(t *testing.T) {
	q := New[string, int]()
	q.Push("a", 1)
	q.Push("b", 2)
	q.Push("c", 3)

	if p, ok := q.Remove("b"); !ok || p != 2 {
		t.Fatalf("Remove(b) = %v %v, want 2 true", p, ok)
	}
	checkInvariants(t, q)
	if q.Contains("b") {
		t.Fatal("b still present after Remove")
	}
	if p, ok := q.Remove("b"); ok || p != 0 {
		t.Fatalf("second Remove(b) = %v %v, want 0 false", p, ok)
	}
	if _, ok := q.Remove("zzz"); ok {
		t.Fatal("Remove of missing key reported true")
	}
	if p, ok := q.Remove("a"); !ok || p != 1 { // the root
		t.Fatalf("Remove(a) = %v %v", p, ok)
	}
	checkInvariants(t, q)
	if got := popAll(q); !slices.Equal(got, []Item[string, int]{{"c", 3}}) {
		t.Fatalf("remaining = %v", got)
	}
}

func TestPriorityContains(t *testing.T) {
	q := New[string, int]()
	q.Push("a", 4)
	if p, ok := q.Priority("a"); !ok || p != 4 {
		t.Fatalf("Priority(a) = %v %v", p, ok)
	}
	if p, ok := q.Priority("x"); ok || p != 0 {
		t.Fatalf("Priority(x) = %v %v", p, ok)
	}
	if !q.Contains("a") || q.Contains("x") {
		t.Fatal("Contains mismatch")
	}
	q.Pop()
	if q.Contains("a") {
		t.Fatal("popped key still reported present")
	}
	if q.Push("a", 1) != true {
		t.Fatal("re-pushing a popped key must report it as new")
	}
}

// TestRandomOps runs a long random mix of pushes, priority updates, pops, and
// removals against a reference map and checks the invariants after each step.
func TestRandomOps(t *testing.T) {
	r := rand.New(rand.NewPCG(11, 12))
	q := New[int, int]()
	ref := map[int]int{}

	for range 20000 {
		k := r.IntN(200)
		switch op := r.IntN(10); {
		case op < 5:
			_, existed := ref[k]
			if isNew := q.Push(k, r.IntN(100)); isNew == existed {
				t.Fatalf("Push(%d) new=%v, existed=%v", k, isNew, existed)
			}
			ref[k], _ = q.Priority(k)
		case op < 8:
			k, p, ok := q.Pop()
			if ok != (len(ref) > 0) {
				t.Fatalf("Pop ok=%v with %d keys", ok, len(ref))
			}
			if ok {
				for _, rp := range ref {
					if rp < p {
						t.Fatalf("Pop returned priority %d, but %d is queued", p, rp)
					}
				}
				if ref[k] != p {
					t.Fatalf("Pop(%d) priority %d, want %d", k, p, ref[k])
				}
				delete(ref, k)
			}
		default:
			p, ok := q.Remove(k)
			rp, want := ref[k]
			if ok != want || (ok && p != rp) {
				t.Fatalf("Remove(%d) = %d %v, want %d %v", k, p, ok, rp, want)
			}
			delete(ref, k)
		}
		checkInvariants(t, q)
		if q.Len() != len(ref) {
			t.Fatalf("Len = %d, want %d", q.Len(), len(ref))
		}
	}
}

func TestAllSliceSorted(t *testing.T) {
	q := New[string, int]()
	q.Push("c", 3)
	q.Push("a", 1)
	q.Push("b", 1)
	q.Push("d", 0)

	var all []Item[string, int]
	for k, p := range q.All() {
		all = append(all, Item[string, int]{k, p})
	}
	if !slices.Equal(all, q.Slice()) {
		t.Fatalf("All %v != Slice %v", all, q.Slice())
	}
	if all[0] != (Item[string, int]{"d", 0}) {
		t.Fatalf("layout order must start with the least item, got %v", all[0])
	}

	want := []Item[string, int]{{"d", 0}, {"a", 1}, {"b", 1}, {"c", 3}}
	if got := q.Sorted(); !slices.Equal(got, want) {
		t.Fatalf("Sorted = %v, want %v", got, want)
	}
	if q.Len() != 4 {
		t.Fatal("Sorted consumed the queue")
	}
	checkInvariants(t, q)

	n := 0
	for range q.All() {
		n++
		break
	}
	if n != 1 {
		t.Fatal("early break did not stop iteration")
	}
}

func TestSlice_Independent(t *testing.T) {
	q := New[string, int]()
	q.Push("a", 1)
	s := q.Slice()
	s[0].Priority = 99
	if p, _ := q.Priority("a"); p != 1 {
		t.Fatal("mutating Slice result changed the queue")
	}
}

func TestClone_Independent(t *testing.T) {
	q := New[int, int]()
	for k := range 20 {
		q.Push(k, 20-k)
	}
	c := q.Clone()
	checkInvariants(t, c)
	if &c.data[0] == &q.data[0] {
		t.Fatal("clone shares the backing array")
	}
	if c.Cap() != c.Len() {
		t.Fatalf("clone Cap=%d, want Len=%d", c.Cap(), c.Len())
	}

	// Mutating the clone must never touch the original's index.
	for k := range 10 {
		c.Push(k, -k)
	}
	c.Pop()
	c.Remove(15)
	checkInvariants(t, c)
	checkInvariants(t, q)

	if q.Len() != 20 {
		t.Fatalf("original Len = %d, want 20", q.Len())
	}
	if p, _ := q.Priority(3); p != 17 {
		t.Fatalf("original priority of 3 = %d, want 17", p)
	}
}

func TestClone_KeepsStableOrder(t *testing.T) {
	q := New[string, int]()
	q.Push("x", 1)
	q.Push("y", 1)
	c := q.Clone()
	c.Push("z", 1)
	var keys []string
	for _, it := range popAll(c) {
		keys = append(keys, it.Key)
	}
	if want := []string{"x", "y", "z"}; !slices.Equal(keys, want) {
		t.Fatalf("got %v, want %v", keys, want)
	}
}

func TestResetClear(t *testing.T) {
	q := NewWithCap[string, int](8)
	q.Push("a", 1)
	q.Push("b", 2)

	q.Reset()
	if q.Len() != 0 || q.Cap() != 8 || q.Contains("a") {
		t.Fatalf("after Reset: Len=%d Cap=%d Contains(a)=%v", q.Len(), q.Cap(), q.Contains("a"))
	}
	for i, e := range q.data[:cap(q.data)] {
		if e != (entry[string, int]{}) {
			t.Fatalf("Reset left slot %d: %+v", i, e)
		}
	}
	q.Push("c", 3)
	checkInvariants(t, q)

	q.Clear()
	if q.Len() != 0 || q.Cap() != 0 || q.Contains("c") {
		t.Fatalf("after Clear: Len=%d Cap=%d", q.Len(), q.Cap())
	}
	q.Push("d", 4)
	checkInvariants(t, q)
	if k, _, _ := q.Pop(); k != "d" {
		t.Fatal("queue unusable after Clear")
	}
}

func TestCapacityControls(t *testing.T) {
	q := New[int, int]()
	q.Grow(100)
	if q.Cap() < 100 {
		t.Fatalf("Grow(100): Cap=%d", q.Cap())
	}
	q.Grow(0)
	q.Grow(-1)
	for k := range 5 {
		q.Push(k, k)
	}

	q.Clip()
	if q.Cap() != q.Len() {
		t.Fatalf("Clip: Len=%d Cap=%d", q.Len(), q.Cap())
	}
	checkInvariants(t, q)

	q.Grow(50)
	q.Shrink()
	if q.Cap() != q.Len() {
		t.Fatalf("Shrink: Len=%d Cap=%d", q.Len(), q.Cap())
	}
	checkInvariants(t, q)
	q.Shrink() // already tight: still valid
	checkInvariants(t, q)

	q.Push(99, -1)
	if k, _, _ := q.Pop(); k != 99 {
		t.Fatal("queue unusable after Shrink")
	}
}

func TestNilReceiver(t *testing.T) {
	var q *PriorityQueue[string, int]

	if q.Len() != 0 || q.Cap() != 0 || !q.IsEmpty() {
		t.Fatal("nil queue must report empty")
	}
	if _, _, ok := q.Peek(); ok {
		t.Fatal("Peek on nil")
	}
	if _, _, ok := q.Pop(); ok {
		t.Fatal("Pop on nil")
	}
	if q.PopN(3) != nil || q.Slice() != nil || q.Sorted() != nil || q.Clone() != nil {
		t.Fatal("nil queue snapshots must be nil")
	}
	if _, ok := q.Remove("a"); ok {
		t.Fatal("Remove on nil")
	}
	if _, ok := q.Priority("a"); ok || q.Contains("a") {
		t.Fatal("lookup on nil")
	}
	for range q.All() {
		t.Fatal("All on nil yielded")
	}

	for name, f := range map[string]func(){
		"Push":   func() { q.Push("a", 1) },
		"Grow":   func() { q.Grow(1) },
		"Clear":  func() { q.Clear() },
		"Reset":  func() { q.Reset() },
		"Shrink": func() { q.Shrink() },
		"Clip":   func() { q.Clip() },
	} {
		expectPanic(t, name+" on nil receiver", f)
	}
}

func TestZeroValue(t *testing.T) {
	var q PriorityQueue[string, int]

	if !q.IsEmpty() || q.Contains("a") {
		t.Fatal("zero value must read as empty")
	}
	if _, _, ok := q.Pop(); ok {
		t.Fatal("Pop on zero value")
	}
	if _, ok := q.Remove("a"); ok {
		t.Fatal("Remove on zero value")
	}
	expectPanic(t, "zero-value PriorityQueue", func() { q.Push("a", 1) })

	q.Reset()
	q.Clip()
	q.Shrink()
	q.Grow(4)
	if c := q.Clone(); c == nil || !c.IsEmpty() {
		t.Fatal("Clone of zero value must be an empty queue")
	}
}
