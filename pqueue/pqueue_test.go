package pqueue

import (
	"cmp"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Nergous/structures/internal/shared/sharedtest"
)

// assertNewEmpty fails the test unless q is an empty priority queue with no
// preallocated capacity.
func assertNewEmpty[K comparable, P any](t *testing.T, q *Queue[K, P]) {
	t.Helper()

	if got := q.Len(); got != 0 {
		t.Errorf("Len() = %d, want 0", got)
	}

	if got := q.Cap(); got != 0 {
		t.Errorf("Cap() = %d, want 0", got)
	}
}

// assertInvariants fails the test unless q.data is in heap order, every key's
// recorded position matches the backing array, and the index holds no stale
// keys.
func assertInvariants[K comparable, P any](t *testing.T, q *Queue[K, P]) {
	t.Helper()

	sharedtest.Heap(t, q.data, q.compare)

	if got, want := len(q.index), len(q.data); got != want {
		t.Fatalf("len(index) = %d, want len(data) = %d", got, want)
	}

	for i, e := range q.data {
		if got, ok := q.index[e.key]; !ok || got != i {
			t.Fatalf("index[%v] = (%d, %v), want (%d, true)",
				e.key, got, ok, i)
		}
	}
}

// assertTailZeroed fails the test unless every slot of the backing array
// between Len and Cap holds the zero entry.
func assertTailZeroed[K, P comparable](t *testing.T, q *Queue[K, P]) {
	t.Helper()

	var zero entry[K, P]
	for i, e := range q.data[len(q.data):cap(q.data)] {
		if e != zero {
			t.Errorf("data[%d] = %v, want zero value", len(q.data)+i, e)
		}
	}
}

// fill returns a min priority queue holding the keys 0..n-1, each with its own
// key as priority.
func fill(n int) *Queue[int, int] {
	q := New[int, int]()
	for i := range n {
		q.Push(i, i)
	}

	return q
}

// drain pops every key from q and returns the items in pop order.
func drain[K comparable, P any](q *Queue[K, P]) []Item[K, P] {
	var out []Item[K, P]
	for {
		k, p, ok := q.Pop()
		if !ok {
			return out
		}

		out = append(out, Item[K, P]{Key: k, Priority: p})
	}
}

// keysOf returns the keys of items in order.
func keysOf[K comparable, P any](items []Item[K, P]) []K {
	out := make([]K, len(items))
	for i, it := range items {
		out[i] = it.Key
	}

	return out
}

func TestQueue_New(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		assertNewEmpty(t, New[string, int]())
	})

	t.Run("string", func(t *testing.T) {
		assertNewEmpty(t, New[string, string]())
	})

	t.Run("float32", func(t *testing.T) {
		assertNewEmpty(t, New[string, float32]())
	})
}

func TestQueue_NewWithCap(t *testing.T) {
	constructors := []struct {
		name string
		newQ func(n int) *Queue[int, int]
	}{
		{
			name: "NewWithCap",
			newQ: NewWithCap[int, int],
		},
		{
			name: "NewMaxWithCap",
			newQ: NewMaxWithCap[int, int],
		},
		{
			name: "NewFuncWithCap",
			newQ: func(n int) *Queue[int, int] {
				return NewFuncWithCap[int](n, cmp.Compare[int])
			},
		},
	}

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

	for _, ctor := range constructors {
		t.Run(ctor.name, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					q := ctor.newQ(tt.n)

					if got := q.Len(); got != 0 {
						t.Errorf("Len() = %d, want 0", got)
					}

					if got := q.Cap(); got != tt.want {
						t.Errorf("Cap() = %d, want %d", got, tt.want)
					}

					q.Push(1, 1)

					if got, p, ok := q.Pop(); !ok || got != 1 || p != 1 {
						t.Errorf("Pop() after Push(1, 1) = (%d, %d, %v), "+
							"want (1, 1, true)", got, p, ok)
					}
				})
			}
		})
	}

	t.Run("nil comparator panics", func(t *testing.T) {
		sharedtest.Panics(t, "NewFuncWithCap(4, nil)",
			"pqueue: nil comparator",
			func() { NewFuncWithCap[string, int](4, nil) })
	})
}

