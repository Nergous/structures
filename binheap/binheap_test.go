package binheap

import (
	"cmp"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Nergous/structures/internal/shared/sharedtest"
)

// assertNewEmpty fails the test unless h is an empty heap with no
// preallocated capacity.
func assertNewEmpty[T any](t *testing.T, h *Heap[T]) {
	t.Helper()

	if got := h.Len(); got != 0 {
		t.Errorf("Len() = %d, want 0", got)
	}

	if got := h.Cap(); got != 0 {
		t.Errorf("Cap() = %d, want 0", got)
	}
}

// assertTailZeroed fails the test unless every slot of the backing array
// between Len and Cap holds the zero value of T.
func assertTailZeroed[T comparable](t *testing.T, h *Heap[T]) {
	t.Helper()

	var zero T
	for i, v := range h.data[len(h.data):cap(h.data)] {
		if v != zero {
			t.Errorf("data[%d] = %v, want zero value", len(h.data)+i, v)
		}
	}
}

// drain pops every element of h and returns them in pop order.
func drain[T any](h *Heap[T]) []T {
	var out []T
	for {
		v, ok := h.Pop()
		if !ok {
			return out
		}

		out = append(out, v)
	}
}

// byPtr orders pointers by the values they point to.
func byPtr(a, b *int) int { return cmp.Compare(*a, *b) }

// ptrs returns pointers to copies of vs, in argument order.
func ptrs(vs ...int) []*int {
	out := make([]*int, len(vs))
	for i := range vs {
		out[i] = &vs[i]
	}

	return out
}

func TestHeap_New(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		assertNewEmpty(t, New[int]())
	})

	t.Run("string", func(t *testing.T) {
		assertNewEmpty(t, New[string]())
	})

	t.Run("float32", func(t *testing.T) {
		assertNewEmpty(t, New[float32]())
	})
}

func TestHeap_NewWithCap(t *testing.T) {
	constructors := []struct {
		name string
		newH func(n int) *Heap[int]
	}{
		{
			name: "NewWithCap",
			newH: NewWithCap[int],
		},
		{
			name: "NewMaxWithCap",
			newH: NewMaxWithCap[int],
		},
		{
			name: "NewFuncWithCap",
			newH: func(n int) *Heap[int] {
				return NewFuncWithCap(n, cmp.Compare[int])
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
					h := ctor.newH(tt.n)

					if got := h.Len(); got != 0 {
						t.Errorf("Len() = %d, want 0", got)
					}

					if got := h.Cap(); got != tt.want {
						t.Errorf("Cap() = %d, want %d", got, tt.want)
					}

					h.Push(1)

					if got, ok := h.Pop(); !ok || got != 1 {
						t.Errorf("Pop() after Push(1) = (%d, %v), "+
							"want (1, true)", got, ok)
					}
				})
			}
		})
	}

	t.Run("nil comparator panics", func(t *testing.T) {
		sharedtest.Panics(t, "NewFuncWithCap(4, nil)",
			"binheap: nil comparator",
			func() { NewFuncWithCap[int](4, nil) })
	})
}

func TestHeap_From(t *testing.T) {
	tests := []struct {
		name string
		s    []int
	}{
		{
			name: "nil",
			s:    nil,
		},
		{
			name: "empty",
			s:    []int{},
		},
		{
			name: "one",
			s:    []int{1},
		},
		{
			name: "sorted",
			s:    []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			name: "reversed",
			s:    []int{10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
		},
		{
			name: "shuffled",
			s:    []int{1, 6, 2, 7, 3, 8, 4, 9, 5, 10},
		},
		{
			name: "duplicates",
			s:    []int{3, 1, 3, 1, 2, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := From(tt.s)

			sharedtest.Heap(t, h.data, h.compare)

			if got := h.Len(); got != len(tt.s) {
				t.Errorf("Len() = %d, want %d", got, len(tt.s))
			}

			if got := h.Cap(); got != len(tt.s) {
				t.Errorf("Cap() = %d, want %d", got, len(tt.s))
			}

			want := slices.Sorted(slices.Values(tt.s))
			if got := drain(h); !slices.Equal(got, want) {
				t.Errorf("Pop() order = %v, want %v", got, want)
			}
		})
	}

	t.Run("never aliases the input", func(t *testing.T) {
		s := []int{5, 3, 8, 1}
		h := From(s)

		s[0], s[3] = -100, -200

		if got, ok := h.Peek(); !ok || got != 1 {
			t.Errorf("Peek() after changing input = (%d, %v), "+
				"want (1, true)", got, ok)
		}

		h.Push(0)

		if want := []int{-100, 3, 8, -200}; !slices.Equal(s, want) {
			t.Errorf("input after Push(0) = %v, want %v", s, want)
		}
	})
}

func TestHeap_NewMax(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		assertNewEmpty(t, NewMax[int]())
	})

	t.Run("string", func(t *testing.T) {
		assertNewEmpty(t, NewMax[string]())
	})

	t.Run("float32", func(t *testing.T) {
		assertNewEmpty(t, NewMax[float32]())
	})
}

func TestHeap_FromMax(t *testing.T) {
	h := FromMax([]int{1, 6, 2, 7, 3, 8, 4, 9, 5, 10})

	sharedtest.Heap(t, h.data, h.compare)

	want := []int{10, 9, 8, 7, 6, 5, 4, 3, 2, 1}
	if got := drain(h); !slices.Equal(got, want) {
		t.Errorf("Pop() order = %v, want %v", got, want)
	}
}

