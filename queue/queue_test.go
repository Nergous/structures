package queue

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Nergous/structures/internal/shared/sharedtest"
)

// assertNewEmpty fails the test unless New returns an empty queue with no
// preallocated capacity for element type T.
func assertNewEmpty[T any](t *testing.T) {
	t.Helper()

	q := New[T]()

	if got := q.Len(); got != 0 {
		t.Errorf("Len() = %d, want 0", got)
	}

	if got := q.Cap(); got != 0 {
		t.Errorf("Cap() = %d, want 0", got)
	}
}

// assertTailZeroed fails the test unless every slot of the backing array
// past Len holds the zero value of T. The live elements of q must start at
// index 0 of the backing array.
func assertTailZeroed[T comparable](t *testing.T, q *Queue[T]) {
	t.Helper()

	var zero T
	n := q.Len()
	for i, v := range q.buf.Data()[n:] {
		if v != zero {
			t.Errorf("data[%d] = %v, want zero value", n+i, v)
		}
	}
}

// fill returns a queue holding the elements 0..n-1 from front to back.
func fill(n int) *Queue[int] {
	q := New[int]()
	for i := range n {
		q.Push(i)
	}

	return q
}

// wrapped returns a full queue of capacity 4 holding 3, 4, 5, 6 from front to
// back, whose live elements wrap around the end of the backing array.
func wrapped() *Queue[int] {
	q := NewWithCap[int](4)
	q.PushN(1, 2, 3, 4)
	q.Pop()
	q.Pop()
	q.PushN(5, 6)

	return q
}

func TestQueue_New(t *testing.T) {
	t.Run("int", assertNewEmpty[int])
	t.Run("string", assertNewEmpty[string])
	t.Run("float32", assertNewEmpty[float32])
}

func TestQueue_NewWithCap(t *testing.T) {
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
			q := NewWithCap[int](tt.n)

			if got := q.Len(); got != 0 {
				t.Errorf("Len() = %d, want 0", got)
			}

			if got := q.Cap(); got != tt.want {
				t.Errorf("Cap() = %d, want %d", got, tt.want)
			}

			q.Push(1)

			if got, ok := q.Pop(); !ok || got != 1 {
				t.Errorf("Pop() after Push(1) = (%d, %v), want (1, true)",
					got, ok)
			}
		})
	}
}

func TestQueue_Len(t *testing.T) {
	q := New[int]()
	q.PushN(1, 2, 3)

	if got := q.Len(); got != 3 {
		t.Errorf("Len() after PushN(1, 2, 3) = %d, want 3", got)
	}

	q.Pop()

	if got := q.Len(); got != 2 {
		t.Errorf("Len() after Pop() = %d, want 2", got)
	}
}

func TestQueue_Cap(t *testing.T) {
	t.Run("new queue has zero cap", func(t *testing.T) {
		q := New[int]()

		if got := q.Cap(); got != 0 {
			t.Errorf("Cap() = %d, want 0", got)
		}
	})

	t.Run("never less than Len while growing", func(t *testing.T) {
		q := New[int]()
		for i := range 100 {
			q.Push(i)
			if c, l := q.Cap(), q.Len(); c < l {
				t.Fatalf("Cap() #%d = %d, want >= Len() = %d", i, c, l)
			}
		}
	})

	t.Run("no reallocation within preallocated capacity", func(t *testing.T) {
		q := NewWithCap[int](4)
		for i := range 4 {
			q.Push(i)
		}

		if got := q.Cap(); got != 4 {
			t.Errorf("Cap() after 4 Push() = %d, want 4", got)
		}
	})

	t.Run("stays stable when elements are popped", func(t *testing.T) {
		q := NewWithCap[int](4)
		q.PushN(1, 2, 3, 4)
		q.Pop()
		q.Pop()

		if got := q.Cap(); got != 4 {
			t.Errorf("Cap() after Pop(), Pop() = %d, want 4", got)
		}
	})

	t.Run("reuses freed slots without growing", func(t *testing.T) {
		q := NewWithCap[int](4)
		q.PushN(1, 2, 3, 4)
		q.Pop()
		q.Pop()
		q.Push(5)
		q.Push(6)

		if got := q.Cap(); got != 4 {
			t.Errorf("Cap() after refilling freed slots = %d, want 4", got)
		}

		if got, want := q.Slice(), []int{3, 4, 5, 6}; !slices.Equal(got, want) {
			t.Errorf("Slice() after refilling freed slots = %v, want %v",
				got, want)
		}
	})
}