func TestQueue_From(t *testing.T) {
	tests := []struct {
		name    string
		items   []Item[int, int]
		want    []Item[int, int]
		wantLen int
	}{
		{
			name:    "nil",
			items:   nil,
			want:    nil,
			wantLen: 0,
		},
		{
			name:    "empty",
			items:   []Item[int, int]{},
			want:    nil,
			wantLen: 0,
		},
		{
			name:    "one",
			items:   []Item[int, int]{{1, 1}},
			want:    []Item[int, int]{{1, 1}},
			wantLen: 1,
		},
		{
			name:    "sorted",
			items:   []Item[int, int]{{1, 1}, {2, 2}, {3, 3}, {4, 4}},
			want:    []Item[int, int]{{1, 1}, {2, 2}, {3, 3}, {4, 4}},
			wantLen: 4,
		},
		{
			name:    "reversed",
			items:   []Item[int, int]{{4, 4}, {3, 3}, {2, 2}, {1, 1}},
			want:    []Item[int, int]{{1, 1}, {2, 2}, {3, 3}, {4, 4}},
			wantLen: 4,
		},
		{
			name:    "shuffled",
			items:   []Item[int, int]{{3, 3}, {1, 1}, {4, 4}, {2, 2}},
			want:    []Item[int, int]{{1, 1}, {2, 2}, {3, 3}, {4, 4}},
			wantLen: 4,
		},
		{
			name: "duplicate keys keep last priority",
			items: []Item[int, int]{
				{1, 5}, {2, 3}, {1, 1}, {3, 4}, {2, 9},
			},
			want:    []Item[int, int]{{1, 1}, {3, 4}, {2, 9}},
			wantLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := From(tt.items)

			assertInvariants(t, q)

			if got := q.Len(); got != tt.wantLen {
				t.Errorf("Len() = %d, want %d", got, tt.wantLen)
			}

			if got := q.Cap(); got != len(tt.items) {
				t.Errorf("Cap() = %d, want %d", got, len(tt.items))
			}

			if got := drain(q); !slices.Equal(got, tt.want) {
				t.Errorf("Pop() order = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("never aliases the input", func(t *testing.T) {
		items := []Item[int, int]{{1, 5}, {2, 3}, {3, 8}}
		q := From(items)

		items[1].Priority = 100

		if got, p, ok := q.Peek(); !ok || got != 2 || p != 3 {
			t.Errorf("Peek() after changing input = (%d, %d, %v), "+
				"want (2, 3, true)", got, p, ok)
		}

		q.Push(2, 0)

		want := []Item[int, int]{{1, 5}, {2, 100}, {3, 8}}
		if !slices.Equal(items, want) {
			t.Errorf("input after Push(2, 0) = %v, want %v", items, want)
		}
	})

	t.Run("matches pushing one by one", func(t *testing.T) {
		items := []Item[string, int]{
			{"a", 1}, {"b", 1}, {"c", 0}, {"a", 1},
			{"d", 1}, {"b", 0}, {"e", 1},
		}
		q := From(items)
		pushed := New[string, int]()
		pushed.PushN(items...)

		assertInvariants(t, q)

		if got, want := drain(q), drain(pushed); !slices.Equal(got, want) {
			t.Errorf("Pop() order of From() = %v, want %v (PushN order)",
				got, want)
		}
	})
}

func TestQueue_NewMax(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		assertNewEmpty(t, NewMax[string, int]())
	})

	t.Run("string", func(t *testing.T) {
		assertNewEmpty(t, NewMax[string, string]())
	})

	t.Run("float32", func(t *testing.T) {
		assertNewEmpty(t, NewMax[string, float32]())
	})
}

func TestQueue_FromMax(t *testing.T) {
	items := []Item[string, int]{
		{"low", 1}, {"high", 9}, {"mid", 5}, {"tied", 9},
	}
	q := FromMax(items)

	assertInvariants(t, q)

	want := []Item[string, int]{
		{"high", 9}, {"tied", 9}, {"mid", 5}, {"low", 1},
	}
	if got := drain(q); !slices.Equal(got, want) {
		t.Errorf("Pop() order = %v, want %v", got, want)
	}
}

func TestQueue_NewFunc(t *testing.T) {
	t.Run("returns an empty queue", func(t *testing.T) {
		assertNewEmpty(t, NewFunc[string](cmp.Compare[int]))
	})

	t.Run("orders by comparator", func(t *testing.T) {
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

		want := []string{"earliest", "early", "late"}
		if got := keysOf(drain(q)); !slices.Equal(got, want) {
			t.Errorf("Pop() keys = %v, want %v", got, want)
		}
	})

	t.Run("nil comparator panics", func(t *testing.T) {
		sharedtest.Panics(t, "NewFunc(nil)", "pqueue: nil comparator",
			func() { NewFunc[string, int](nil) })
	})
}

func TestQueue_FromFunc(t *testing.T) {
	t.Run("orders by comparator", func(t *testing.T) {
		byLen := func(a, b string) int {
			return cmp.Compare(len(a), len(b))
		}
		items := []Item[string, string]{
			{"x", "ccc"}, {"y", "a"}, {"z", "dddd"}, {"w", "bb"},
		}
		q := FromFunc(items, byLen)

		assertInvariants(t, q)

		want := []string{"y", "w", "x", "z"}
		if got := keysOf(drain(q)); !slices.Equal(got, want) {
			t.Errorf("Pop() keys = %v, want %v", got, want)
		}
	})

	t.Run("nil comparator panics", func(t *testing.T) {
		items := []Item[string, int]{{"a", 1}}
		sharedtest.Panics(t, "FromFunc(items, nil)", "pqueue: nil comparator",
			func() { FromFunc[string, int](items, nil) })
	})
}

func TestQueue_Len(t *testing.T) {
	q := New[string, int]()
	q.Push("a", 1)
	q.Push("b", 2)
	q.Push("c", 3)

	if got := q.Len(); got != 3 {
		t.Errorf("Len() after three Push() = %d, want 3", got)
	}

	q.Pop()

	if got := q.Len(); got != 2 {
		t.Errorf("Len() after Pop() = %d, want 2", got)
	}
}

func TestQueue_Cap(t *testing.T) {
	t.Run("new queue has zero cap", func(t *testing.T) {
		q := New[int, int]()

		if got := q.Cap(); got != 0 {
			t.Errorf("Cap() = %d, want 0", got)
		}
	})

	t.Run("never less than Len while growing", func(t *testing.T) {
		q := New[int, int]()
		for i := range 100 {
			q.Push(i, i)
			if c, l := q.Cap(), q.Len(); c < l {
				t.Fatalf("Cap() #%d = %d, want >= Len() = %d", i, c, l)
			}
		}
	})

	t.Run("no reallocation within preallocated capacity", func(t *testing.T) {
		q := NewWithCap[int, int](4)
		for i := range 4 {
			q.Push(i, i)
		}

		if got := q.Cap(); got != 4 {
			t.Errorf("Cap() after 4 Push() = %d, want 4", got)
		}
	})
}

func TestQueue_IsEmpty(t *testing.T) {
	q := New[string, int]()

	if got := q.IsEmpty(); !got {
		t.Errorf("IsEmpty() on empty = %v, want true", got)
	}

	q.Push("a", 1)

	if got := q.IsEmpty(); got {
		t.Errorf("IsEmpty() after Push(a, 1) = %v, want false", got)
	}

	q.Pop()

	if got := q.IsEmpty(); !got {
		t.Errorf("IsEmpty() after Pop() = %v, want true", got)
	}
}