func TestHeap_NewFunc(t *testing.T) {
	t.Run("returns an empty heap", func(t *testing.T) {
		assertNewEmpty(t, NewFunc(cmp.Compare[int]))
	})

	t.Run("orders by comparator", func(t *testing.T) {
		h := NewFunc(func(a, b string) int {
			return cmp.Compare(len(a), len(b))
		})
		h.PushN("ccc", "a", "dddd", "bb")

		want := []string{"a", "bb", "ccc", "dddd"}
		if got := drain(h); !slices.Equal(got, want) {
			t.Errorf("Pop() order = %v, want %v", got, want)
		}
	})

	t.Run("nil comparator panics", func(t *testing.T) {
		sharedtest.Panics(t, "NewFunc(nil)", "binheap: nil comparator",
			func() { NewFunc[int](nil) })
	})
}

func TestHeap_FromFunc(t *testing.T) {
	t.Run("orders by comparator", func(t *testing.T) {
		byLen := func(a, b string) int {
			return cmp.Compare(len(a), len(b))
		}
		h := FromFunc([]string{"ccc", "a", "dddd", "bb"}, byLen)

		sharedtest.Heap(t, h.data, h.compare)

		want := []string{"a", "bb", "ccc", "dddd"}
		if got := drain(h); !slices.Equal(got, want) {
			t.Errorf("Pop() order = %v, want %v", got, want)
		}
	})

	t.Run("nil comparator panics", func(t *testing.T) {
		sharedtest.Panics(t, "FromFunc(s, nil)", "binheap: nil comparator",
			func() { FromFunc([]int{1}, nil) })
	})
}

func TestHeap_Len(t *testing.T) {
	h := New[int]()
	h.Push(1)
	h.Push(2)
	h.Push(3)

	if got := h.Len(); got != 3 {
		t.Errorf("Len() after three Push() = %d, want 3", got)
	}

	h.Pop()

	if got := h.Len(); got != 2 {
		t.Errorf("Len() after Pop() = %d, want 2", got)
	}
}

func TestHeap_Cap(t *testing.T) {
	t.Run("new heap has zero cap", func(t *testing.T) {
		h := New[int]()

		if got := h.Cap(); got != 0 {
			t.Errorf("Cap() = %d, want 0", got)
		}
	})

	t.Run("never less than Len while growing", func(t *testing.T) {
		h := New[int]()
		for i := range 100 {
			h.Push(i)
			if c, l := h.Cap(), h.Len(); c < l {
				t.Fatalf("Cap() #%d = %d, want >= Len() = %d", i, c, l)
			}
		}
	})

	t.Run("no reallocation within preallocated capacity", func(t *testing.T) {
		h := NewWithCap[int](4)
		for i := range 4 {
			h.Push(i)
		}

		if got := h.Cap(); got != 4 {
			t.Errorf("Cap() after 4 Push() = %d, want 4", got)
		}
	})
}

func TestHeap_IsEmpty(t *testing.T) {
	h := New[int]()

	if got := h.IsEmpty(); !got {
		t.Errorf("IsEmpty() on empty = %v, want true", got)
	}

	h.Push(1)

	if got := h.IsEmpty(); got {
		t.Errorf("IsEmpty() after Push(1) = %v, want false", got)
	}

	h.Pop()

	if got := h.IsEmpty(); !got {
		t.Errorf("IsEmpty() after Pop() = %v, want true", got)
	}
}

func TestHeap_Peek(t *testing.T) {
	t.Run("empty returns zero value and false", func(t *testing.T) {
		h := New[int]()

		if got, ok := h.Peek(); ok || got != 0 {
			t.Errorf("Peek() on empty = (%d, %v), want (0, false)", got, ok)
		}
	})

	t.Run("returns least without removing it", func(t *testing.T) {
		h := From([]int{4, 2, 9})

		if got, ok := h.Peek(); !ok || got != 2 {
			t.Errorf("Peek() = (%d, %v), want (2, true)", got, ok)
		}

		if got := h.Len(); got != 3 {
			t.Errorf("Len() after Peek() = %d, want 3", got)
		}
	})
}

func TestHeap_Contains(t *testing.T) {
	h := From([]int{4, 8, 15, 16, 23, 42})

	if got := h.Contains(15); !got {
		t.Errorf("Contains(15) = %v, want true", got)
	}

	if got := h.Contains(7); got {
		t.Errorf("Contains(7) = %v, want false", got)
	}

	h.Remove(15)

	if got := h.Contains(15); got {
		t.Errorf("Contains(15) after Remove(15) = %v, want false", got)
	}

	type task struct {
		name     string
		priority int
	}

	byPriority := NewFunc(func(a, b task) int {
		return cmp.Compare(a.priority, b.priority)
	})
	byPriority.Push(task{name: "a", priority: 1})

	if got := byPriority.Contains(task{name: "b", priority: 1}); !got {
		t.Errorf("Contains(same priority) = %v, want true "+
			"(match by comparator)", got)
	}
}

