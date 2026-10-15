package pqueue

import (
	"cmp"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Nergous/structures/internal/shared/sharedtest"
)

// checkInvariants verifies heap order, that every key's recorded position
// matches the backing array, and that the index holds no stale keys.
func checkInvariants[K comparable, P any](t *testing.T, q *PriorityQueue[K, P]) {
	t.Helper()

	for i := 1; i < len(q.data); i++ {
		parent := (i - 1) / 2
		if got := q.compare(q.data[parent], q.data[i]); got > 0 {
			t.Fatalf("compare(data[%d], data[%d]) = %d, want <= 0 (heap order broken)", parent, i, got)
		}
	}

	if got, want := len(q.pos), len(q.data); got != want {
		t.Fatalf("len(pos) = %d, want len(data) = %d", got, want)
	}

	for i, e := range q.data {
		if got, ok := q.pos[e.key]; !ok || got != i {
			t.Fatalf("pos[%v] = (%d, %v), want (%d, true)", e.key, got, ok, i)
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

func TestPriorityQueue_New(t *testing.T) {
	q := New[string, int]()
	q.Push("c", 3)
	q.Push("a", 1)
	q.Push("b", 2)

	checkInvariants(t, q)

	want := []Item[string, int]{{"a", 1}, {"b", 2}, {"c", 3}}
	if got := popAll(q); !slices.Equal(got, want) {
		t.Errorf("Pop() order = %v, want %v", got, want)
	}
}

func TestPriorityQueue_NewMax(t *testing.T) {
	q := NewMax[string, int]()
	q.Push("low", 1)
	q.Push("high", 9)
	q.Push("mid", 5)

	if gotKey, gotPriority, ok := q.Peek(); !ok || gotKey != "high" || gotPriority != 9 {
		t.Errorf("Peek() = (%q, %d, %v), want (%q, 9, true)", gotKey, gotPriority, ok, "high")
	}

	want := []Item[string, int]{{"high", 9}, {"mid", 5}, {"low", 1}}
	if got := popAll(q); !slices.Equal(got, want) {
		t.Errorf("Pop() order = %v, want %v", got, want)
	}
}

func TestPriorityQueue_NewFunc(t *testing.T) {
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

	var got []string
	for _, it := range popAll(q) {
		got = append(got, it.Key)
	}

	if want := []string{"earliest", "early", "late"}; !slices.Equal(got, want) {
		t.Errorf("Pop() keys = %v, want %v", got, want)
	}
}

func TestPriorityQueue_NilComparatorPanics(t *testing.T) {
	sharedtest.Panics(t, "NewFunc(nil)", "nil comparator", func() { NewFunc[string, int](nil) })
	sharedtest.Panics(t, "NewFuncWithCap(nil)", "nil comparator", func() { NewFuncWithCap[string, int](4, nil) })
}

func TestPriorityQueue_WithCap(t *testing.T) {
	tests := []struct {
		name string
		q    *PriorityQueue[int, int]
	}{
		{
			name: "NewWithCap",
			q:    NewWithCap[int, int](16),
		},
		{
			name: "NewMaxWithCap",
			q:    NewMaxWithCap[int, int](16),
		},
		{
			name: "NewFuncWithCap",
			q:    NewFuncWithCap[int](16, cmp.Compare[int]),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.q.Len(); got != 0 {
				t.Errorf("Len() = %d, want 0", got)
			}

			if got := tt.q.Cap(); got < 16 {
				t.Errorf("Cap() = %d, want >= 16", got)
			}
		})
	}

	if got := NewWithCap[int, int](-3).Cap(); got != 0 {
		t.Errorf("NewWithCap(-3).Cap() = %d, want 0", got)
	}
}

func TestPriorityQueue_Push(t *testing.T) {
	t.Run("reports new key", func(t *testing.T) {
		q := New[string, int]()

		if got := q.Push("a", 5); !got {
			t.Errorf("Push(a, 5) = %v, want true", got)
		}

		if got := q.Push("a", 1); got {
			t.Errorf("Push(a, 1) on existing = %v, want false", got)
		}

		if got := q.Len(); got != 1 {
			t.Errorf("Len() = %d, want 1", got)
		}

		if got, ok := q.Priority("a"); !ok || got != 1 {
			t.Errorf("Priority(a) = (%d, %v), want (1, true)", got, ok)
		}
	})

	t.Run("update moves both ways", func(t *testing.T) {
		q := New[string, int]()
		for i, k := range []string{"a", "b", "c", "d", "e"} {
			q.Push(k, (i+1)*10)
		}

		q.Push("e", 1) // decrease: e goes to the top

		checkInvariants(t, q)

		if got, _, _ := q.Peek(); got != "e" {
			t.Errorf("Peek() after Push(e, 1) = %q, want %q", got, "e")
		}

		q.Push("e", 100) // increase: e sinks to the bottom

		checkInvariants(t, q)

		all := popAll(q)

		if got, want := all[len(all)-1], (Item[string, int]{"e", 100}); got != want {
			t.Errorf("last Pop() after Push(e, 100) = %v, want %v", got, want)
		}
	})
}

func TestPriorityQueue_Stable(t *testing.T) {
	t.Run("equal priorities come out in push order", func(t *testing.T) {
		tests := []struct {
			name string
			q    *PriorityQueue[int, int]
		}{
			{
				name: "New",
				q:    New[int, int](),
			},
			{
				name: "NewMax",
				q:    NewMax[int, int](),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				for k := range 50 {
					tt.q.Push(k, k%3)
				}

				prev := map[int]int{} // priority -> last key seen
				for _, it := range popAll(tt.q) {
					if last, ok := prev[it.Priority]; ok && it.Key < last {
						t.Fatalf("Pop() key with priority %d = %d, want > %d (push order)", it.Priority, it.Key, last)
					}

					prev[it.Priority] = it.Key
				}
			})
		}
	})

	t.Run("update keeps place", func(t *testing.T) {
		q := New[string, int]()
		q.Push("first", 5)
		q.Push("second", 5)
		q.Push("third", 9)

		q.Push("first", 5) // same priority: must stay ahead of second
		q.Push("third", 5) // joins the 5s, but was pushed last
		q.Push("first", 7) // leave the 5s ...
		q.Push("first", 5) // ... and come back: original place is kept

		var got []string
		for _, it := range popAll(q) {
			got = append(got, it.Key)
		}

		if want := []string{"first", "second", "third"}; !slices.Equal(got, want) {
			t.Errorf("Pop() keys = %v, want %v", got, want)
		}
	})
}

func TestPriorityQueue_Pop(t *testing.T) {
	t.Run("empty returns zero values and false", func(t *testing.T) {
		q := New[string, int]()

		if gotKey, gotPriority, ok := q.Pop(); ok || gotKey != "" || gotPriority != 0 {
			t.Errorf("Pop() on empty = (%q, %d, %v), want (\"\", 0, false)", gotKey, gotPriority, ok)
		}

		if gotKey, gotPriority, ok := q.Peek(); ok || gotKey != "" || gotPriority != 0 {
			t.Errorf("Peek() on empty = (%q, %d, %v), want (\"\", 0, false)", gotKey, gotPriority, ok)
		}
	})

	t.Run("zeroes vacated slot", func(t *testing.T) {
		q := New[*int, int]()
		a, b := new(int), new(int)
		q.Push(a, 1)
		q.Push(b, 2)
		q.Pop()
		q.Pop()

		full := q.data[:cap(q.data)]
		for i, got := range full {
			if got != (entry[*int, int]{}) {
				t.Errorf("data[%d] = %+v, want zero entry (Pop must zero vacated slots)", i, got)
			}
		}
	})
}

func TestPriorityQueue_PopN(t *testing.T) {
	q := New[int, int]()
	for k := range 5 {
		q.Push(k, 10-k)
	}

	if got, want := q.PopN(2), []Item[int, int]{{4, 6}, {3, 7}}; !slices.Equal(got, want) {
		t.Errorf("PopN(2) = %v, want %v", got, want)
	}

	checkInvariants(t, q)

	if got := q.PopN(10); len(got) != 3 {
		t.Errorf("PopN(10) = %v, want 3 items", got)
	}

	if got := q.IsEmpty(); !got {
		t.Errorf("IsEmpty() after PopN(10) = %v, want true", got)
	}

	if got := q.PopN(1); got != nil {
		t.Errorf("PopN(1) on empty = %v, want nil", got)
	}

	if got := q.PopN(0); got != nil {
		t.Errorf("PopN(0) = %v, want nil", got)
	}

	if got := q.PopN(-1); got != nil {
		t.Errorf("PopN(-1) = %v, want nil", got)
	}
}

func TestPriorityQueue_Remove(t *testing.T) {
	q := New[string, int]()
	q.Push("a", 1)
	q.Push("b", 2)
	q.Push("c", 3)

	if got, ok := q.Remove("b"); !ok || got != 2 {
		t.Errorf("Remove(b) = (%d, %v), want (2, true)", got, ok)
	}

	checkInvariants(t, q)

	if got := q.Contains("b"); got {
		t.Errorf("Contains(b) after Remove(b) = %v, want false", got)
	}

	if got, ok := q.Remove("b"); ok || got != 0 {
		t.Errorf("Remove(b) on removed = (%d, %v), want (0, false)", got, ok)
	}

	if _, ok := q.Remove("zzz"); ok {
		t.Errorf("Remove(zzz) on missing: ok = %v, want false", ok)
	}

	if got, ok := q.Remove("a"); !ok || got != 1 { // the root
		t.Errorf("Remove(a) = (%d, %v), want (1, true)", got, ok)
	}

	checkInvariants(t, q)

	if got, want := popAll(q), []Item[string, int]{{"c", 3}}; !slices.Equal(got, want) {
		t.Errorf("Pop() order after Remove() = %v, want %v", got, want)
	}
}

func TestPriorityQueue_PriorityContains(t *testing.T) {
	q := New[string, int]()
	q.Push("a", 4)

	if got, ok := q.Priority("a"); !ok || got != 4 {
		t.Errorf("Priority(a) = (%d, %v), want (4, true)", got, ok)
	}

	if got, ok := q.Priority("x"); ok || got != 0 {
		t.Errorf("Priority(x) = (%d, %v), want (0, false)", got, ok)
	}

	if got := q.Contains("a"); !got {
		t.Errorf("Contains(a) = %v, want true", got)
	}

	if got := q.Contains("x"); got {
		t.Errorf("Contains(x) = %v, want false", got)
	}

	q.Pop()

	if got := q.Contains("a"); got {
		t.Errorf("Contains(a) after Pop() = %v, want false", got)
	}

	if got := q.Push("a", 1); !got {
		t.Errorf("Push(a, 1) after Pop() = %v, want true (popped key is new again)", got)
	}
}

// TestPriorityQueue_RandomOps runs a long random mix of pushes, priority
// updates, pops, and removals against a reference map and checks the
// invariants after each step.
func TestPriorityQueue_RandomOps(t *testing.T) {
	r := rand.New(rand.NewPCG(11, 12))
	q := New[int, int]()
	ref := map[int]int{}

	for i := range 20000 {
		k := r.IntN(200)
		switch op := r.IntN(10); {
		case op < 5:
			_, existed := ref[k]
			if got := q.Push(k, r.IntN(100)); got == existed {
				t.Fatalf("Push(%d, _) #%d = %v, want %v", k, i, got, !existed)
			}

			ref[k], _ = q.Priority(k)
		case op < 8:
			gotKey, gotPriority, ok := q.Pop()
			if ok != (len(ref) > 0) {
				t.Fatalf("Pop() #%d: ok = %v, want %v (%d keys queued)", i, ok, len(ref) > 0, len(ref))
			}

			if ok {
				for _, rp := range ref {
					if rp < gotPriority {
						t.Fatalf("Pop() #%d priority = %d, want <= %d (smaller priority still queued)", i, gotPriority, rp)
					}
				}

				if want := ref[gotKey]; gotPriority != want {
					t.Fatalf("Pop() #%d = (%d, %d), want priority %d", i, gotKey, gotPriority, want)
				}

				delete(ref, gotKey)
			}
		default:
			got, ok := q.Remove(k)
			want, wantOK := ref[k]
			if ok != wantOK || (ok && got != want) {
				t.Fatalf("Remove(%d) #%d = (%d, %v), want (%d, %v)", k, i, got, ok, want, wantOK)
			}

			delete(ref, k)
		}

		checkInvariants(t, q)

		if got := q.Len(); got != len(ref) {
			t.Fatalf("Len() #%d = %d, want %d", i, got, len(ref))
		}
	}
}

func TestPriorityQueue_AllSliceSorted(t *testing.T) {
	q := New[string, int]()
	q.Push("c", 3)
	q.Push("a", 1)
	q.Push("b", 1)
	q.Push("d", 0)

	var got []Item[string, int]
	for k, p := range q.All() {
		got = append(got, Item[string, int]{k, p})
	}

	if want := q.Slice(); !slices.Equal(got, want) {
		t.Errorf("All() = %v, want Slice() = %v", got, want)
	}

	if want := (Item[string, int]{"d", 0}); got[0] != want {
		t.Errorf("All() first = %v, want %v (least item first)", got[0], want)
	}

	want := []Item[string, int]{{"d", 0}, {"a", 1}, {"b", 1}, {"c", 3}}
	if got := q.Sorted(); !slices.Equal(got, want) {
		t.Errorf("Sorted() = %v, want %v", got, want)
	}

	if got := q.Len(); got != 4 {
		t.Errorf("Len() after Sorted() = %d, want 4 (Sorted must not consume)", got)
	}

	checkInvariants(t, q)

	count := 0
	for range q.All() {
		count++
		break
	}

	if count != 1 {
		t.Errorf("All() with early break count = %d, want 1", count)
	}
}

func TestPriorityQueue_Slice(t *testing.T) {
	q := New[string, int]()
	q.Push("a", 1)
	s := q.Slice()
	s[0].Priority = 99

	if got, _ := q.Priority("a"); got != 1 {
		t.Errorf("Priority(a) after mutating Slice() = %d, want 1 (slice must be detached)", got)
	}
}

func TestPriorityQueue_Clone(t *testing.T) {
	t.Run("is independent of the original", func(t *testing.T) {
		q := New[int, int]()
		for k := range 20 {
			q.Push(k, 20-k)
		}

		c := q.Clone()

		checkInvariants(t, c)

		if got := &c.data[0]; got == &q.data[0] {
			t.Fatalf("&data[0] of clone = %p, want new backing array (not %p)", got, &q.data[0])
		}

		if got, want := c.Cap(), c.Len(); got != want {
			t.Errorf("Cap() of clone = %d, want Len() = %d", got, want)
		}

		// Mutating the clone must never touch the original's index.
		for k := range 10 {
			c.Push(k, -k)
		}

		c.Pop()
		c.Remove(15)

		checkInvariants(t, c)
		checkInvariants(t, q)

		if got := q.Len(); got != 20 {
			t.Errorf("original Len() = %d, want 20", got)
		}

		if got, _ := q.Priority(3); got != 17 {
			t.Errorf("original Priority(3) = %d, want 17", got)
		}
	})

	t.Run("keeps stable order", func(t *testing.T) {
		q := New[string, int]()
		q.Push("x", 1)
		q.Push("y", 1)
		c := q.Clone()
		c.Push("z", 1)

		var got []string
		for _, it := range popAll(c) {
			got = append(got, it.Key)
		}

		if want := []string{"x", "y", "z"}; !slices.Equal(got, want) {
			t.Errorf("Pop() keys of clone = %v, want %v", got, want)
		}
	})
}

func TestPriorityQueue_ResetClear(t *testing.T) {
	q := NewWithCap[string, int](8)
	q.Push("a", 1)
	q.Push("b", 2)

	q.Reset()

	if got := q.Len(); got != 0 {
		t.Errorf("Len() after Reset() = %d, want 0", got)
	}

	if got := q.Cap(); got != 8 {
		t.Errorf("Cap() after Reset() = %d, want 8", got)
	}

	if got := q.Contains("a"); got {
		t.Errorf("Contains(a) after Reset() = %v, want false", got)
	}

	for i, got := range q.data[:cap(q.data)] {
		if got != (entry[string, int]{}) {
			t.Errorf("data[%d] after Reset() = %+v, want zero entry", i, got)
		}
	}

	q.Push("c", 3)

	checkInvariants(t, q)

	q.Clear()

	if got := q.Len(); got != 0 {
		t.Errorf("Len() after Clear() = %d, want 0", got)
	}

	if got := q.Cap(); got != 0 {
		t.Errorf("Cap() after Clear() = %d, want 0", got)
	}

	if got := q.Contains("c"); got {
		t.Errorf("Contains(c) after Clear() = %v, want false", got)
	}

	q.Push("d", 4)

	checkInvariants(t, q)

	if got, _, _ := q.Pop(); got != "d" {
		t.Errorf("Pop() after Clear() and Push(d, 4) = %q, want %q", got, "d")
	}
}

func TestPriorityQueue_Capacity(t *testing.T) {
	q := New[int, int]()
	q.Grow(100)

	if got := q.Cap(); got < 100 {
		t.Errorf("Cap() after Grow(100) = %d, want >= 100", got)
	}

	q.Grow(0)
	q.Grow(-1)
	for k := range 5 {
		q.Push(k, k)
	}

	q.Clip()

	if got, want := q.Cap(), q.Len(); got != want {
		t.Errorf("Cap() after Clip() = %d, want Len() = %d", got, want)
	}

	checkInvariants(t, q)

	q.Grow(50)
	q.Shrink()

	if got, want := q.Cap(), q.Len(); got != want {
		t.Errorf("Cap() after Shrink() = %d, want Len() = %d", got, want)
	}

	checkInvariants(t, q)

	q.Shrink() // already tight: still valid

	checkInvariants(t, q)

	q.Push(99, -1)

	if got, _, _ := q.Pop(); got != 99 {
		t.Errorf("Pop() after Shrink() and Push(99, -1) = %d, want 99", got)
	}
}

func TestPriorityQueue_NilReceiver(t *testing.T) {
	var q *PriorityQueue[string, int]

	if got := q.Len(); got != 0 {
		t.Errorf("nil.Len() = %d, want 0", got)
	}

	if got := q.Cap(); got != 0 {
		t.Errorf("nil.Cap() = %d, want 0", got)
	}

	if got := q.IsEmpty(); !got {
		t.Errorf("nil.IsEmpty() = %v, want true", got)
	}

	if _, _, ok := q.Peek(); ok {
		t.Errorf("nil.Peek(): ok = %v, want false", ok)
	}

	if _, _, ok := q.Pop(); ok {
		t.Errorf("nil.Pop(): ok = %v, want false", ok)
	}

	if got := q.PopN(3); got != nil {
		t.Errorf("nil.PopN(3) = %v, want nil", got)
	}

	if got := q.Slice(); got != nil {
		t.Errorf("nil.Slice() = %v, want nil", got)
	}

	if got := q.Sorted(); got != nil {
		t.Errorf("nil.Sorted() = %v, want nil", got)
	}

	if got := q.Clone(); got != nil {
		t.Errorf("nil.Clone() = %p, want nil", got)
	}

	if _, ok := q.Remove("a"); ok {
		t.Errorf("nil.Remove(a): ok = %v, want false", ok)
	}

	if _, ok := q.Priority("a"); ok {
		t.Errorf("nil.Priority(a): ok = %v, want false", ok)
	}

	if got := q.Contains("a"); got {
		t.Errorf("nil.Contains(a) = %v, want false", got)
	}

	count := 0
	for range q.All() {
		count++
	}

	if count != 0 {
		t.Errorf("nil.All() count = %d, want 0", count)
	}

	t.Run("mutators panic", func(t *testing.T) {
		tests := []struct {
			name string
			call func(q *PriorityQueue[string, int])
		}{
			{
				name: "Push",
				call: func(q *PriorityQueue[string, int]) { q.Push("a", 1) },
			},
			{
				name: "Grow",
				call: func(q *PriorityQueue[string, int]) { q.Grow(1) },
			},
			{
				name: "Clear",
				call: func(q *PriorityQueue[string, int]) { q.Clear() },
			},
			{
				name: "Reset",
				call: func(q *PriorityQueue[string, int]) { q.Reset() },
			},
			{
				name: "Shrink",
				call: func(q *PriorityQueue[string, int]) { q.Shrink() },
			},
			{
				name: "Clip",
				call: func(q *PriorityQueue[string, int]) { q.Clip() },
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				sharedtest.Panics(t, tt.name, tt.name+" on nil receiver", func() { tt.call(q) })
			})
		}
	})
}

func TestPriorityQueue_ZeroValue(t *testing.T) {
	var q PriorityQueue[string, int]

	if got := q.IsEmpty(); !got {
		t.Errorf("IsEmpty() on zero value = %v, want true", got)
	}

	if got := q.Contains("a"); got {
		t.Errorf("Contains(a) on zero value = %v, want false", got)
	}

	if _, _, ok := q.Pop(); ok {
		t.Errorf("Pop() on zero value: ok = %v, want false", ok)
	}

	if _, ok := q.Remove("a"); ok {
		t.Errorf("Remove(a) on zero value: ok = %v, want false", ok)
	}

	sharedtest.Panics(t, "Push on zero value", "zero-value PriorityQueue", func() { q.Push("a", 1) })

	q.Reset()
	q.Clip()
	q.Shrink()
	q.Grow(4)

	c := q.Clone()
	if c == nil {
		t.Fatalf("Clone() on zero value = nil, want empty queue")
	}

	if got := c.IsEmpty(); !got {
		t.Errorf("Clone().IsEmpty() on zero value = %v, want true", got)
	}
}