func TestQueue_Peek(t *testing.T) {
	t.Run("empty returns zero values and false", func(t *testing.T) {
		q := New[string, int]()

		if got, p, ok := q.Peek(); ok || got != "" || p != 0 {
			t.Errorf("Peek() on empty = (%q, %d, %v), "+
				"want (\"\", 0, false)", got, p, ok)
		}
	})

	t.Run("returns least without removing it", func(t *testing.T) {
		q := New[string, int]()
		q.Push("b", 2)
		q.Push("a", 1)

		if got, p, ok := q.Peek(); !ok || got != "a" || p != 1 {
			t.Errorf("Peek() = (%q, %d, %v), want (\"a\", 1, true)",
				got, p, ok)
		}

		if got := q.Len(); got != 2 {
			t.Errorf("Len() after Peek() = %d, want 2", got)
		}
	})

	t.Run("follows priority updates", func(t *testing.T) {
		q := New[string, int]()
		q.Push("a", 1)
		q.Push("b", 2)

		q.Push("b", 0)

		if got, p, ok := q.Peek(); !ok || got != "b" || p != 0 {
			t.Errorf("Peek() after Push(b, 0) = (%q, %d, %v), "+
				"want (\"b\", 0, true)", got, p, ok)
		}
	})
}

func TestQueue_Contains(t *testing.T) {
	q := New[string, int]()
	q.Push("a", 1)
	q.Push("b", 2)

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

	q.Remove("b")

	if got := q.Contains("b"); got {
		t.Errorf("Contains(b) after Remove(b) = %v, want false", got)
	}
}

func TestQueue_Priority(t *testing.T) {
	q := New[string, int]()
	q.Push("a", 4)

	if got, ok := q.Priority("a"); !ok || got != 4 {
		t.Errorf("Priority(a) = (%d, %v), want (4, true)", got, ok)
	}

	if got, ok := q.Priority("x"); ok || got != 0 {
		t.Errorf("Priority(x) = (%d, %v), want (0, false)", got, ok)
	}

	q.Push("a", 7)

	if got, ok := q.Priority("a"); !ok || got != 7 {
		t.Errorf("Priority(a) after Push(a, 7) = (%d, %v), want (7, true)",
			got, ok)
	}
}

func TestQueue_All(t *testing.T) {
	newQ := func() *Queue[string, int] {
		q := New[string, int]()
		q.Push("c", 3)
		q.Push("a", 1)
		q.Push("b", 1)
		q.Push("d", 0)

		return q
	}

	t.Run("yields layout order with least first", func(t *testing.T) {
		q := newQ()

		var got []Item[string, int]
		for k, p := range q.All() {
			got = append(got, Item[string, int]{k, p})
		}

		if want := items(q.data); !slices.Equal(got, want) {
			t.Errorf("All() = %v, want layout order %v", got, want)
		}

		if want := (Item[string, int]{"d", 0}); got[0] != want {
			t.Errorf("All() first = %v, want least item %v", got[0], want)
		}
	})

	t.Run("empty queue yields nothing", func(t *testing.T) {
		q := New[string, int]()

		count := 0
		for range q.All() {
			count++
		}

		if count != 0 {
			t.Errorf("All() on empty count = %d, want 0", count)
		}
	})

	t.Run("does not consume", func(t *testing.T) {
		q := newQ()

		for range q.All() {
		}

		if got := q.Len(); got != 4 {
			t.Errorf("Len() after All() = %d, want 4", got)
		}

		assertInvariants(t, q)
	})

	t.Run("break stops iteration early", func(t *testing.T) {
		q := newQ()

		var got []string
		for k := range q.All() {
			got = append(got, k)
			if len(got) == 2 {
				break
			}
		}

		if want := keysOf(q.Slice())[:2]; !slices.Equal(got, want) {
			t.Errorf("All() with early break = %v, want %v", got, want)
		}

		if got := q.Len(); got != 4 {
			t.Errorf("Len() after break = %d, want 4", got)
		}
	})
}

func TestQueue_Clone(t *testing.T) {
	t.Run("copies all elements in order", func(t *testing.T) {
		q := From([]Item[int, int]{{5, 5}, {3, 3}, {8, 8}, {1, 1}, {9, 9}})

		c := q.Clone()

		if got, want := c.Slice(), q.Slice(); !slices.Equal(got, want) {
			t.Errorf("Slice() of clone = %v, want %v", got, want)
		}
	})

	t.Run("capacity equals length", func(t *testing.T) {
		q := NewWithCap[int, int](64)
		for i := range 5 {
			q.Push(i, i)
		}

		c := q.Clone()

		if got, want := c.Cap(), c.Len(); got != want {
			t.Errorf("Cap() of clone = %d, want Len() = %d", got, want)
		}
	})

	t.Run("is independent of the original", func(t *testing.T) {
		q := New[int, int]()
		for k := range 20 {
			q.Push(k, 20-k)
		}

		c := q.Clone()

		assertInvariants(t, c)

		if got := &c.data[0]; got == &q.data[0] {
			t.Fatalf("&data[0] of clone = %p, want a new backing array", got)
		}

		// Mutating the clone must never touch the original's index.
		for k := range 10 {
			c.Push(k, -k)
		}

		c.Pop()
		c.Remove(15)

		assertInvariants(t, c)
		assertInvariants(t, q)

		if got := q.Len(); got != 20 {
			t.Errorf("Len() of original = %d, want 20", got)
		}

		if got, ok := q.Priority(3); !ok || got != 17 {
			t.Errorf("Priority(3) of original = (%d, %v), want (17, true)",
				got, ok)
		}
	})

	t.Run("clone of empty queue is empty and usable", func(t *testing.T) {
		q := New[string, int]()

		c := q.Clone()

		if got := c.Len(); got != 0 {
			t.Errorf("Len() of clone = %d, want 0", got)
		}

		c.Push("a", 1)

		if got, p, ok := c.Pop(); !ok || got != "a" || p != 1 {
			t.Errorf("Pop() of clone after Push(a, 1) = (%q, %d, %v), "+
				"want (\"a\", 1, true)", got, p, ok)
		}
	})

	t.Run("keeps stable order", func(t *testing.T) {
		q := New[string, int]()
		q.Push("x", 1)
		q.Push("y", 1)

		c := q.Clone()
		c.Push("z", 1)

		want := []string{"x", "y", "z"}
		if got := keysOf(drain(c)); !slices.Equal(got, want) {
			t.Errorf("Pop() keys of clone = %v, want %v", got, want)
		}
	})
}