func TestHeap_All(t *testing.T) {
	t.Run("yields layout order with least first", func(t *testing.T) {
		h := From([]int{5, 3, 8, 1, 9})

		got := slices.Collect(h.All())
		if !slices.Equal(got, h.data) {
			t.Errorf("All() = %v, want layout order %v", got, h.data)
		}

		if got[0] != 1 {
			t.Errorf("All() first = %d, want least element 1", got[0])
		}
	})

	t.Run("empty heap yields nothing", func(t *testing.T) {
		h := New[int]()

		if got := slices.Collect(h.All()); len(got) != 0 {
			t.Errorf("All() on empty = %v, want no elements", got)
		}
	})

	t.Run("does not consume", func(t *testing.T) {
		h := From([]int{5, 3, 8, 1, 9})

		_ = slices.Collect(h.All())

		if got := h.Len(); got != 5 {
			t.Errorf("Len() after All() = %d, want 5", got)
		}

		sharedtest.Heap(t, h.data, h.compare)
	})

	t.Run("break stops iteration early", func(t *testing.T) {
		h := From([]int{5, 3, 8, 1, 9})

		var got []int
		for v := range h.All() {
			got = append(got, v)
			if len(got) == 2 {
				break
			}
		}

		if want := h.data[:2]; !slices.Equal(got, want) {
			t.Errorf("All() with early break = %v, want %v", got, want)
		}

		if got := h.Len(); got != 5 {
			t.Errorf("Len() after break = %d, want 5", got)
		}
	})
}

func TestHeap_Clone(t *testing.T) {
	t.Run("copies all elements in order", func(t *testing.T) {
		h := From([]int{5, 3, 8, 1, 9})

		c := h.Clone()

		if got, want := c.Slice(), h.Slice(); !slices.Equal(got, want) {
			t.Errorf("Slice() of clone = %v, want %v", got, want)
		}
	})

	t.Run("capacity equals length", func(t *testing.T) {
		h := NewWithCap[int](64)
		h.PushN(1, 2, 3, 4, 5)

		c := h.Clone()

		if got, want := c.Cap(), c.Len(); got != want {
			t.Errorf("Cap() of clone = %d, want Len() = %d", got, want)
		}
	})

	t.Run("is independent of the original", func(t *testing.T) {
		h := NewMaxWithCap[int](8)
		h.PushN(1, 2, 3)

		c := h.Clone()
		c.Pop()
		c.Push(0)

		if got, want := c.Sorted(), []int{2, 1, 0}; !slices.Equal(got, want) {
			t.Errorf("Sorted() of clone = %v, want %v", got, want)
		}

		if got, want := h.Sorted(), []int{3, 2, 1}; !slices.Equal(got, want) {
			t.Errorf("Sorted() of original = %v, want %v", got, want)
		}
	})

	t.Run("clone of empty heap is empty and usable", func(t *testing.T) {
		h := New[int]()

		c := h.Clone()

		if got := c.Len(); got != 0 {
			t.Errorf("Len() of clone = %d, want 0", got)
		}

		c.Push(1)

		if got, ok := c.Pop(); !ok || got != 1 {
			t.Errorf("Pop() of clone after Push(1) = (%d, %v), "+
				"want (1, true)", got, ok)
		}
	})

	t.Run("keeps max comparator", func(t *testing.T) {
		h := FromMax([]int{1, 2, 3})

		c := h.Clone()
		c.Push(5)

		want := []int{5, 3, 2, 1}
		if got := c.Sorted(); !slices.Equal(got, want) {
			t.Errorf("Sorted() of clone after Push(5) = %v, want %v",
				got, want)
		}
	})
}

func TestHeap_Slice(t *testing.T) {
	t.Run("returns layout order", func(t *testing.T) {
		h := From([]int{5, 3, 8, 1})

		if got := h.Slice(); !slices.Equal(got, h.data) {
			t.Errorf("Slice() = %v, want layout order %v", got, h.data)
		}
	})

	t.Run("returns an independent copy", func(t *testing.T) {
		h := From([]int{5, 3, 8, 1})
		want := h.Slice()

		out := h.Slice()
		for i := range out {
			out[i] = 999
		}

		if got := h.Slice(); !slices.Equal(got, want) {
			t.Errorf("Slice() after mutating a copy = %v, want %v", got, want)
		}
	})

	t.Run("empty heap returns nil", func(t *testing.T) {
		h := New[int]()

		if got := h.Slice(); got != nil {
			t.Errorf("Slice() on empty = %v, want nil", got)
		}
	})
}

func TestHeap_Sorted(t *testing.T) {
	t.Run("min heap ascending without consuming", func(t *testing.T) {
		h := From([]int{5, 3, 8, 1, 3})

		want := []int{1, 3, 3, 5, 8}
		if got := h.Sorted(); !slices.Equal(got, want) {
			t.Errorf("Sorted() = %v, want %v", got, want)
		}

		if got := h.Len(); got != 5 {
			t.Errorf("Len() after Sorted() = %d, want 5", got)
		}

		sharedtest.Heap(t, h.data, h.compare)
	})

	t.Run("max heap descending", func(t *testing.T) {
		h := FromMax([]int{1, 8, 3, 5})

		want := []int{8, 5, 3, 1}
		if got := h.Sorted(); !slices.Equal(got, want) {
			t.Errorf("Sorted() on max heap = %v, want %v", got, want)
		}
	})

	t.Run("empty heap returns nil", func(t *testing.T) {
		h := New[int]()

		if got := h.Sorted(); got != nil {
			t.Errorf("Sorted() on empty = %v, want nil", got)
		}
	})
}