func TestQueue_IsEmpty(t *testing.T) {
	q := New[int]()

	if got := q.IsEmpty(); !got {
		t.Errorf("IsEmpty() on empty = %v, want true", got)
	}

	q.Push(1)

	if got := q.IsEmpty(); got {
		t.Errorf("IsEmpty() after Push(1) = %v, want false", got)
	}

	q.Pop()

	if got := q.IsEmpty(); !got {
		t.Errorf("IsEmpty() after Pop() = %v, want true", got)
	}
}

func TestQueue_Peek(t *testing.T) {
	t.Run("empty returns zero value and false", func(t *testing.T) {
		q := New[int]()

		if got, ok := q.Peek(); ok || got != 0 {
			t.Errorf("Peek() on empty = (%d, %v), want (0, false)", got, ok)
		}
	})

	t.Run("returns front without removing it", func(t *testing.T) {
		q := New[int]()
		q.PushN(1, 2)

		if got, ok := q.Peek(); !ok || got != 1 {
			t.Errorf("Peek() = (%d, %v), want (1, true)", got, ok)
		}

		if got := q.Len(); got != 2 {
			t.Errorf("Len() after Peek() = %d, want 2", got)
		}
	})
}

func TestQueue_All(t *testing.T) {
	t.Run("yields front to back", func(t *testing.T) {
		q := New[int]()
		q.PushN(1, 2, 3)

		got := slices.Collect(q.All())
		if want := []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("All() = %v, want %v", got, want)
		}
	})

	t.Run("yields front to back across wraparound", func(t *testing.T) {
		q := wrapped()

		got := slices.Collect(q.All())
		if want := []int{3, 4, 5, 6}; !slices.Equal(got, want) {
			t.Errorf("All() across wraparound = %v, want %v", got, want)
		}
	})

	t.Run("empty queue yields nothing", func(t *testing.T) {
		q := New[int]()

		if got := slices.Collect(q.All()); len(got) != 0 {
			t.Errorf("All() on empty = %v, want no elements", got)
		}
	})

	t.Run("does not consume", func(t *testing.T) {
		q := New[int]()
		q.PushN(1, 2)

		_ = slices.Collect(q.All())

		if got := q.Len(); got != 2 {
			t.Errorf("Len() after All() = %d, want 2", got)
		}

		if got, ok := q.Pop(); !ok || got != 1 {
			t.Errorf("Pop() after All() = (%d, %v), want (1, true)", got, ok)
		}
	})

	t.Run("break stops iteration early", func(t *testing.T) {
		q := New[int]()
		q.PushN(1, 2, 3)

		var got []int
		for v := range q.All() {
			got = append(got, v)
			if len(got) == 2 {
				break
			}
		}

		if want := []int{1, 2}; !slices.Equal(got, want) {
			t.Errorf("All() with early break = %v, want %v", got, want)
		}

		if got := q.Len(); got != 3 {
			t.Errorf("Len() after break = %d, want 3", got)
		}
	})

	t.Run("break across wraparound", func(t *testing.T) {
		q := wrapped()

		var got []int
		for v := range q.All() {
			got = append(got, v)
			if len(got) == 3 {
				break
			}
		}

		if want := []int{3, 4, 5}; !slices.Equal(got, want) {
			t.Errorf("All() with early break = %v, want %v", got, want)
		}
	})
}