func TestQueue_Slice(t *testing.T) {
	t.Run("returns layout order", func(t *testing.T) {
		q := New[string, int]()
		q.Push("c", 3)
		q.Push("a", 1)
		q.Push("b", 2)

		got := q.Slice()

		if want := items(q.data); !slices.Equal(got, want) {
			t.Errorf("Slice() = %v, want layout order %v", got, want)
		}

		if want := (Item[string, int]{"a", 1}); got[0] != want {
			t.Errorf("Slice()[0] = %v, want least item %v", got[0], want)
		}
	})

	t.Run("returns an independent copy", func(t *testing.T) {
		q := New[string, int]()
		q.Push("a", 1)
		q.Push("b", 2)
		want := q.Slice()

		out := q.Slice()
		for i := range out {
			out[i].Priority = 999
		}

		if got := q.Slice(); !slices.Equal(got, want) {
			t.Errorf("Slice() after mutating a copy = %v, want %v", got, want)
		}
	})

	t.Run("empty queue returns nil", func(t *testing.T) {
		q := New[string, int]()

		if got := q.Slice(); got != nil {
			t.Errorf("Slice() on empty = %v, want nil", got)
		}
	})
}

func TestQueue_Sorted(t *testing.T) {
	t.Run("priority order with ties in push order", func(t *testing.T) {
		q := New[string, int]()
		q.Push("c", 3)
		q.Push("a", 1)
		q.Push("b", 1)
		q.Push("d", 0)

		want := []Item[string, int]{{"d", 0}, {"a", 1}, {"b", 1}, {"c", 3}}
		if got := q.Sorted(); !slices.Equal(got, want) {
			t.Errorf("Sorted() = %v, want %v", got, want)
		}

		if got := q.Len(); got != 4 {
			t.Errorf("Len() after Sorted() = %d, want 4", got)
		}

		assertInvariants(t, q)
	})

	t.Run("max queue descending", func(t *testing.T) {
		q := NewMax[string, int]()
		q.Push("a", 1)
		q.Push("c", 8)
		q.Push("b", 3)

		want := []Item[string, int]{{"c", 8}, {"b", 3}, {"a", 1}}
		if got := q.Sorted(); !slices.Equal(got, want) {
			t.Errorf("Sorted() on max queue = %v, want %v", got, want)
		}
	})

	t.Run("empty queue returns nil", func(t *testing.T) {
		q := New[string, int]()

		if got := q.Sorted(); got != nil {
			t.Errorf("Sorted() on empty = %v, want nil", got)
		}
	})
}

