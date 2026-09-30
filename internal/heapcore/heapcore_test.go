package heapcore

import (
	"cmp"
	"math/rand/v2"
	"slices"
	"testing"
)

type item struct {
	v  int
	id int
}

func cmpItem(a, b item) int { return cmp.Compare(a.v, b.v) }

func checkHeap[T any](t *testing.T, s []T, compare func(a, b T) int) {
	t.Helper()
	for i := 1; i < len(s); i++ {
		if p := (i - 1) / 2; compare(s[p], s[i]) > 0 {
			t.Fatalf("heap order broken: parent %d > child %d", p, i)
		}
	}
}

// tracker records every element's position through the moved callback and
// verifies that the record matches the slice.
type tracker struct {
	pos map[int]int // id -> index
}

func newTracker(s []item) *tracker {
	tr := &tracker{pos: make(map[int]int, len(s))}
	for i, it := range s {
		tr.pos[it.id] = i
	}
	return tr
}

func (tr *tracker) moved(s []item, i int) { tr.pos[s[i].id] = i }

func (tr *tracker) check(t *testing.T, s []item) {
	t.Helper()
	if len(tr.pos) != len(s) {
		t.Fatalf("tracked %d elements, slice has %d", len(tr.pos), len(s))
	}
	for i, it := range s {
		if got := tr.pos[it.id]; got != i {
			t.Fatalf("element id=%d at index %d, tracked at %d", it.id, i, got)
		}
	}
}

func randItems(r *rand.Rand, n int) []item {
	s := make([]item, n)
	for i := range s {
		s[i] = item{v: r.IntN(50), id: i}
	}
	return s
}

func TestHeapify(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for n := range 40 {
		s := randItems(r, n)
		Heapify(s, cmpItem)
		checkHeap(t, s, cmpItem)
	}
}

func TestUpDown_EmptyAndSingle(t *testing.T) {
	var empty []int
	Heapify(empty, cmp.Compare[int])
	if Down(empty, 0, cmp.Compare[int]) {
		t.Fatal("Down on empty slice reported a move")
	}
	one := []int{7}
	Up(one, 0, cmp.Compare[int])
	if Down(one, 0, cmp.Compare[int]) || one[0] != 7 {
		t.Fatal("single element moved")
	}
}

func TestDown_ReportsMove(t *testing.T) {
	s := []int{5, 1, 2}
	if !Down(s, 0, cmp.Compare[int]) {
		t.Fatal("Down did not report a move")
	}
	if Down(s, 0, cmp.Compare[int]) {
		t.Fatal("Down reported a move on a valid heap")
	}
}

func TestFix(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 4))
	for range 500 {
		s := randItems(r, 1+r.IntN(30))
		Heapify(s, cmpItem)
		i := r.IntN(len(s))
		s[i].v = r.IntN(50) - 10
		Fix(s, i, cmpItem)
		checkHeap(t, s, cmpItem)
	}
}

func TestRemoveAt(t *testing.T) {
	r := rand.New(rand.NewPCG(5, 6))
	for range 500 {
		s := randItems(r, 1+r.IntN(30))
		Heapify(s, cmpItem)
		want := slices.Clone(s)
		i := r.IntN(len(s))
		removed := s[i]
		want = slices.DeleteFunc(want, func(it item) bool { return it.id == removed.id })

		full := s[:cap(s)]
		s = RemoveAt(s, i, cmpItem)
		checkHeap(t, s, cmpItem)
		if full[len(s)] != (item{}) {
			t.Fatal("vacated tail slot not zeroed")
		}
		slices.SortFunc(want, func(a, b item) int { return cmp.Compare(a.id, b.id) })
		got := slices.SortedFunc(slices.Values(s), func(a, b item) int { return cmp.Compare(a.id, b.id) })
		if !slices.Equal(got, want) {
			t.Fatalf("RemoveAt(%d) left %v, want %v", i, got, want)
		}
	}
}

func TestRemoveAt_Last(t *testing.T) {
	s := []int{1, 2, 3}
	s = RemoveAt(s, 2, cmp.Compare[int])
	if !slices.Equal(s, []int{1, 2}) {
		t.Fatalf("got %v", s)
	}
	s = RemoveAt(s[:1], 0, cmp.Compare[int])
	if len(s) != 0 {
		t.Fatalf("got %v", s)
	}
}

// TestMoved_Tracking runs a random mix of operations through the Moved
// variants and checks after each one that heap order holds and the moved
// callback kept every element's recorded position accurate.
func TestMoved_Tracking(t *testing.T) {
	r := rand.New(rand.NewPCG(7, 8))
	s := randItems(r, 64)
	tr := newTracker(s)
	HeapifyMoved(s, cmpItem, tr.moved)
	checkHeap(t, s, cmpItem)
	tr.check(t, s)

	nextID := len(s)
	for range 5000 {
		switch op := r.IntN(3); {
		case op == 0 || len(s) == 0: // push
			s = append(s, item{v: r.IntN(50), id: nextID})
			tr.pos[nextID] = len(s) - 1
			nextID++
			UpMoved(s, len(s)-1, cmpItem, tr.moved)
		case op == 1: // update priority
			i := r.IntN(len(s))
			s[i].v = r.IntN(60) - 5
			FixMoved(s, i, cmpItem, tr.moved)
		default: // remove
			i := r.IntN(len(s))
			delete(tr.pos, s[i].id)
			s = RemoveAtMoved(s, i, cmpItem, tr.moved)
		}
		checkHeap(t, s, cmpItem)
		tr.check(t, s)
	}
}

// TestMoved_MatchesPlain checks that the Moved variants reorder elements
// exactly like the plain ones.
func TestMoved_MatchesPlain(t *testing.T) {
	r := rand.New(rand.NewPCG(9, 10))
	noop := func([]item, int) {}
	for range 200 {
		a := randItems(r, r.IntN(40))
		b := slices.Clone(a)
		Heapify(a, cmpItem)
		HeapifyMoved(b, cmpItem, noop)
		if len(a) > 0 {
			i := r.IntN(len(a))
			a[i].v, b[i].v = -1, -1
			Fix(a, i, cmpItem)
			FixMoved(b, i, cmpItem, noop)
			j := r.IntN(len(a))
			a = RemoveAt(a, j, cmpItem)
			b = RemoveAtMoved(b, j, cmpItem, noop)
		}
		if !slices.Equal(a, b) {
			t.Fatalf("plain %v != moved %v", a, b)
		}
	}
}