func TestQueue_Clone(t *testing.T) {
	t.Run("copies all elements in order", func(t *testing.T) {
		q := New[int]()
		q.PushN(1, 2, 3)

		c := q.Clone()

		if got, want := c.Slice(), []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("Slice() of clone = %v, want %v", got, want)
		}
	})

	t.Run("copies across wraparound", func(t *testing.T) {
		q := wrapped()

		c := q.Clone()

		want := []int{3, 4, 5, 6}
		if got := c.Slice(); !slices.Equal(got, want) {
			t.Errorf("Slice() of clone across wraparound = %v, want %v",
				got, want)
		}
	})

	t.Run("capacity equals length", func(t *testing.T) {
		q := NewWithCap[int](64)
		q.PushN(1, 2, 3, 4, 5)

		c := q.Clone()

		if got, want := c.Cap(), c.Len(); got != want {
			t.Errorf("Cap() of clone = %d, want Len() = %d", got, want)
		}
	})

	t.Run("is independent of the original", func(t *testing.T) {
		q := New[int]()
		q.PushN(1, 2)

		c := q.Clone()
		c.Push(99)
		q.Pop()

		if got, want := c.Slice(), []int{1, 2, 99}; !slices.Equal(got, want) {
			t.Errorf("Slice() of clone = %v, want %v", got, want)
		}

		if got, want := q.Slice(), []int{2}; !slices.Equal(got, want) {
			t.Errorf("Slice() of original = %v, want %v", got, want)
		}
	})

	t.Run("clone of empty queue is empty and usable", func(t *testing.T) {
		q := New[int]()

		c := q.Clone()

		if got := c.Len(); got != 0 {
			t.Errorf("Len() of clone = %d, want 0", got)
		}

		c.Push(1)

		if got, ok := c.Pop(); !ok || got != 1 {
			t.Errorf("Pop() of clone after Push(1) = (%d, %v), "+
				"want (1, true)", got, ok)
		}
	})
}

func TestQueue_Slice(t *testing.T) {
	t.Run("returns elements front to back", func(t *testing.T) {
		q := New[int]()
		q.PushN(1, 2, 3)

		if got, want := q.Slice(), []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("Slice() = %v, want %v", got, want)
		}
	})

	t.Run("returns elements across wraparound", func(t *testing.T) {
		q := wrapped()

		if got, want := q.Slice(), []int{3, 4, 5, 6}; !slices.Equal(got, want) {
			t.Errorf("Slice() across wraparound = %v, want %v", got, want)
		}
	})

	t.Run("returns an independent copy", func(t *testing.T) {
		q := New[int]()
		q.PushN(1, 2)

		out := q.Slice()
		for i := range out {
			out[i] = 999
		}

		if got, want := q.Slice(), []int{1, 2}; !slices.Equal(got, want) {
			t.Errorf("Slice() after mutating a copy = %v, want %v", got, want)
		}
	})

	t.Run("does not consume", func(t *testing.T) {
		q := New[int]()
		q.PushN(1, 2)

		_ = q.Slice()

		if got := q.Len(); got != 2 {
			t.Errorf("Len() after Slice() = %d, want 2", got)
		}
	})

	t.Run("empty queue returns nil", func(t *testing.T) {
		drained := NewWithCap[int](8)
		drained.Push(1)
		drained.Pop()

		reset := NewWithCap[int](8)
		reset.Push(1)
		reset.Reset()

		cleared := NewWithCap[int](8)
		cleared.Push(1)
		cleared.Clear()

		tests := []struct {
			name string
			q    *Queue[int]
		}{
			{
				name: "fresh New",
				q:    New[int](),
			},
			{
				name: "preallocated",
				q:    NewWithCap[int](8),
			},
			{
				name: "drained to zero",
				q:    drained,
			},
			{
				name: "after Reset",
				q:    reset,
			},
			{
				name: "after Clear",
				q:    cleared,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.q.Slice(); got != nil {
					t.Errorf("Slice() = %v, want nil", got)
				}
			})
		}
	})
}

func TestQueue_Push(t *testing.T) {
	t.Run("appends to the back", func(t *testing.T) {
		q := New[int]()
		q.Push(1)
		q.Push(2)
		q.Push(3)

		if got, want := q.Slice(), []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("Slice() after Push() = %v, want %v", got, want)
		}
	})

	t.Run("grows past preallocated capacity", func(t *testing.T) {
		q := NewWithCap[int](2)
		q.PushN(1, 2)

		q.Push(3)

		if got := q.Cap(); got < 3 {
			t.Errorf("Cap() after Push(3) = %d, want >= 3", got)
		}

		if got, want := q.Slice(), []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("Slice() after Push(3) = %v, want %v", got, want)
		}
	})
}