func TestQueue_Push(t *testing.T) {
	t.Run("reports new key", func(t *testing.T) {
		q := New[string, int]()

		if got := q.Push("a", 5); !got {
			t.Errorf("Push(a, 5) = %v, want true", got)
		}

		if got := q.Push("a", 1); got {
			t.Errorf("Push(a, 1) on existing = %v, want false", got)
		}

		if got := q.Len(); got != 1 {
			t.Errorf("Len() after Push(a, 1) = %d, want 1", got)
		}

		if got, ok := q.Priority("a"); !ok || got != 1 {
			t.Errorf("Priority(a) after Push(a, 1) = (%d, %v), "+
				"want (1, true)", got, ok)
		}
	})

	t.Run("update moves both ways", func(t *testing.T) {
		q := New[string, int]()
		for i, k := range []string{"a", "b", "c", "d", "e"} {
			q.Push(k, (i+1)*10)
		}

		q.Push("e", 1) // decrease: e goes to the top

		assertInvariants(t, q)

		if got, _, ok := q.Peek(); !ok || got != "e" {
			t.Errorf("Peek() after Push(e, 1) = (%q, %v), want (\"e\", true)",
				got, ok)
		}

		q.Push("e", 100) // increase: e sinks to the bottom

		assertInvariants(t, q)

		all := drain(q)

		want := Item[string, int]{"e", 100}
		if got := all[len(all)-1]; got != want {
			t.Errorf("last Pop() after Push(e, 100) = %v, want %v", got, want)
		}
	})

	t.Run("equal priorities come out in push order", func(t *testing.T) {
		tests := []struct {
			name string
			newQ func() *Queue[int, int]
		}{
			{
				name: "New",
				newQ: New[int, int],
			},
			{
				name: "NewMax",
				newQ: NewMax[int, int],
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				q := tt.newQ()
				for k := range 50 {
					q.Push(k, k%3)
				}

				prev := map[int]int{} // priority -> last key seen
				for _, it := range drain(q) {
					last, ok := prev[it.Priority]
					if ok && it.Key < last {
						t.Fatalf("Pop() key with priority %d = %d, "+
							"want > %d (push order)",
							it.Priority, it.Key, last)
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

		want := []string{"first", "second", "third"}
		if got := keysOf(drain(q)); !slices.Equal(got, want) {
			t.Errorf("Pop() keys = %v, want %v", got, want)
		}
	})

	t.Run("popped key is new again", func(t *testing.T) {
		q := New[string, int]()
		q.Push("a", 1)
		q.Pop()

		if got := q.Push("a", 1); !got {
			t.Errorf("Push(a, 1) after Pop() = %v, want true", got)
		}
	})
}

func TestQueue_PushN(t *testing.T) {
	t.Run("no arguments is a no-op", func(t *testing.T) {
		q := NewWithCap[string, int](4)

		q.PushN()

		if got := q.Len(); got != 0 {
			t.Errorf("Len() after PushN() = %d, want 0", got)
		}

		if got := q.Cap(); got != 4 {
			t.Errorf("Cap() after PushN() = %d, want 4", got)
		}
	})

	t.Run("adds new keys", func(t *testing.T) {
		q := New[string, int]()
		q.Push("m", 5)

		q.PushN(
			Item[string, int]{"z", 9},
			Item[string, int]{"a", 1},
			Item[string, int]{"k", 3},
		)

		assertInvariants(t, q)

		want := []Item[string, int]{{"a", 1}, {"k", 3}, {"m", 5}, {"z", 9}}
		if got := drain(q); !slices.Equal(got, want) {
			t.Errorf("Pop() order after PushN() = %v, want %v", got, want)
		}
	})

	t.Run("updates existing keys in place", func(t *testing.T) {
		q := New[string, int]()
		q.Push("a", 5)
		q.Push("b", 5)

		q.PushN(
			Item[string, int]{"a", 7},
			Item[string, int]{"a", 5},
			Item[string, int]{"c", 5},
		)

		assertInvariants(t, q)

		want := []string{"a", "b", "c"}
		if got := keysOf(drain(q)); !slices.Equal(got, want) {
			t.Errorf("Pop() keys after PushN() = %v, "+
				"want %v (a keeps its place)", got, want)
		}
	})

	t.Run("repeated key last wins", func(t *testing.T) {
		q := New[string, int]()

		q.PushN(
			Item[string, int]{"a", 1},
			Item[string, int]{"b", 2},
			Item[string, int]{"a", 3},
		)

		if got := q.Len(); got != 2 {
			t.Errorf("Len() after PushN() = %d, want 2", got)
		}

		if got, ok := q.Priority("a"); !ok || got != 3 {
			t.Errorf("Priority(a) after PushN() = (%d, %v), want (3, true)",
				got, ok)
		}
	})
}

func TestQueue_Pop(t *testing.T) {
	tests := []struct {
		name string
		newQ func() *Queue[int, int]
		push []Item[int, int]
		want []Item[int, int]
	}{
		{
			name: "empty",
			newQ: New[int, int],
			push: nil,
			want: nil,
		},
		{
			name: "single element",
			newQ: New[int, int],
			push: []Item[int, int]{{42, 42}},
			want: []Item[int, int]{{42, 42}},
		},
		{
			name: "min order",
			newQ: New[int, int],
			push: []Item[int, int]{{5, 5}, {1, 1}, {4, 4}, {2, 2}, {3, 3}},
			want: []Item[int, int]{{1, 1}, {2, 2}, {3, 3}, {4, 4}, {5, 5}},
		},
		{
			name: "min order with duplicates",
			newQ: New[int, int],
			push: []Item[int, int]{{1, 2}, {2, 1}, {3, 2}, {4, 1}},
			want: []Item[int, int]{{2, 1}, {4, 1}, {1, 2}, {3, 2}},
		},
		{
			name: "max order",
			newQ: NewMax[int, int],
			push: []Item[int, int]{{5, 5}, {1, 1}, {4, 4}, {2, 2}, {3, 3}},
			want: []Item[int, int]{{5, 5}, {4, 4}, {3, 3}, {2, 2}, {1, 1}},
		},
		{
			name: "comparator order",
			newQ: func() *Queue[int, int] {
				return NewFunc[int](func(a, b int) int {
					return cmp.Compare(a%10, b%10)
				})
			},
			push: []Item[int, int]{{19, 19}, {21, 21}, {35, 35}, {3, 3}},
			want: []Item[int, int]{{21, 21}, {3, 3}, {35, 35}, {19, 19}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := tt.newQ()
			q.PushN(tt.push...)

			for i, want := range tt.want {
				got, p, ok := q.Pop()
				if !ok || got != want.Key || p != want.Priority {
					t.Errorf("Pop() #%d = (%d, %d, %v), want (%d, %d, true)",
						i, got, p, ok, want.Key, want.Priority)
				}

				assertInvariants(t, q)
			}

			if got, p, ok := q.Pop(); ok || got != 0 || p != 0 {
				t.Errorf("Pop() on empty = (%d, %d, %v), "+
					"want (0, 0, false)", got, p, ok)
			}
		})
	}

	t.Run("zeroes vacated slot", func(t *testing.T) {
		q := New[*int, int]()
		q.Push(new(int), 1)
		q.Push(new(int), 2)
		data := q.data[:2]

		if _, _, ok := q.Pop(); !ok {
			t.Fatalf("Pop() ok = false, want true")
		}

		if got := data[1]; got != (entry[*int, int]{}) {
			t.Errorf("data[1] after Pop() = %v, "+
				"want zero value (Pop must zero the slot)", got)
		}
	})
}

func TestQueue_PopN(t *testing.T) {
	five := []Item[int, int]{{0, 10}, {1, 9}, {2, 8}, {3, 7}, {4, 6}}
	sorted := []Item[int, int]{{4, 6}, {3, 7}, {2, 8}, {1, 9}, {0, 10}}

	tests := []struct {
		name string
		push []Item[int, int]
		n    int
		want []Item[int, int]
		left []Item[int, int]
	}{
		{
			name: "one",
			push: five,
			n:    1,
			want: sorted[:1],
			left: sorted[1:],
		},
		{
			name: "several in min order",
			push: five,
			n:    3,
			want: sorted[:3],
			left: sorted[3:],
		},
		{
			name: "exactly Len",
			push: five,
			n:    5,
			want: sorted,
			left: nil,
		},
		{
			name: "more than Len drains",
			push: five,
			n:    99,
			want: sorted,
			left: nil,
		},
		{
			name: "zero",
			push: five,
			n:    0,
			want: nil,
			left: sorted,
		},
		{
			name: "negative",
			push: five,
			n:    -3,
			want: nil,
			left: sorted,
		},
		{
			name: "empty",
			push: nil,
			n:    3,
			want: nil,
			left: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := New[int, int]()
			q.PushN(tt.push...)
			capBefore := q.Cap()

			got := q.PopN(tt.n)

			if !slices.Equal(got, tt.want) || (got == nil) != (tt.want == nil) {
				t.Errorf("PopN(%d) = %v, want %v", tt.n, got, tt.want)
			}

			if got := q.Sorted(); !slices.Equal(got, tt.left) {
				t.Errorf("Sorted() after PopN(%d) = %v, want %v",
					tt.n, got, tt.left)
			}

			if got := q.Cap(); got != capBefore {
				t.Errorf("Cap() after PopN(%d) = %d, want %d",
					tt.n, got, capBefore)
			}

			assertInvariants(t, q)
		})
	}

	t.Run("zeroes vacated slots", func(t *testing.T) {
		q := New[*int, int]()
		for i := range 4 {
			q.Push(new(int), i)
		}

		q.PopN(3)

		assertTailZeroed(t, q)
	})
}

func TestQueue_Remove(t *testing.T) {
	t.Run("removes inner key", func(t *testing.T) {
		q := From([]Item[string, int]{{"a", 1}, {"b", 2}, {"c", 3}})

		if got, ok := q.Remove("b"); !ok || got != 2 {
			t.Errorf("Remove(b) = (%d, %v), want (2, true)", got, ok)
		}

		assertInvariants(t, q)

		if got, ok := q.Remove("b"); ok || got != 0 {
			t.Errorf("Remove(b) on removed = (%d, %v), want (0, false)",
				got, ok)
		}
	})

	t.Run("missing key returns zero and false", func(t *testing.T) {
		q := From([]Item[string, int]{{"a", 1}})

		if got, ok := q.Remove("zzz"); ok || got != 0 {
			t.Errorf("Remove(zzz) = (%d, %v), want (0, false)", got, ok)
		}

		if got := q.Len(); got != 1 {
			t.Errorf("Len() after Remove(zzz) = %d, want 1", got)
		}
	})

	t.Run("removes root", func(t *testing.T) {
		q := From([]Item[string, int]{{"a", 1}, {"b", 2}, {"c", 3}})

		if got, ok := q.Remove("a"); !ok || got != 1 {
			t.Errorf("Remove(a) = (%d, %v), want (1, true)", got, ok)
		}

		assertInvariants(t, q)

		want := []Item[string, int]{{"b", 2}, {"c", 3}}
		if got := drain(q); !slices.Equal(got, want) {
			t.Errorf("Pop() order after Remove(a) = %v, want %v", got, want)
		}
	})

	t.Run("zeroes vacated slot", func(t *testing.T) {
		q := New[*int, int]()
		a := new(int)
		q.Push(new(int), 1)
		q.Push(a, 2)
		q.Push(new(int), 3)

		q.Remove(a)

		assertTailZeroed(t, q)
	})
}

func TestQueue_Clear(t *testing.T) {
	q := NewWithCap[string, int](8)
	q.Push("a", 1)
	q.Push("b", 2)

	q.Clear()

	if got := q.Len(); got != 0 {
		t.Errorf("Len() after Clear() = %d, want 0", got)
	}

	if got := q.Cap(); got != 0 {
		t.Errorf("Cap() after Clear() = %d, want 0", got)
	}

	if got, p, ok := q.Peek(); ok || got != "" || p != 0 {
		t.Errorf("Peek() after Clear() = (%q, %d, %v), "+
			"want (\"\", 0, false)", got, p, ok)
	}

	if q.index != nil {
		t.Errorf("index after Clear() = %v, want nil", q.index)
	}

	q.Push("d", 4)

	if got, p, ok := q.Pop(); !ok || got != "d" || p != 4 {
		t.Errorf("Pop() after Clear() and Push(d, 4) = (%q, %d, %v), "+
			"want (\"d\", 4, true)", got, p, ok)
	}
}

func TestQueue_Reset(t *testing.T) {
	t.Run("Reset keeps capacity", func(t *testing.T) {
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
	})

	t.Run("zeroes backing array", func(t *testing.T) {
		q := NewWithCap[*int, int](4)
		q.Push(new(int), 1)
		q.Push(new(int), 2)

		q.Reset()

		assertTailZeroed(t, q)
	})

	t.Run("usable after Reset", func(t *testing.T) {
		q := NewWithCap[string, int](8)
		q.Push("a", 1)
		q.Reset()

		q.Push("c", 3)

		assertInvariants(t, q)

		if got, p, ok := q.Pop(); !ok || got != "c" || p != 3 {
			t.Errorf("Pop() after Reset() and Push(c, 3) = (%q, %d, %v), "+
				"want (\"c\", 3, true)", got, p, ok)
		}
	})
}

func TestQueue_Shrink(t *testing.T) {
	t.Run("reduces capacity to length and keeps contents", func(t *testing.T) {
		q := fill(5)
		q.Grow(50)
		want := q.Sorted()

		q.Shrink()

		if got, want := q.Cap(), q.Len(); got != want {
			t.Errorf("Cap() after Shrink() = %d, want Len() = %d", got, want)
		}

		if got := q.Sorted(); !slices.Equal(got, want) {
			t.Errorf("Sorted() after Shrink() = %v, want %v", got, want)
		}

		assertInvariants(t, q)
	})

	t.Run("allocates a new backing array", func(t *testing.T) {
		q := fill(5)
		q.Grow(50)
		old := &q.data[0]

		q.Shrink()

		if got := &q.data[0]; got == old {
			t.Errorf("&data[0] after Shrink() = %p, want a new backing array",
				got)
		}
	})

	t.Run("rebuilds index", func(t *testing.T) {
		q := fill(100)
		q.PopN(90)

		q.Shrink()

		assertInvariants(t, q)

		if got := q.Contains(95); !got {
			t.Errorf("Contains(95) after Shrink() = %v, want true", got)
		}
	})

	t.Run("no-op when tight", func(t *testing.T) {
		q := fill(5)
		q.Shrink()
		old := &q.data[0]

		q.Shrink()

		if got := &q.data[0]; got != old {
			t.Errorf("&data[0] after Shrink() = %p, want %p (no copy)",
				got, old)
		}

		assertInvariants(t, q)
	})

	t.Run("empty queue drops to zero capacity", func(t *testing.T) {
		q := NewWithCap[int, int](8)

		q.Shrink()

		if got := q.Cap(); got != 0 {
			t.Errorf("Cap() after Shrink() on empty = %d, want 0", got)
		}
	})

	t.Run("usable after Shrink", func(t *testing.T) {
		q := fill(5)
		q.Shrink()

		q.Push(99, -1)

		if got, p, ok := q.Pop(); !ok || got != 99 || p != -1 {
			t.Errorf("Pop() after Shrink() and Push(99, -1) = "+
				"(%d, %d, %v), want (99, -1, true)", got, p, ok)
		}
	})
}

func TestQueue_Clip(t *testing.T) {
	t.Run("reduces capacity to length and keeps contents", func(t *testing.T) {
		q := fill(5)
		q.Grow(50)
		want := q.Sorted()

		q.Clip()

		if got, want := q.Cap(), q.Len(); got != want {
			t.Errorf("Cap() after Clip() = %d, want Len() = %d", got, want)
		}

		if got := q.Sorted(); !slices.Equal(got, want) {
			t.Errorf("Sorted() after Clip() = %v, want %v", got, want)
		}

		assertInvariants(t, q)
	})

	t.Run("does not reallocate", func(t *testing.T) {
		q := fill(5)
		q.Grow(50)
		old := &q.data[0]

		q.Clip()

		if got := &q.data[0]; got != old {
			t.Errorf("&data[0] after Clip() = %p, want %p (no copy)", got, old)
		}
	})

	t.Run("empty queue drops to zero capacity", func(t *testing.T) {
		q := NewWithCap[int, int](8)

		q.Clip()

		if got := q.Cap(); got != 0 {
			t.Errorf("Cap() after Clip() on empty = %d, want 0", got)
		}
	})

	t.Run("usable after Clip", func(t *testing.T) {
		q := fill(5)
		q.Clip()

		q.Push(99, -1)

		if got, p, ok := q.Pop(); !ok || got != 99 || p != -1 {
			t.Errorf("Pop() after Clip() and Push(99, -1) = (%d, %d, %v), "+
				"want (99, -1, true)", got, p, ok)
		}
	})
}

func TestQueue_Grow(t *testing.T) {
	t.Run("reserves capacity without changing contents", func(t *testing.T) {
		q := fill(3)
		want := q.Sorted()

		q.Grow(100)

		if got := q.Cap(); got < 103 {
			t.Errorf("Cap() after Grow(100) = %d, want >= 103", got)
		}

		if got := q.Sorted(); !slices.Equal(got, want) {
			t.Errorf("Sorted() after Grow(100) = %v, want %v", got, want)
		}
	})

	t.Run("no reallocation within grown capacity", func(t *testing.T) {
		q := New[int, int]()
		q.Grow(8)
		capBefore := q.Cap()

		for i := range 8 {
			q.Push(i, i)
		}

		if got := q.Cap(); got != capBefore {
			t.Errorf("Cap() after 8 Push() = %d, want %d", got, capBefore)
		}
	})

	t.Run("non-positive n is a no-op", func(t *testing.T) {
		q := fill(3)
		capBefore := q.Cap()

		q.Grow(0)

		if got := q.Cap(); got != capBefore {
			t.Errorf("Cap() after Grow(0) = %d, want %d", got, capBefore)
		}

		q.Grow(-1)

		if got := q.Cap(); got != capBefore {
			t.Errorf("Cap() after Grow(-1) = %d, want %d", got, capBefore)
		}
	})
}

// TestQueue_RandomOps runs a seeded random sequence of operations against a
// map model and checks that both agree after every step.
func TestQueue_RandomOps(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	q := New[int, int]()
	model := map[int]int{} // key -> priority

	for i := range 20000 {
		k := r.IntN(200)
		switch op := r.IntN(10); {
		case op < 4:
			_, existed := model[k]
			p := r.IntN(100)
			model[k] = p

			if got := q.Push(k, p); got == existed {
				t.Fatalf("step %d: Push(%d, %d) = %v, want %v",
					i, k, p, got, !existed)
			}
		case op < 5:
			batch := []Item[int, int]{
				{k, r.IntN(100)},
				{r.IntN(200), r.IntN(100)},
				{k, r.IntN(100)},
			}
			for _, it := range batch {
				model[it.Key] = it.Priority
			}

			q.PushN(batch...)
		case op < 8:
			got, p, ok := q.Pop()
			if ok != (len(model) > 0) {
				t.Fatalf("step %d: Pop() = (%d, %d, %v), want ok = %v",
					i, got, p, ok, len(model) > 0)
			}

			if !ok {
				break
			}

			for _, mp := range model {
				if mp < p {
					t.Fatalf("step %d: Pop() priority = %d, want <= %d",
						i, p, mp)
				}
			}

			if want := model[got]; p != want {
				t.Fatalf("step %d: Pop() = (%d, %d), want priority %d",
					i, got, p, want)
			}

			delete(model, got)
		case op < 9:
			want, wantOK := model[k]
			delete(model, k)

			if got, ok := q.Remove(k); ok != wantOK || got != want {
				t.Fatalf("step %d: Remove(%d) = (%d, %v), want (%d, %v)",
					i, k, got, ok, want, wantOK)
			}
		default:
			switch r.IntN(5) {
			case 0:
				q.Clear()
				clear(model)
			case 1:
				q.Reset()
				clear(model)
			case 2:
				q.Shrink()
			case 3:
				q.Clip()
			default:
				q.Grow(r.IntN(16))
			}
		}

		assertInvariants(t, q)

		if got := q.Len(); got != len(model) {
			t.Fatalf("step %d: Len() = %d, want %d", i, got, len(model))
		}
	}
}

// TestQueue_NilReceiver checks that every read-only and removal method treats
// a nil receiver as an empty priority queue.
func TestQueue_NilReceiver(t *testing.T) {
	var q *Queue[string, int]

	if got := q.Len(); got != 0 {
		t.Errorf("nil.Len() = %d, want 0", got)
	}

	if got := q.Cap(); got != 0 {
		t.Errorf("nil.Cap() = %d, want 0", got)
	}

	if got := q.IsEmpty(); !got {
		t.Errorf("nil.IsEmpty() = %v, want true", got)
	}

	if got, p, ok := q.Peek(); ok || got != "" || p != 0 {
		t.Errorf("nil.Peek() = (%q, %d, %v), want (\"\", 0, false)",
			got, p, ok)
	}

	if got := q.Contains("a"); got {
		t.Errorf("nil.Contains(a) = %v, want false", got)
	}

	if got, ok := q.Priority("a"); ok || got != 0 {
		t.Errorf("nil.Priority(a) = (%d, %v), want (0, false)", got, ok)
	}

	count := 0
	for range q.All() {
		count++
	}

	if count != 0 {
		t.Errorf("nil.All() count = %d, want 0", count)
	}

	if got := q.Clone(); got != nil {
		t.Errorf("nil.Clone() = %p, want nil", got)
	}

	if got := q.Slice(); got != nil {
		t.Errorf("nil.Slice() = %v, want nil", got)
	}

	if got := q.Sorted(); got != nil {
		t.Errorf("nil.Sorted() = %v, want nil", got)
	}

	if got, p, ok := q.Pop(); ok || got != "" || p != 0 {
		t.Errorf("nil.Pop() = (%q, %d, %v), want (\"\", 0, false)",
			got, p, ok)
	}

	if got := q.PopN(3); got != nil {
		t.Errorf("nil.PopN(3) = %v, want nil", got)
	}

	if got, ok := q.Remove("a"); ok || got != 0 {
		t.Errorf("nil.Remove(a) = (%d, %v), want (0, false)", got, ok)
	}
}

// TestQueue_NilReceiverMutatorsPanic checks that every method that stores,
// clears, resets, or resizes data panics on a nil receiver.
func TestQueue_NilReceiverMutatorsPanic(t *testing.T) {
	tests := []struct {
		name string
		op   string
		call func(*Queue[string, int])
	}{
		{
			name: "Push",
			op:   "Push",
			call: func(q *Queue[string, int]) { q.Push("a", 1) },
		},
		{
			name: "PushN",
			op:   "PushN",
			call: func(q *Queue[string, int]) {
				q.PushN(Item[string, int]{"a", 1})
			},
		},
		{
			name: "PushN with no arguments",
			op:   "PushN",
			call: func(q *Queue[string, int]) { q.PushN() },
		},
		{
			name: "Clear",
			op:   "Clear",
			call: func(q *Queue[string, int]) { q.Clear() },
		},
		{
			name: "Reset",
			op:   "Reset",
			call: func(q *Queue[string, int]) { q.Reset() },
		},
		{
			name: "Shrink",
			op:   "Shrink",
			call: func(q *Queue[string, int]) { q.Shrink() },
		},
		{
			name: "Clip",
			op:   "Clip",
			call: func(q *Queue[string, int]) { q.Clip() },
		},
		{
			name: "Grow",
			op:   "Grow",
			call: func(q *Queue[string, int]) { q.Grow(8) },
		},
		{
			name: "Grow zero",
			op:   "Grow",
			call: func(q *Queue[string, int]) { q.Grow(0) },
		},
		{
			name: "Grow negative",
			op:   "Grow",
			call: func(q *Queue[string, int]) { q.Grow(-1) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var q *Queue[string, int]
			sharedtest.Panics(t, tt.name, "pqueue: "+tt.op+" on nil receiver",
				func() { tt.call(q) })
		})
	}
}

func TestQueue_ZeroValue(t *testing.T) {
	var q Queue[string, int]

	if got := q.Len(); got != 0 {
		t.Errorf("Len() on zero value = %d, want 0", got)
	}

	if got := q.Cap(); got != 0 {
		t.Errorf("Cap() on zero value = %d, want 0", got)
	}

	if got := q.IsEmpty(); !got {
		t.Errorf("IsEmpty() on zero value = %v, want true", got)
	}

	if got, p, ok := q.Peek(); ok || got != "" || p != 0 {
		t.Errorf("Peek() on zero value = (%q, %d, %v), "+
			"want (\"\", 0, false)", got, p, ok)
	}

	if got := q.Contains("a"); got {
		t.Errorf("Contains(a) on zero value = %v, want false", got)
	}

	c := q.Clone()
	if c == nil {
		t.Fatalf("Clone() on zero value = nil, want an empty queue")
	}

	if got := c.Len(); got != 0 {
		t.Errorf("Len() of Clone() on zero value = %d, want 0", got)
	}

	if got, p, ok := q.Pop(); ok || got != "" || p != 0 {
		t.Errorf("Pop() on zero value = (%q, %d, %v), "+
			"want (\"\", 0, false)", got, p, ok)
	}

	if got, ok := q.Remove("a"); ok || got != 0 {
		t.Errorf("Remove(a) on zero value = (%d, %v), want (0, false)",
			got, ok)
	}

	q.Clear()
	q.Reset()
	q.Shrink()
	q.Clip()
	q.Grow(4)

	if got := q.Cap(); got < 4 {
		t.Errorf("Cap() after Grow(4) = %d, want >= 4", got)
	}

	sharedtest.Panics(t, "Push on zero value",
		"pqueue: Push on zero-value Queue", func() { q.Push("a", 1) })
	sharedtest.Panics(t, "PushN on zero value",
		"pqueue: PushN on zero-value Queue", func() { q.PushN() })
}
