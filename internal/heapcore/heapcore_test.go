package heapcore

import (
	"cmp"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Nergous/structures/internal/shared/sharedtest"
)

// item is a heap element with a priority v and a unique id that identifies it
// regardless of where it sits in the slice.
type item struct {
	v  int
	id int
}

// cmpItem orders items by priority only, so items with equal v tie.
func cmpItem(a, b item) int {
	return cmp.Compare(a.v, b.v)
}

// cmpID orders items by id; it sorts heaps into a comparable canonical order.
func cmpID(a, b item) int {
	return cmp.Compare(a.id, b.id)
}

// randItems returns n items with random priorities in [0, 50) and ids 0..n-1.
func randItems(r *rand.Rand, n int) []item {
	s := make([]item, n)
	for i := range s {
		s[i] = item{v: r.IntN(50), id: i}
	}

	return s
}

// tracker records every element's position through the moved callback and
// verifies that the record matches the slice.
type tracker struct {
	pos map[int]int // id -> index
}

// newTracker returns a tracker that records the current position of every
// item in s.
func newTracker(s []item) *tracker {
	tr := &tracker{
		pos: make(map[int]int, len(s)),
	}

	for i, it := range s {
		tr.pos[it.id] = i
	}

	return tr
}

// moved is the callback passed to the Moved variants.
func (tr *tracker) moved(s []item, i int) {
	tr.pos[s[i].id] = i
}

// assertPositions fails the test unless the recorded position of every item
// in s equals its index and no stale ids remain.
func (tr *tracker) assertPositions(t *testing.T, s []item) {
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

func TestHeapify(t *testing.T) {
	t.Run("random slices satisfy heap order", func(t *testing.T) {
		r := rand.New(rand.NewPCG(1, 2))
		for n := range 40 {
			s := randItems(r, n)

			Heapify(s, cmpItem)

			sharedtest.Heap(t, s, cmpItem)
		}
	})

	t.Run("empty and single are no-ops", func(t *testing.T) {
		var empty []int
		Heapify(empty, cmp.Compare[int])

		one := []int{7}
		Heapify(one, cmp.Compare[int])

		if got := one[0]; got != 7 {
			t.Errorf("one[0] after Heapify(one) = %d, want 7", got)
		}
	})
}

func TestDown(t *testing.T) {
	t.Run("reports move", func(t *testing.T) {
		s := []int{5, 1, 2}

		if got := Down(s, 0, cmp.Compare[int]); !got {
			t.Errorf("Down(s, 0) = %v, want true", got)
		}

		sharedtest.Heap(t, s, cmp.Compare[int])
	})

	t.Run("valid heap does not move", func(t *testing.T) {
		s := []int{1, 5, 2}

		if got := Down(s, 0, cmp.Compare[int]); got {
			t.Errorf("Down(s, 0) on valid heap = %v, want false", got)
		}

		if want := []int{1, 5, 2}; !slices.Equal(s, want) {
			t.Errorf("s after Down(s, 0) = %v, want %v", s, want)
		}
	})

	t.Run("single element does not move", func(t *testing.T) {
		one := []int{7}

		if got := Down(one, 0, cmp.Compare[int]); got {
			t.Errorf("Down(one, 0) = %v, want false", got)
		}
	})
}

func TestUp(t *testing.T) {
	t.Run("new least leaf rises to the root", func(t *testing.T) {
		s := []int{1, 2, 3, 0}

		Up(s, 3, cmp.Compare[int])

		if got := s[0]; got != 0 {
			t.Errorf("s[0] after Up(s, 3) = %d, want 0", got)
		}

		sharedtest.Heap(t, s, cmp.Compare[int])
	})

	t.Run("single element stays", func(t *testing.T) {
		one := []int{7}

		Up(one, 0, cmp.Compare[int])

		if got := one[0]; got != 7 {
			t.Errorf("one[0] after Up(one, 0) = %d, want 7", got)
		}
	})
}

func TestFix(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 4))
	for range 500 {
		s := randItems(r, 1+r.IntN(30))
		Heapify(s, cmpItem)
		i := r.IntN(len(s))
		s[i].v = r.IntN(50) - 10

		Fix(s, i, cmpItem)

		sharedtest.Heap(t, s, cmpItem)
	}
}

func TestRemoveAt(t *testing.T) {
	t.Run("random removals keep heap order", func(t *testing.T) {
		r := rand.New(rand.NewPCG(5, 6))
		for range 500 {
			s := randItems(r, 1+r.IntN(30))
			Heapify(s, cmpItem)
			i := r.IntN(len(s))
			removed := s[i]
			want := slices.DeleteFunc(slices.Clone(s), func(it item) bool {
				return it.id == removed.id
			})
			slices.SortFunc(want, cmpID)
			full := s[:cap(s)]

			s = RemoveAt(s, i, cmpItem)

			sharedtest.Heap(t, s, cmpItem)

			if got := full[len(s)]; got != (item{}) {
				t.Fatalf("full[%d] after RemoveAt(%d) = %v, want zero item",
					len(s), i, got)
			}

			got := slices.SortedFunc(slices.Values(s), cmpID)
			if !slices.Equal(got, want) {
				t.Fatalf("RemoveAt(%d) = %v, want %v (sorted by id)",
					i, got, want)
			}
		}
	})

	t.Run("last element", func(t *testing.T) {
		got := RemoveAt([]int{1, 2, 3}, 2, cmp.Compare[int])

		if want := []int{1, 2}; !slices.Equal(got, want) {
			t.Errorf("RemoveAt(s, 2) = %v, want %v", got, want)
		}
	})

	t.Run("only element", func(t *testing.T) {
		got := RemoveAt([]int{1}, 0, cmp.Compare[int])

		if got := len(got); got != 0 {
			t.Errorf("len(RemoveAt(s, 0)) = %d, want 0", got)
		}
	})
}

// TestMovedVariants checks the Moved variants against the moved-callback
// contract and against the plain functions they mirror.
func TestMovedVariants(t *testing.T) {
	t.Run("moved tracks every position", func(t *testing.T) {
		r := rand.New(rand.NewPCG(7, 8))
		s := randItems(r, 64)
		tr := newTracker(s)

		HeapifyMoved(s, cmpItem, tr.moved)

		sharedtest.Heap(t, s, cmpItem)
		tr.assertPositions(t, s)

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

			sharedtest.Heap(t, s, cmpItem)
			tr.assertPositions(t, s)
		}
	})

	t.Run("same arrangement as plain variants", func(t *testing.T) {
		r := rand.New(rand.NewPCG(9, 10))
		noop := func([]item, int) {}
		for range 200 {
			a := randItems(r, r.IntN(40))
			b := slices.Clone(a)
			Heapify(a, cmpItem)
			HeapifyMoved(b, cmpItem, noop)

			if len(a) > 0 {
				i := r.IntN(len(a))
				a[i].v = -1
				b[i].v = -1
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
	})
}