func TestQueue_PushN(t *testing.T) {
	t.Run("pushes in argument order, last at the back", func(t *testing.T) {
		q := New[int]()
		q.PushN(1, 2, 3)

		if got, want := q.Slice(), []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("Slice() after PushN(1, 2, 3) = %v, want %v", got, want)
		}
	})

	t.Run("no arguments is a no-op", func(t *testing.T) {
		q := New[int]()
		q.PushN()

		if got := q.Len(); got != 0 {
			t.Errorf("Len() after PushN() = %d, want 0", got)
		}
	})

	t.Run("appends onto existing elements", func(t *testing.T) {
		q := New[int]()
		q.Push(1)
		q.PushN(2, 3)

		if got, want := q.Slice(), []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("Slice() after PushN(2, 3) = %v, want %v", got, want)
		}
	})

	t.Run("grows the backing buffer at most once", func(t *testing.T) {
		q := New[int]()
		q.PushN(1, 2, 3, 4, 5, 6, 7, 8)

		if got := q.Cap(); got < 8 {
			t.Errorf("Cap() after PushN(1, ..., 8) = %d, want >= 8", got)
		}
	})
}

func TestQueue_Pop(t *testing.T) {
	tests := []struct {
		name string
		push []int
		want []int
	}{
		{
			name: "empty",
			push: nil,
			want: nil,
		},
		{
			name: "single element",
			push: []int{42},
			want: []int{42},
		},
		{
			name: "FIFO order",
			push: []int{1, 2, 3},
			want: []int{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := New[int]()
			q.PushN(tt.push...)

			for i, want := range tt.want {
				if got, ok := q.Pop(); !ok || got != want {
					t.Errorf("Pop() #%d = (%d, %v), want (%d, true)",
						i, got, ok, want)
				}
			}

			if got, ok := q.Pop(); ok || got != 0 {
				t.Errorf("Pop() on empty = (%d, %v), want (0, false)",
					got, ok)
			}
		})
	}

	t.Run("zeroes vacated slot", func(t *testing.T) {
		q := New[*int]()
		v := 1
		q.Push(&v)
		data := q.buf.Data()

		if _, ok := q.Pop(); !ok {
			t.Fatalf("Pop() ok = false, want true")
		}

		if got := data[0]; got != nil {
			t.Errorf("data[0] after Pop() = %p, "+
				"want nil (Pop must zero the slot)", got)
		}
	})
}

// TestQueue_PushPopInterleaved checks that FIFO order holds when pushes and
// pops are interleaved.
func TestQueue_PushPopInterleaved(t *testing.T) {
	q := New[int]()
	q.PushN(1, 2)

	if got, ok := q.Pop(); !ok || got != 1 {
		t.Fatalf("Pop() = (%d, %v), want (1, true)", got, ok)
	}

	q.Push(3)

	for i, want := range []int{2, 3} {
		if got, ok := q.Pop(); !ok || got != want {
			t.Errorf("Pop() #%d = (%d, %v), want (%d, true)",
				i, got, ok, want)
		}
	}

	if got, ok := q.Pop(); ok || got != 0 {
		t.Errorf("Pop() on empty = (%d, %v), want (0, false)", got, ok)
	}
}