func TestHeap_Push(t *testing.T) {
	h := New[int]()

	for i, v := range []int{5, 3, 8, 1, 9} {
		h.Push(v)

		sharedtest.Heap(t, h.data, h.compare)

		if got := h.Len(); got != i+1 {
			t.Errorf("Len() after Push(%d) = %d, want %d", v, got, i+1)
		}
	}

	if got, ok := h.Peek(); !ok || got != 1 {
		t.Errorf("Peek() after Push() = (%d, %v), want (1, true)", got, ok)
	}
}

func TestHeap_PushN(t *testing.T) {
	tests := []struct {
		name    string
		initial []int
		push    []int
	}{
		{
			name:    "no arguments",
			initial: []int{3, 1, 2},
			push:    nil,
		},
		{
			name:    "into empty",
			initial: nil,
			push:    []int{5, 3, 9, 1, 7, 2, 8},
		},
		{
			name:    "small batch",
			initial: []int{10, 20, 30, 40, 50, 60, 70, 80},
			push:    []int{5, 45},
		},
		{
			name:    "large batch",
			initial: []int{10, 20},
			push:    []int{9, 8, 7, 6, 5, 4, 3, 2, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := From(tt.initial)

			h.PushN(tt.push...)

			sharedtest.Heap(t, h.data, h.compare)

			all := slices.Concat(tt.initial, tt.push)
			want := slices.Sorted(slices.Values(all))
			if got := drain(h); !slices.Equal(got, want) {
				t.Errorf("Pop() order after PushN() = %v, want %v", got, want)
			}
		})
	}

	t.Run("uses both strategies", func(t *testing.T) {
		// A 6-element batch into a large heap must sift up; the same batch
		// into a tiny heap must rebuild. Both have to keep the heap valid.
		if got := rebuildCheaper(1<<16, 6); got {
			t.Errorf("rebuildCheaper(65536, 6) = %v, "+
				"want false (sift-up for a small batch)", got)
		}

		if got := rebuildCheaper(4, 6); !got {
			t.Errorf("rebuildCheaper(4, 6) = %v, "+
				"want true (rebuild for a batch larger than the heap)", got)
		}

		r := rand.New(rand.NewPCG(1, 2))
		for _, n := range []int{0, 4, 1 << 12} {
			s := make([]int, n)
			for i := range s {
				s[i] = r.IntN(1000)
			}

			h := From(s)
			batch := []int{
				r.IntN(1000), r.IntN(1000), -1,
				r.IntN(1000), r.IntN(1000), r.IntN(1000),
			}

			h.PushN(batch...)

			sharedtest.Heap(t, h.data, h.compare)

			if got, ok := h.Peek(); !ok || got != -1 {
				t.Errorf("Peek() after PushN() into %d elements = (%d, %v), "+
					"want (-1, true)", n, got, ok)
			}
		}
	})
}

func TestHeap_Pop(t *testing.T) {
	tests := []struct {
		name string
		newH func() *Heap[int]
		push []int
		want []int
	}{
		{
			name: "empty",
			newH: New[int],
			push: nil,
			want: nil,
		},
		{
			name: "single element",
			newH: New[int],
			push: []int{42},
			want: []int{42},
		},
		{
			name: "min order",
			newH: New[int],
			push: []int{5, 1, 4, 2, 3},
			want: []int{1, 2, 3, 4, 5},
		},
		{
			name: "min order with duplicates",
			newH: New[int],
			push: []int{2, 1, 2, 1},
			want: []int{1, 1, 2, 2},
		},
		{
			name: "max order",
			newH: NewMax[int],
			push: []int{5, 1, 4, 2, 3},
			want: []int{5, 4, 3, 2, 1},
		},
		{
			name: "comparator order",
			newH: func() *Heap[int] {
				return NewFunc(func(a, b int) int {
					return cmp.Compare(a%10, b%10)
				})
			},
			push: []int{19, 21, 35, 3},
			want: []int{21, 3, 35, 19},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := tt.newH()
			h.PushN(tt.push...)

			for i, want := range tt.want {
				if got, ok := h.Pop(); !ok || got != want {
					t.Errorf("Pop() #%d = (%d, %v), want (%d, true)",
						i, got, ok, want)
				}

				sharedtest.Heap(t, h.data, h.compare)
			}

			if got, ok := h.Pop(); ok || got != 0 {
				t.Errorf("Pop() on empty = (%d, %v), want (0, false)",
					got, ok)
			}
		})
	}

	t.Run("zeroes vacated slot", func(t *testing.T) {
		h := FromFunc(ptrs(3, 1, 2), byPtr)
		data := h.data[:3]

		if _, ok := h.Pop(); !ok {
			t.Fatalf("Pop() ok = false, want true")
		}

		if got := data[2]; got != nil {
			t.Errorf("data[2] after Pop() = %p, "+
				"want nil (Pop must zero the slot)", got)
		}
	})
}

func TestHeap_PopN(t *testing.T) {
	five := []int{5, 3, 4, 1, 2}
	sorted := []int{1, 2, 3, 4, 5}

	tests := []struct {
		name string
		push []int
		n    int
		want []int
		left []int
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
			h := New[int]()
			h.PushN(tt.push...)
			capBefore := h.Cap()

			got := h.PopN(tt.n)

			if !slices.Equal(got, tt.want) || (got == nil) != (tt.want == nil) {
				t.Errorf("PopN(%d) = %v, want %v", tt.n, got, tt.want)
			}

			if got := h.Sorted(); !slices.Equal(got, tt.left) {
				t.Errorf("Sorted() after PopN(%d) = %v, want %v",
					tt.n, got, tt.left)
			}

			if got := h.Cap(); got != capBefore {
				t.Errorf("Cap() after PopN(%d) = %d, want %d",
					tt.n, got, capBefore)
			}

			sharedtest.Heap(t, h.data, h.compare)
		})
	}

	t.Run("zeroes vacated slots", func(t *testing.T) {
		h := FromFunc(ptrs(4, 1, 3, 2), byPtr)

		h.PopN(3)

		assertTailZeroed(t, h)
	})
}

