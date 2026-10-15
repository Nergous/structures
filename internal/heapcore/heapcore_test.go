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

func cmpItem(a, b item) int {
	return cmp.Compare(a.v, b.v)
}

func checkHeap[T any](t *testing.T, s []T, compare func(a, b T) int) {
	t.Helper()

	for i := 1; i < len(s); i++ {
		if p := (i - 1) / 2; compare(s[p], s[i]) > 0 {
			t.Fatalf("compare(s[%d], s[%d]) > 0, want <= 0 (heap order broken)", p, i)
		}
	}
}

// tracker records every element's position through the moved callback and
// verifies that the record matches the slice.
type tracker struct {
	pos map[int]int // id -> index
}

func newTracker(s []item) *tracker {
	tr := &tracker{
		pos: make(map[int]int, len(s)),
	}

	for i, it := range s {
		tr.pos[it.id] = i
	}

	return tr
}

func (tr *tracker) moved(s []item, i int) {
	tr.pos[s[i].id] = i
}

func (tr *tracker) check(t *testing.T, s []item) {
	t.Helper()

	if got := len(tr.pos); got != len(s) {
		t.Fatalf("len(pos) = %d, want %d", got, len(s))
	}

	for i, it := range s {
		if got := tr.pos[it.id]; got != i {
			t.Fatalf("pos[%d] = %d, want %d", it.id, got, i)
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

	if got := Down(empty, 0, cmp.Compare[int]); got {
		t.Errorf("Down() on empty = %v, want false", got)
	}

	one := []int{7}
	Up(one, 0, cmp.Compare[int])

	if got := Down(one, 0, cmp.Compare[int]); got {
		t.Errorf("Down() on single = %v, want false", got)
	}

	if got := one[0]; got != 7 {
		t.Errorf("one[0] after Up(), Down() = %d, want 7", got)
	}
}

func TestDown_ReportsMove(t *testing.T) {
	s := []int{5, 1, 2}

	if got := Down(s, 0, cmp.Compare[int]); !got {
		t.Fatalf("Down() = %v, want true", got)
	}

	if got := Down(s, 0, cmp.Compare[int]); got {
		t.Errorf("Down() on valid heap = %v, want false", got)
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

		if got := full[len(s)]; got != (item{}) {
			t.Fatalf("data[%d] after RemoveAt(%d) = %v, want zero item", len(s), i, got)
		}

		slices.SortFunc(want, func(a, b item) int { return cmp.Compare(a.id, b.id) })
		got := slices.SortedFunc(slices.Values(s), func(a, b item) int { return cmp.Compare(a.id, b.id) })
		if !slices.Equal(got, want) {
			t.Fatalf("RemoveAt(%d) = %v, want %v (sorted by id)", i, got, want)
		}
	}
}

func TestRemoveAt_Last(t *testing.T) {
	s := []int{1, 2, 3}
	s = RemoveAt(s, 2, cmp.Compare[int])

	if want := []int{1, 2}; !slices.Equal(s, want) {
		t.Fatalf("RemoveAt(s, 2) = %v, want %v", s, want)
	}

	s = RemoveAt(s[:1], 0, cmp.Compare[int])

	if got := len(s); got != 0 {
		t.Fatalf("len(RemoveAt(s[:1], 0)) = %d, want 0", got)
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

		if !slices.Equal(b, a) {
			t.Fatalf("Moved variants = %v, want %v (same as plain)", b, a)
		}
	}
}