func TestQueue_PopN(t *testing.T) {
	tests := []struct {
		name string
		push []int
		n    int
		want []int
		left []int
	}{
		{
			name: "one",
			push: []int{1, 2, 3, 4, 5},
			n:    1,
			want: []int{1},
			left: []int{2, 3, 4, 5},
		},
		{
			name: "several in FIFO order",
			push: []int{1, 2, 3, 4, 5},
			n:    3,
			want: []int{1, 2, 3},
			left: []int{4, 5},
		},
		{
			name: "exactly Len",
			push: []int{1, 2, 3, 4, 5},
			n:    5,
			want: []int{1, 2, 3, 4, 5},
			left: nil,
		},
		{
			name: "more than Len drains",
			push: []int{1, 2, 3, 4, 5},
			n:    99,
			want: []int{1, 2, 3, 4, 5},
			left: nil,
		},
		{
			name: "zero",
			push: []int{1, 2, 3, 4, 5},
			n:    0,
			want: nil,
			left: []int{1, 2, 3, 4, 5},
		},
		{
			name: "negative",
			push: []int{1, 2, 3, 4, 5},
			n:    -3,
			want: nil,
			left: []int{1, 2, 3, 4, 5},
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
			q := New[int]()
			q.PushN(tt.push...)
			capBefore := q.Cap()

			got := q.PopN(tt.n)

			if !slices.Equal(got, tt.want) || (got == nil) != (tt.want == nil) {
				t.Errorf("PopN(%d) = %v, want %v", tt.n, got, tt.want)
			}

			if got := q.Slice(); !slices.Equal(got, tt.left) {
				t.Errorf("Slice() after PopN(%d) = %v, want %v",
					tt.n, got, tt.left)
			}

			if got := q.Cap(); got != capBefore {
				t.Errorf("Cap() after PopN(%d) = %d, want %d",
					tt.n, got, capBefore)
			}
		})
	}

	t.Run("works across wraparound", func(t *testing.T) {
		q := wrapped()

		if got, want := q.PopN(3), []int{3, 4, 5}; !slices.Equal(got, want) {
			t.Errorf("PopN(3) across wraparound = %v, want %v", got, want)
		}

		if got, want := q.Slice(), []int{6}; !slices.Equal(got, want) {
			t.Errorf("Slice() after PopN(3) = %v, want %v", got, want)
		}
	})

	t.Run("zeroes vacated slots", func(t *testing.T) {
		q := NewWithCap[*int](4)
		a, b, c := 1, 2, 3
		q.PushN(&a, &b, &c)

		q.PopN(3)

		assertTailZeroed(t, q)
	})
}

func TestQueue_Clear(t *testing.T) {
	q := NewWithCap[int](8)
	q.PushN(0, 1, 2, 3, 4)

	q.Clear()

	if got := q.Len(); got != 0 {
		t.Errorf("Len() after Clear() = %d, want 0", got)
	}

	if got := q.Cap(); got != 0 {
		t.Errorf("Cap() after Clear() = %d, want 0", got)
	}

	if got, ok := q.Pop(); ok || got != 0 {
		t.Errorf("Pop() after Clear() = (%d, %v), want (0, false)", got, ok)
	}

	q.Push(99)

	if got, ok := q.Pop(); !ok || got != 99 {
		t.Errorf("Pop() after Clear() and Push(99) = (%d, %v), "+
			"want (99, true)", got, ok)
	}
}

func TestQueue_Reset(t *testing.T) {
	t.Run("Reset keeps capacity", func(t *testing.T) {
		q := fill(10)
		capBefore := q.Cap()

		q.Reset()

		if got := q.Len(); got != 0 {
			t.Errorf("Len() after Reset() = %d, want 0", got)
		}

		if got := q.Cap(); got != capBefore {
			t.Errorf("Cap() after Reset() = %d, want %d", got, capBefore)
		}
	})

	t.Run("keeps full capacity after pops", func(t *testing.T) {
		q := NewWithCap[int](8)
		q.PushN(1, 2, 3, 4, 5)
		q.Pop()
		q.Pop()

		q.Reset()

		if got := q.Cap(); got != 8 {
			t.Errorf("Cap() after Pop(), Pop(), Reset() = %d, want 8", got)
		}
	})

	t.Run("zeroes backing array", func(t *testing.T) {
		q := NewWithCap[*int](4)
		a, b, c, e := 1, 2, 3, 4
		q.PushN(&a, &b, &c)
		q.Pop()
		q.PushN(&e, &a)

		q.Reset()

		assertTailZeroed(t, q)
	})

	t.Run("usable after Reset", func(t *testing.T) {
		q := fill(3)
		q.Reset()

		q.Push(42)

		if got, ok := q.Pop(); !ok || got != 42 {
			t.Errorf("Pop() after Reset() and Push(42) = (%d, %v), "+
				"want (42, true)", got, ok)
		}

		if got, ok := q.Pop(); ok || got != 0 {
			t.Errorf("Pop() on empty = (%d, %v), want (0, false)", got, ok)
		}
	})
}