func TestHeap_PushPop(t *testing.T) {
	tests := []struct {
		name     string
		newH     func() *Heap[int]
		v        int
		want     int
		wantHeap []int
	}{
		{
			name:     "empty returns v untouched",
			newH:     New[int],
			v:        7,
			want:     7,
			wantHeap: nil,
		},
		{
			name:     "v below root returns v",
			newH:     func() *Heap[int] { return From([]int{5, 10, 20}) },
			v:        1,
			want:     1,
			wantHeap: []int{5, 10, 20},
		},
		{
			name:     "v equal to root returns v",
			newH:     func() *Heap[int] { return From([]int{5, 10, 20}) },
			v:        5,
			want:     5,
			wantHeap: []int{5, 10, 20},
		},
		{
			name:     "v above root replaces root",
			newH:     func() *Heap[int] { return From([]int{5, 10, 20}) },
			v:        15,
			want:     5,
			wantHeap: []int{10, 15, 20},
		},
		{
			name:     "max heap",
			newH:     func() *Heap[int] { return FromMax([]int{5, 10, 20}) },
			v:        12,
			want:     20,
			wantHeap: []int{12, 10, 5},
		},
		{
			name:     "max heap v above root returns v",
			newH:     func() *Heap[int] { return FromMax([]int{5, 10, 20}) },
			v:        30,
			want:     30,
			wantHeap: []int{20, 10, 5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := tt.newH()
			capBefore := h.Cap()

			if got := h.PushPop(tt.v); got != tt.want {
				t.Errorf("PushPop(%d) = %d, want %d", tt.v, got, tt.want)
			}

			sharedtest.Heap(t, h.data, h.compare)

			if got := h.Sorted(); !slices.Equal(got, tt.wantHeap) {
				t.Errorf("Sorted() after PushPop(%d) = %v, want %v",
					tt.v, got, tt.wantHeap)
			}

			if got := h.Cap(); got != capBefore {
				t.Errorf("Cap() after PushPop(%d) = %d, "+
					"want %d (PushPop must not grow)", tt.v, got, capBefore)
			}
		})
	}
}

func TestHeap_Replace(t *testing.T) {
	t.Run("empty stores v and reports false", func(t *testing.T) {
		h := New[int]()

		if got, ok := h.Replace(7); ok || got != 0 {
			t.Errorf("Replace(7) on empty = (%d, %v), want (0, false)",
				got, ok)
		}

		if got := h.Len(); got != 1 {
			t.Errorf("Len() after Replace(7) on empty = %d, want 1", got)
		}

		if got, ok := h.Peek(); !ok || got != 7 {
			t.Errorf("Peek() after Replace(7) on empty = (%d, %v), "+
				"want (7, true)", got, ok)
		}
	})

	t.Run("pops old root even when v is smaller", func(t *testing.T) {
		h := From([]int{5, 10, 20})

		if got, ok := h.Replace(1); !ok || got != 5 {
			t.Errorf("Replace(1) = (%d, %v), want (5, true)", got, ok)
		}

		sharedtest.Heap(t, h.data, h.compare)

		if got, want := h.Sorted(), []int{1, 10, 20}; !slices.Equal(got, want) {
			t.Errorf("Sorted() after Replace(1) = %v, want %v", got, want)
		}
	})

	t.Run("sifts larger v down", func(t *testing.T) {
		h := From([]int{5, 10, 20})

		if got, ok := h.Replace(30); !ok || got != 5 {
			t.Errorf("Replace(30) = (%d, %v), want (5, true)", got, ok)
		}

		sharedtest.Heap(t, h.data, h.compare)

		want := []int{10, 20, 30}
		if got := h.Sorted(); !slices.Equal(got, want) {
			t.Errorf("Sorted() after Replace(30) = %v, want %v", got, want)
		}
	})
}