func TestQueue_Shrink(t *testing.T) {
	t.Run("reduces capacity to length and keeps contents", func(t *testing.T) {
		q := NewWithCap[int](64)
		q.PushN(1, 2, 3)

		q.Shrink()

		if got, want := q.Cap(), q.Len(); got != want {
			t.Errorf("Cap() after Shrink() = %d, want Len() = %d", got, want)
		}

		if got, want := q.Slice(), []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("Slice() after Shrink() = %v, want %v", got, want)
		}
	})

	t.Run("allocates a new backing array", func(t *testing.T) {
		q := NewWithCap[int](64)
		q.PushN(1, 2, 3)
		old := &q.buf.Data()[0]

		q.Shrink()

		if got := &q.buf.Data()[0]; got == old {
			t.Errorf("&data[0] after Shrink() = %p, want a new backing array",
				got)
		}
	})

	t.Run("compacts wrapped elements", func(t *testing.T) {
		q := wrapped()
		q.Pop()

		q.Shrink()

		if got, want := q.Cap(), q.Len(); got != want {
			t.Errorf("Cap() after Shrink() = %d, want Len() = %d", got, want)
		}

		if got, want := q.Slice(), []int{4, 5, 6}; !slices.Equal(got, want) {
			t.Errorf("Slice() after Shrink() = %v, want %v", got, want)
		}
	})

	t.Run("empty queue drops to zero capacity", func(t *testing.T) {
		q := NewWithCap[int](8)

		q.Shrink()

		if got := q.Cap(); got != 0 {
			t.Errorf("Cap() after Shrink() on empty = %d, want 0", got)
		}
	})

	t.Run("usable after Shrink", func(t *testing.T) {
		q := NewWithCap[int](16)
		q.Push(1)
		q.Shrink()

		q.Push(2)

		if got, want := q.Slice(), []int{1, 2}; !slices.Equal(got, want) {
			t.Errorf("Slice() after Shrink() and Push(2) = %v, want %v",
				got, want)
		}
	})
}

func TestQueue_Clip(t *testing.T) {
	t.Run("reduces capacity to length and keeps contents", func(t *testing.T) {
		q := NewWithCap[int](64)
		q.PushN(1, 2, 3)

		q.Clip()

		if got, want := q.Cap(), q.Len(); got != want {
			t.Errorf("Cap() after Clip() = %d, want Len() = %d", got, want)
		}

		if got, want := q.Slice(), []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("Slice() after Clip() = %v, want %v", got, want)
		}
	})

	t.Run("does not reallocate", func(t *testing.T) {
		q := NewWithCap[int](64)
		q.PushN(1, 2, 3)
		old := &q.buf.Data()[0]

		q.Clip()

		if got := &q.buf.Data()[0]; got != old {
			t.Errorf("&data[0] after Clip() = %p, want %p (no copy)", got, old)
		}
	})

	t.Run("compacts wrapped elements", func(t *testing.T) {
		q := wrapped()
		q.Pop()

		q.Clip()

		if got, want := q.Cap(), q.Len(); got != want {
			t.Errorf("Cap() after Clip() = %d, want Len() = %d", got, want)
		}

		if got, want := q.Slice(), []int{4, 5, 6}; !slices.Equal(got, want) {
			t.Errorf("Slice() after Clip() = %v, want %v", got, want)
		}
	})

	t.Run("empty queue drops to zero capacity", func(t *testing.T) {
		q := NewWithCap[int](8)

		q.Clip()

		if got := q.Cap(); got != 0 {
			t.Errorf("Cap() after Clip() on empty = %d, want 0", got)
		}
	})

	t.Run("usable after Clip", func(t *testing.T) {
		q := NewWithCap[int](16)
		q.Push(1)
		q.Clip()

		q.Push(2)

		if got, want := q.Slice(), []int{1, 2}; !slices.Equal(got, want) {
			t.Errorf("Slice() after Clip() and Push(2) = %v, want %v",
				got, want)
		}
	})
}

func TestQueue_Grow(t *testing.T) {
	t.Run("reserves capacity without changing contents", func(t *testing.T) {
		q := wrapped()

		q.Grow(100)

		if got := q.Cap(); got < 104 {
			t.Errorf("Cap() after Grow(100) = %d, want >= 104", got)
		}

		if got, want := q.Slice(), []int{3, 4, 5, 6}; !slices.Equal(got, want) {
			t.Errorf("Slice() after Grow(100) = %v, want %v", got, want)
		}
	})

	t.Run("no reallocation within grown capacity", func(t *testing.T) {
		q := New[int]()
		q.Grow(8)
		capBefore := q.Cap()

		for i := range 8 {
			q.Push(i)
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

// TestQueue_SustainedChurn interleaves Push and Pop at a fixed live size over
// many iterations: FIFO order must hold throughout and the capacity must stay
// bounded near the live size instead of growing with the iteration count.
func TestQueue_SustainedChurn(t *testing.T) {
	const (
		live       = 16
		iterations = 100000
	)

	q := fill(live)
	next := live
	maxCap := 0
	for i := range iterations {
		if got, ok := q.Pop(); !ok || got != next-live {
			t.Fatalf("Pop() #%d = (%d, %v), want (%d, true)",
				i, got, ok, next-live)
		}

		q.Push(next)
		next++
		maxCap = max(maxCap, q.Cap())

		if got := q.Len(); got != live {
			t.Fatalf("Len() #%d = %d, want %d", i, got, live)
		}
	}

	if maxCap > live*64 {
		t.Errorf("max Cap() during churn = %d, want <= %d", maxCap, live*64)
	}
}

// TestQueue_RandomOps runs a seeded random sequence of operations against a
// slice model and checks that both agree after every step.
func TestQueue_RandomOps(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	q := New[int]()
	var model []int // front to back

	for i := range 5000 {
		switch op := r.IntN(100); {
		case op < 40:
			q.Push(i)
			model = append(model, i)
		case op < 50:
			q.PushN(i, i+1, i+2)
			model = append(model, i, i+1, i+2)
		case op < 80:
			want, wantOK := 0, false
			if len(model) > 0 {
				want, wantOK = model[0], true
				model = model[1:]
			}

			if got, ok := q.Pop(); ok != wantOK || got != want {
				t.Fatalf("step %d: Pop() = (%d, %v), want (%d, %v)",
					i, got, ok, want, wantOK)
			}
		case op < 88:
			n := r.IntN(5) - 1
			k := min(max(n, 0), len(model))
			want := slices.Clone(model[:k])
			model = model[k:]

			if got := q.PopN(n); !slices.Equal(got, want) {
				t.Fatalf("step %d: PopN(%d) = %v, want %v", i, n, got, want)
			}
		case op < 94:
			want, wantOK := 0, false
			if len(model) > 0 {
				want, wantOK = model[0], true
			}

			if got, ok := q.Peek(); ok != wantOK || got != want {
				t.Fatalf("step %d: Peek() = (%d, %v), want (%d, %v)",
					i, got, ok, want, wantOK)
			}
		default:
			switch r.IntN(5) {
			case 0:
				q.Clear()
				model = nil
			case 1:
				q.Reset()
				model = nil
			case 2:
				q.Shrink()
			case 3:
				q.Clip()
			default:
				q.Grow(r.IntN(16))
			}
		}

		if got := q.Slice(); !slices.Equal(got, model) {
			t.Fatalf("step %d: Slice() = %v, want %v", i, got, model)
		}

		if got := q.Len(); got != len(model) {
			t.Fatalf("step %d: Len() = %d, want %d", i, got, len(model))
		}
	}
}

// TestQueue_NilReceiver checks that every read-only and removal method treats
// a nil receiver as an empty queue.
func TestQueue_NilReceiver(t *testing.T) {
	var q *Queue[int]

	if got := q.Len(); got != 0 {
		t.Errorf("nil.Len() = %d, want 0", got)
	}

	if got := q.Cap(); got != 0 {
		t.Errorf("nil.Cap() = %d, want 0", got)
	}

	if got := q.IsEmpty(); !got {
		t.Errorf("nil.IsEmpty() = %v, want true", got)
	}

	if got, ok := q.Peek(); ok || got != 0 {
		t.Errorf("nil.Peek() = (%d, %v), want (0, false)", got, ok)
	}

	count := 0
	for range q.All() {
		count++
	}

	if count != 0 {
		t.Errorf("nil.All() count = %d, want 0", count)
	}

	c := q.Clone()
	if c == nil {
		t.Fatalf("nil.Clone() = nil, want a usable empty queue")
	}

	if got := c.Len(); got != 0 {
		t.Errorf("nil.Clone().Len() = %d, want 0", got)
	}

	c.Push(42)

	if got, ok := c.Pop(); !ok || got != 42 {
		t.Errorf("Pop() after Push(42) on nil.Clone() = (%d, %v), "+
			"want (42, true)", got, ok)
	}

	if got := q.Slice(); got != nil {
		t.Errorf("nil.Slice() = %v, want nil", got)
	}

	if got, ok := q.Pop(); ok || got != 0 {
		t.Errorf("nil.Pop() = (%d, %v), want (0, false)", got, ok)
	}

	if got := q.PopN(3); got != nil {
		t.Errorf("nil.PopN(3) = %v, want nil", got)
	}
}

// TestQueue_NilReceiverMutatorsPanic checks that every method that stores,
// clears, resets, or resizes data panics on a nil receiver.
func TestQueue_NilReceiverMutatorsPanic(t *testing.T) {
	tests := []struct {
		name string
		op   string
		call func(*Queue[int])
	}{
		{
			name: "Push",
			op:   "Push",
			call: func(q *Queue[int]) { q.Push(1) },
		},
		{
			name: "PushN",
			op:   "PushN",
			call: func(q *Queue[int]) { q.PushN(1, 2) },
		},
		{
			name: "PushN with no arguments",
			op:   "PushN",
			call: func(q *Queue[int]) { q.PushN() },
		},
		{
			name: "Clear",
			op:   "Clear",
			call: func(q *Queue[int]) { q.Clear() },
		},
		{
			name: "Reset",
			op:   "Reset",
			call: func(q *Queue[int]) { q.Reset() },
		},
		{
			name: "Shrink",
			op:   "Shrink",
			call: func(q *Queue[int]) { q.Shrink() },
		},
		{
			name: "Clip",
			op:   "Clip",
			call: func(q *Queue[int]) { q.Clip() },
		},
		{
			name: "Grow",
			op:   "Grow",
			call: func(q *Queue[int]) { q.Grow(8) },
		},
		{
			name: "Grow zero",
			op:   "Grow",
			call: func(q *Queue[int]) { q.Grow(0) },
		},
		{
			name: "Grow negative",
			op:   "Grow",
			call: func(q *Queue[int]) { q.Grow(-1) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var q *Queue[int]
			sharedtest.Panics(t, tt.name, "queue: "+tt.op+" on nil receiver",
				func() { tt.call(q) })
		})
	}
}

func TestQueue_ZeroValue(t *testing.T) {
	var q Queue[int]

	if got := q.Len(); got != 0 {
		t.Errorf("Len() on zero value = %d, want 0", got)
	}

	if got := q.Cap(); got != 0 {
		t.Errorf("Cap() on zero value = %d, want 0", got)
	}

	if got := q.IsEmpty(); !got {
		t.Errorf("IsEmpty() on zero value = %v, want true", got)
	}

	if got, ok := q.Peek(); ok || got != 0 {
		t.Errorf("Peek() on zero value = (%d, %v), want (0, false)", got, ok)
	}

	if got, ok := q.Pop(); ok || got != 0 {
		t.Errorf("Pop() on zero value = (%d, %v), want (0, false)", got, ok)
	}

	q.Clear()
	q.Reset()
	q.Shrink()
	q.Clip()
	q.Grow(4)

	if got := q.Cap(); got < 4 {
		t.Errorf("Cap() after Grow(4) = %d, want >= 4", got)
	}

	q.PushN(1, 2)

	if got, ok := q.Pop(); !ok || got != 1 {
		t.Errorf("Pop() after PushN(1, 2) on zero value = (%d, %v), "+
			"want (1, true)", got, ok)
	}
}