func TestHeap_Remove(t *testing.T) {
	tests := []struct {
		name    string
		initial []int
		v       int
		found   bool
		want    []int
	}{
		{
			name:    "root",
			initial: []int{1, 2, 3, 4},
			v:       1,
			found:   true,
			want:    []int{2, 3, 4},
		},
		{
			name:    "last element",
			initial: []int{1, 2, 3, 4},
			v:       4,
			found:   true,
			want:    []int{1, 2, 3},
		},
		{
			name:    "only element",
			initial: []int{9},
			v:       9,
			found:   true,
			want:    nil,
		},
		{
			name:    "one of duplicates",
			initial: []int{2, 2, 1},
			v:       2,
			found:   true,
			want:    []int{1, 2},
		},
		{
			name:    "missing",
			initial: []int{1, 2, 3},
			v:       7,
			found:   false,
			want:    []int{1, 2, 3},
		},
		{
			name:    "empty",
			initial: nil,
			v:       1,
			found:   false,
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := From(tt.initial)

			if got := h.Remove(tt.v); got != tt.found {
				t.Errorf("Remove(%d) = %v, want %v", tt.v, got, tt.found)
			}

			sharedtest.Heap(t, h.data, h.compare)

			if got := drain(h); !slices.Equal(got, tt.want) {
				t.Errorf("Pop() order after Remove(%d) = %v, want %v",
					tt.v, got, tt.want)
			}
		})
	}

	t.Run("moved element sifts up", func(t *testing.T) {
		// Removing 11 moves 4 under 10; without a sift-up the heap breaks.
		h := &Heap[int]{
			data:    []int{1, 10, 2, 11, 12, 3, 4},
			compare: cmp.Compare[int],
		}

		sharedtest.Heap(t, h.data, h.compare)

		if got := h.Remove(11); !got {
			t.Fatalf("Remove(11) = %v, want true", got)
		}

		sharedtest.Heap(t, h.data, h.compare)

		want := []int{1, 2, 3, 4, 10, 12}
		if got := drain(h); !slices.Equal(got, want) {
			t.Errorf("Pop() order after Remove(11) = %v, want %v", got, want)
		}
	})

	t.Run("zeroes vacated slot", func(t *testing.T) {
		h := FromFunc(ptrs(1, 2, 3, 4), byPtr)
		v := 3

		if got := h.Remove(&v); !got {
			t.Fatalf("Remove(3) = %v, want true", got)
		}

		assertTailZeroed(t, h)
	})
}

func TestHeap_Clear(t *testing.T) {
	h := NewMaxWithCap[int](8)
	h.PushN(1, 2)

	h.Clear()

	if got := h.Len(); got != 0 {
		t.Errorf("Len() after Clear() = %d, want 0", got)
	}

	if got := h.Cap(); got != 0 {
		t.Errorf("Cap() after Clear() = %d, want 0", got)
	}

	if got, ok := h.Peek(); ok || got != 0 {
		t.Errorf("Peek() after Clear() = (%d, %v), want (0, false)", got, ok)
	}

	h.PushN(1, 3, 2)

	if got, ok := h.Peek(); !ok || got != 3 {
		t.Errorf("Peek() after Clear() and PushN(1, 3, 2) = (%d, %v), "+
			"want (3, true)", got, ok)
	}
}

func TestHeap_Reset(t *testing.T) {
	t.Run("Reset keeps capacity", func(t *testing.T) {
		h := NewWithCap[int](8)
		h.PushN(1, 2, 3)

		h.Reset()

		if got := h.Len(); got != 0 {
			t.Errorf("Len() after Reset() = %d, want 0", got)
		}

		if got := h.Cap(); got != 8 {
			t.Errorf("Cap() after Reset() = %d, want 8", got)
		}
	})

	t.Run("zeroes backing array", func(t *testing.T) {
		h := NewFuncWithCap(8, byPtr)
		h.PushN(ptrs(1, 2, 3)...)

		h.Reset()

		assertTailZeroed(t, h)
	})

	t.Run("usable after Reset", func(t *testing.T) {
		h := NewMaxWithCap[int](8)
		h.PushN(1, 2, 3)
		h.Reset()

		h.PushN(4, 5)

		if got, ok := h.Pop(); !ok || got != 5 {
			t.Errorf("Pop() after Reset() and PushN(4, 5) = (%d, %v), "+
				"want (5, true)", got, ok)
		}
	})
}

func TestHeap_Shrink(t *testing.T) {
	t.Run("reduces capacity to length and keeps contents", func(t *testing.T) {
		h := NewWithCap[int](64)
		h.PushN(3, 1, 2)

		h.Shrink()

		if got, want := h.Cap(), h.Len(); got != want {
			t.Errorf("Cap() after Shrink() = %d, want Len() = %d", got, want)
		}

		if got, want := h.Sorted(), []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("Sorted() after Shrink() = %v, want %v", got, want)
		}
	})

	t.Run("allocates a new backing array", func(t *testing.T) {
		h := NewWithCap[int](64)
		h.PushN(3, 1, 2)
		old := &h.data[0]

		h.Shrink()

		if got := &h.data[0]; got == old {
			t.Errorf("&data[0] after Shrink() = %p, want a new backing array",
				got)
		}
	})

	t.Run("no-op when tight", func(t *testing.T) {
		h := From([]int{3, 1, 2})
		old := &h.data[0]

		h.Shrink()

		if got := &h.data[0]; got != old {
			t.Errorf("&data[0] after Shrink() = %p, want %p (no copy)",
				got, old)
		}
	})

	t.Run("empty heap drops to zero capacity", func(t *testing.T) {
		h := NewWithCap[int](8)

		h.Shrink()

		if got := h.Cap(); got != 0 {
			t.Errorf("Cap() after Shrink() on empty = %d, want 0", got)
		}
	})

	t.Run("usable after Shrink", func(t *testing.T) {
		h := NewWithCap[int](16)
		h.Push(2)
		h.Shrink()

		h.Push(1)

		if got, ok := h.Pop(); !ok || got != 1 {
			t.Errorf("Pop() after Shrink() and Push(1) = (%d, %v), "+
				"want (1, true)", got, ok)
		}
	})
}

func TestHeap_Clip(t *testing.T) {
	t.Run("reduces capacity to length and keeps contents", func(t *testing.T) {
		h := NewWithCap[int](64)
		h.PushN(3, 1, 2)

		h.Clip()

		if got, want := h.Cap(), h.Len(); got != want {
			t.Errorf("Cap() after Clip() = %d, want Len() = %d", got, want)
		}

		if got, want := h.Sorted(), []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("Sorted() after Clip() = %v, want %v", got, want)
		}
	})

	t.Run("does not reallocate", func(t *testing.T) {
		h := NewWithCap[int](64)
		h.PushN(3, 1, 2)
		old := &h.data[0]

		h.Clip()

		if got := &h.data[0]; got != old {
			t.Errorf("&data[0] after Clip() = %p, want %p (no copy)", got, old)
		}
	})

	t.Run("empty heap drops to zero capacity", func(t *testing.T) {
		h := NewWithCap[int](8)

		h.Clip()

		if got := h.Cap(); got != 0 {
			t.Errorf("Cap() after Clip() on empty = %d, want 0", got)
		}
	})

	t.Run("usable after Clip", func(t *testing.T) {
		h := NewWithCap[int](16)
		h.Push(2)
		h.Clip()

		h.Push(1)

		if got, ok := h.Pop(); !ok || got != 1 {
			t.Errorf("Pop() after Clip() and Push(1) = (%d, %v), "+
				"want (1, true)", got, ok)
		}
	})
}

func TestHeap_Grow(t *testing.T) {
	t.Run("reserves capacity without changing contents", func(t *testing.T) {
		h := From([]int{3, 1, 2})

		h.Grow(100)

		if got := h.Cap(); got < 103 {
			t.Errorf("Cap() after Grow(100) = %d, want >= 103", got)
		}

		if got, want := h.Sorted(), []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("Sorted() after Grow(100) = %v, want %v", got, want)
		}
	})

	t.Run("no reallocation within grown capacity", func(t *testing.T) {
		h := New[int]()
		h.Grow(8)
		capBefore := h.Cap()

		for i := range 8 {
			h.Push(i)
		}

		if got := h.Cap(); got != capBefore {
			t.Errorf("Cap() after 8 Push() = %d, want %d", got, capBefore)
		}
	})

	t.Run("non-positive n is a no-op", func(t *testing.T) {
		h := From([]int{3, 1, 2})
		capBefore := h.Cap()

		h.Grow(0)

		if got := h.Cap(); got != capBefore {
			t.Errorf("Cap() after Grow(0) = %d, want %d", got, capBefore)
		}

		h.Grow(-1)

		if got := h.Cap(); got != capBefore {
			t.Errorf("Cap() after Grow(-1) = %d, want %d", got, capBefore)
		}
	})
}

// TestHeap_RandomOps runs a seeded random sequence of operations against a
// sorted slice model and checks that both agree after every step.
func TestHeap_RandomOps(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	h := New[int]()
	var model []int // sorted ascending

	for i := range 5000 {
		switch op := r.IntN(8); {
		case op < 2:
			v := r.IntN(100)
			h.Push(v)
			model = append(model, v)
		case op < 3:
			vs := make([]int, r.IntN(20))
			for j := range vs {
				vs[j] = r.IntN(100)
			}

			h.PushN(vs...)
			model = append(model, vs...)
		case op < 4:
			want, wantOK := 0, false
			if len(model) > 0 {
				want, wantOK = model[0], true
				model = model[1:]
			}

			if got, ok := h.Pop(); ok != wantOK || got != want {
				t.Fatalf("step %d: Pop() = (%d, %v), want (%d, %v)",
					i, got, ok, want, wantOK)
			}
		case op < 5:
			v := r.IntN(100)
			model = append(model, v)
			slices.Sort(model)
			want := model[0]
			model = model[1:]

			if got := h.PushPop(v); got != want {
				t.Fatalf("step %d: PushPop(%d) = %d, want %d",
					i, v, got, want)
			}
		case op < 6:
			v := r.IntN(100)
			want, wantOK := 0, false
			if len(model) > 0 {
				want, wantOK = model[0], true
				model = model[1:]
			}

			model = append(model, v)

			if got, ok := h.Replace(v); ok != wantOK || got != want {
				t.Fatalf("step %d: Replace(%d) = (%d, %v), want (%d, %v)",
					i, v, got, ok, want, wantOK)
			}
		case op < 7:
			v := r.IntN(100)
			j := slices.Index(model, v)
			if j >= 0 {
				model = slices.Delete(model, j, j+1)
			}

			if got := h.Remove(v); got != (j >= 0) {
				t.Fatalf("step %d: Remove(%d) = %v, want %v",
					i, v, got, j >= 0)
			}
		default:
			switch r.IntN(5) {
			case 0:
				h.Clear()
				model = nil
			case 1:
				h.Reset()
				model = nil
			case 2:
				h.Shrink()
			case 3:
				h.Clip()
			default:
				h.Grow(r.IntN(16))
			}
		}

		slices.Sort(model)

		sharedtest.Heap(t, h.data, h.compare)

		if got := h.Sorted(); !slices.Equal(got, model) {
			t.Fatalf("step %d: Sorted() = %v, want %v", i, got, model)
		}

		if got := h.Len(); got != len(model) {
			t.Fatalf("step %d: Len() = %d, want %d", i, got, len(model))
		}
	}
}

// TestHeap_NilReceiver checks that every read-only and removal method treats a
// nil receiver as an empty heap.
func TestHeap_NilReceiver(t *testing.T) {
	var h *Heap[int]

	if got := h.Len(); got != 0 {
		t.Errorf("nil.Len() = %d, want 0", got)
	}

	if got := h.Cap(); got != 0 {
		t.Errorf("nil.Cap() = %d, want 0", got)
	}

	if got := h.IsEmpty(); !got {
		t.Errorf("nil.IsEmpty() = %v, want true", got)
	}

	if got, ok := h.Peek(); ok || got != 0 {
		t.Errorf("nil.Peek() = (%d, %v), want (0, false)", got, ok)
	}

	if got := h.Contains(1); got {
		t.Errorf("nil.Contains(1) = %v, want false", got)
	}

	count := 0
	for range h.All() {
		count++
	}

	if count != 0 {
		t.Errorf("nil.All() count = %d, want 0", count)
	}

	if got := h.Clone(); got != nil {
		t.Errorf("nil.Clone() = %p, want nil", got)
	}

	if got := h.Slice(); got != nil {
		t.Errorf("nil.Slice() = %v, want nil", got)
	}

	if got := h.Sorted(); got != nil {
		t.Errorf("nil.Sorted() = %v, want nil", got)
	}

	if got, ok := h.Pop(); ok || got != 0 {
		t.Errorf("nil.Pop() = (%d, %v), want (0, false)", got, ok)
	}

	if got := h.PopN(3); got != nil {
		t.Errorf("nil.PopN(3) = %v, want nil", got)
	}

	if got := h.PushPop(5); got != 5 {
		t.Errorf("nil.PushPop(5) = %d, want 5", got)
	}

	if got := h.Remove(1); got {
		t.Errorf("nil.Remove(1) = %v, want false", got)
	}
}

// TestHeap_NilReceiverMutatorsPanic checks that every method that stores,
// clears, resets, or resizes data panics on a nil receiver.
func TestHeap_NilReceiverMutatorsPanic(t *testing.T) {
	tests := []struct {
		name string
		op   string
		call func(*Heap[int])
	}{
		{
			name: "Push",
			op:   "Push",
			call: func(h *Heap[int]) { h.Push(1) },
		},
		{
			name: "PushN",
			op:   "PushN",
			call: func(h *Heap[int]) { h.PushN(1, 2) },
		},
		{
			name: "PushN with no arguments",
			op:   "PushN",
			call: func(h *Heap[int]) { h.PushN() },
		},
		{
			name: "Replace",
			op:   "Replace",
			call: func(h *Heap[int]) { h.Replace(1) },
		},
		{
			name: "Clear",
			op:   "Clear",
			call: func(h *Heap[int]) { h.Clear() },
		},
		{
			name: "Reset",
			op:   "Reset",
			call: func(h *Heap[int]) { h.Reset() },
		},
		{
			name: "Shrink",
			op:   "Shrink",
			call: func(h *Heap[int]) { h.Shrink() },
		},
		{
			name: "Clip",
			op:   "Clip",
			call: func(h *Heap[int]) { h.Clip() },
		},
		{
			name: "Grow",
			op:   "Grow",
			call: func(h *Heap[int]) { h.Grow(8) },
		},
		{
			name: "Grow zero",
			op:   "Grow",
			call: func(h *Heap[int]) { h.Grow(0) },
		},
		{
			name: "Grow negative",
			op:   "Grow",
			call: func(h *Heap[int]) { h.Grow(-1) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var h *Heap[int]
			sharedtest.Panics(t, tt.name, "binheap: "+tt.op+" on nil receiver",
				func() { tt.call(h) })
		})
	}
}

func TestHeap_ZeroValue(t *testing.T) {
	var h Heap[int]

	if got := h.Len(); got != 0 {
		t.Errorf("Len() on zero value = %d, want 0", got)
	}

	if got := h.Cap(); got != 0 {
		t.Errorf("Cap() on zero value = %d, want 0", got)
	}

	if got := h.IsEmpty(); !got {
		t.Errorf("IsEmpty() on zero value = %v, want true", got)
	}

	if got, ok := h.Peek(); ok || got != 0 {
		t.Errorf("Peek() on zero value = (%d, %v), want (0, false)", got, ok)
	}

	if got := h.Contains(1); got {
		t.Errorf("Contains(1) on zero value = %v, want false", got)
	}

	c := h.Clone()
	if c == nil {
		t.Fatalf("Clone() on zero value = nil, want an empty heap")
	}

	if got := c.Len(); got != 0 {
		t.Errorf("Len() of Clone() on zero value = %d, want 0", got)
	}

	if got, ok := h.Pop(); ok || got != 0 {
		t.Errorf("Pop() on zero value = (%d, %v), want (0, false)", got, ok)
	}

	if got := h.PushPop(3); got != 3 {
		t.Errorf("PushPop(3) on zero value = %d, want 3", got)
	}

	if got := h.Remove(1); got {
		t.Errorf("Remove(1) on zero value = %v, want false", got)
	}

	h.Clear()
	h.Reset()
	h.Shrink()
	h.Clip()
	h.Grow(4)

	if got := h.Cap(); got < 4 {
		t.Errorf("Cap() after Grow(4) = %d, want >= 4", got)
	}

	sharedtest.Panics(t, "Push on zero value",
		"binheap: Push on zero-value Heap", func() { h.Push(1) })
	sharedtest.Panics(t, "PushN on zero value",
		"binheap: PushN on zero-value Heap", func() { h.PushN(1) })
	sharedtest.Panics(t, "Replace on zero value",
		"binheap: Replace on zero-value Heap", func() { h.Replace(1) })
}
