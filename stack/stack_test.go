package stack

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Nergous/structures/internal/shared/sharedtest"
)

// assertNewEmpty fails the test unless New returns an empty stack with no
// preallocated capacity for element type T.
func assertNewEmpty[T any](t *testing.T) {
	t.Helper()

	s := New[T]()

	if got := s.Len(); got != 0 {
		t.Errorf("Len() = %d, want 0", got)
	}

	if got := s.Cap(); got != 0 {
		t.Errorf("Cap() = %d, want 0", got)
	}
}

// assertTailZeroed fails the test unless every slot of the backing array
// between Len and Cap holds the zero value of T.
func assertTailZeroed[T comparable](t *testing.T, s *Stack[T]) {
	t.Helper()

	var zero T
	tail := s.data[len(s.data):cap(s.data)]
	for i, v := range tail {
		if v != zero {
			t.Errorf("data[%d] = %v, want zero value", len(s.data)+i, v)
		}
	}
}

// fill returns a stack holding the elements 0..n-1 in push order, so n-1 is
// on top.
func fill(n int) *Stack[int] {
	s := New[int]()
	for i := range n {
		s.Push(i)
	}

	return s
}

func TestStack_New(t *testing.T) {
	t.Run("int", assertNewEmpty[int])
	t.Run("string", assertNewEmpty[string])
	t.Run("float32", assertNewEmpty[float32])
}

func TestStack_NewWithCap(t *testing.T) {
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
			s := NewWithCap[int](tt.n)

			if got := s.Len(); got != 0 {
				t.Errorf("Len() = %d, want 0", got)
			}

			if got := s.Cap(); got != tt.want {
				t.Errorf("Cap() = %d, want %d", got, tt.want)
			}

			s.Push(1)

			if got, ok := s.Pop(); !ok || got != 1 {
				t.Errorf("Pop() after Push(1) = (%d, %v), want (1, true)",
					got, ok)
			}
		})
	}
}

func TestStack_Len(t *testing.T) {
	s := New[int]()
	s.PushN(1, 2, 3)

	if got := s.Len(); got != 3 {
		t.Errorf("Len() after PushN(1, 2, 3) = %d, want 3", got)
	}

	s.Pop()

	if got := s.Len(); got != 2 {
		t.Errorf("Len() after Pop() = %d, want 2", got)
	}
}

func TestStack_Cap(t *testing.T) {
	t.Run("new stack has zero cap", func(t *testing.T) {
		s := New[int]()

		if got := s.Cap(); got != 0 {
			t.Errorf("Cap() = %d, want 0", got)
		}
	})

	t.Run("never less than Len while growing", func(t *testing.T) {
		s := New[int]()
		for i := range 100 {
			s.Push(i)
			if c, l := s.Cap(), s.Len(); c < l {
				t.Fatalf("Cap() #%d = %d, want >= Len() = %d", i, c, l)
			}
		}
	})

	t.Run("no reallocation within preallocated capacity", func(t *testing.T) {
		s := NewWithCap[int](4)
		for i := range 4 {
			s.Push(i)
		}

		if got := s.Cap(); got != 4 {
			t.Errorf("Cap() after 4 Push() = %d, want 4", got)
		}
	})
}

func TestStack_IsEmpty(t *testing.T) {
	s := New[int]()

	if got := s.IsEmpty(); !got {
		t.Errorf("IsEmpty() on empty = %v, want true", got)
	}

	s.Push(1)

	if got := s.IsEmpty(); got {
		t.Errorf("IsEmpty() after Push(1) = %v, want false", got)
	}

	s.Pop()

	if got := s.IsEmpty(); !got {
		t.Errorf("IsEmpty() after Pop() = %v, want true", got)
	}
}

func TestStack_Peek(t *testing.T) {
	t.Run("empty returns zero value and false", func(t *testing.T) {
		s := New[int]()

		if got, ok := s.Peek(); ok || got != 0 {
			t.Errorf("Peek() on empty = (%d, %v), want (0, false)", got, ok)
		}
	})

	t.Run("returns top without removing it", func(t *testing.T) {
		s := New[int]()
		s.PushN(1, 2)

		if got, ok := s.Peek(); !ok || got != 2 {
			t.Errorf("Peek() = (%d, %v), want (2, true)", got, ok)
		}

		if got := s.Len(); got != 2 {
			t.Errorf("Len() after Peek() = %d, want 2", got)
		}
	})
}

func TestStack_All(t *testing.T) {
	t.Run("yields top to bottom", func(t *testing.T) {
		s := New[int]()
		s.PushN(1, 2, 3)

		got := slices.Collect(s.All())
		if want := []int{3, 2, 1}; !slices.Equal(got, want) {
			t.Errorf("All() = %v, want %v", got, want)
		}
	})

	t.Run("empty stack yields nothing", func(t *testing.T) {
		s := New[int]()

		if got := slices.Collect(s.All()); len(got) != 0 {
			t.Errorf("All() on empty = %v, want no elements", got)
		}
	})

	t.Run("does not consume", func(t *testing.T) {
		s := New[int]()
		s.PushN(1, 2)

		_ = slices.Collect(s.All())

		if got := s.Len(); got != 2 {
			t.Errorf("Len() after All() = %d, want 2", got)
		}

		if got, ok := s.Pop(); !ok || got != 2 {
			t.Errorf("Pop() after All() = (%d, %v), want (2, true)", got, ok)
		}
	})

	t.Run("break stops iteration early", func(t *testing.T) {
		s := New[int]()
		s.PushN(1, 2, 3)

		var got []int
		for v := range s.All() {
			got = append(got, v)
			if len(got) == 2 {
				break
			}
		}

		if want := []int{3, 2}; !slices.Equal(got, want) {
			t.Errorf("All() with early break = %v, want %v", got, want)
		}

		if got := s.Len(); got != 3 {
			t.Errorf("Len() after break = %d, want 3", got)
		}
	})
}

func TestStack_Clone(t *testing.T) {
	t.Run("copies all elements in order", func(t *testing.T) {
		s := New[int]()
		s.PushN(1, 2, 3)

		c := s.Clone()

		if got, want := c.Slice(), []int{3, 2, 1}; !slices.Equal(got, want) {
			t.Errorf("Slice() of clone = %v, want %v", got, want)
		}
	})

	t.Run("capacity equals length", func(t *testing.T) {
		// Five ints do not fill an allocation size class, so a clone built
		// with append would report spare capacity.
		s := NewWithCap[int](64)
		s.PushN(1, 2, 3, 4, 5)

		c := s.Clone()

		if got, want := c.Cap(), c.Len(); got != want {
			t.Errorf("Cap() of clone = %d, want Len() = %d", got, want)
		}
	})

	t.Run("is independent of the original", func(t *testing.T) {
		s := New[int]()
		s.PushN(1, 2)

		c := s.Clone()
		c.Push(99)
		s.Pop()

		if got, want := c.Slice(), []int{99, 2, 1}; !slices.Equal(got, want) {
			t.Errorf("Slice() of clone = %v, want %v", got, want)
		}

		if got, want := s.Slice(), []int{1}; !slices.Equal(got, want) {
			t.Errorf("Slice() of original = %v, want %v", got, want)
		}
	})

	t.Run("clone of empty stack is empty and usable", func(t *testing.T) {
		s := New[int]()

		c := s.Clone()

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

func TestStack_Slice(t *testing.T) {
	t.Run("returns elements top to bottom", func(t *testing.T) {
		s := New[int]()
		s.PushN(1, 2, 3)

		if got, want := s.Slice(), []int{3, 2, 1}; !slices.Equal(got, want) {
			t.Errorf("Slice() = %v, want %v", got, want)
		}
	})

	t.Run("returns an independent copy", func(t *testing.T) {
		s := New[int]()
		s.PushN(1, 2)

		out := s.Slice()
		for i := range out {
			out[i] = 999
		}

		if got, want := s.Slice(), []int{2, 1}; !slices.Equal(got, want) {
			t.Errorf("Slice() after mutating a copy = %v, want %v", got, want)
		}
	})

	t.Run("does not consume", func(t *testing.T) {
		s := New[int]()
		s.PushN(1, 2)

		_ = s.Slice()

		if got := s.Len(); got != 2 {
			t.Errorf("Len() after Slice() = %d, want 2", got)
		}
	})

	t.Run("empty stack returns nil", func(t *testing.T) {
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
			s    *Stack[int]
		}{
			{
				name: "fresh New",
				s:    New[int](),
			},
			{
				name: "preallocated",
				s:    NewWithCap[int](8),
			},
			{
				name: "drained to zero",
				s:    drained,
			},
			{
				name: "after Reset",
				s:    reset,
			},
			{
				name: "after Clear",
				s:    cleared,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.s.Slice(); got != nil {
					t.Errorf("Slice() = %v, want nil", got)
				}
			})
		}
	})
}

func TestStack_Push(t *testing.T) {
	t.Run("adds to the top", func(t *testing.T) {
		s := New[int]()
		s.Push(1)
		s.Push(2)

		if got, ok := s.Peek(); !ok || got != 2 {
			t.Errorf("Peek() after Push(2) = (%d, %v), want (2, true)",
				got, ok)
		}

		if got := s.Len(); got != 2 {
			t.Errorf("Len() after two Push() = %d, want 2", got)
		}
	})
}

func TestStack_PushN(t *testing.T) {
	t.Run("pushes in argument order, last on top", func(t *testing.T) {
		s := New[int]()
		s.PushN(1, 2, 3)

		if got, want := s.Slice(), []int{3, 2, 1}; !slices.Equal(got, want) {
			t.Errorf("Slice() after PushN(1, 2, 3) = %v, want %v", got, want)
		}
	})

	t.Run("no arguments is a no-op", func(t *testing.T) {
		s := New[int]()
		s.PushN()

		if got := s.Len(); got != 0 {
			t.Errorf("Len() after PushN() = %d, want 0", got)
		}
	})

	t.Run("appends onto existing elements", func(t *testing.T) {
		s := New[int]()
		s.Push(1)
		s.PushN(2, 3)

		if got, want := s.Slice(), []int{3, 2, 1}; !slices.Equal(got, want) {
			t.Errorf("Slice() after PushN(2, 3) = %v, want %v", got, want)
		}
	})
}

func TestStack_Pop(t *testing.T) {
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
			name: "LIFO order",
			push: []int{1, 2, 3},
			want: []int{3, 2, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New[int]()
			s.PushN(tt.push...)

			for i, want := range tt.want {
				if got, ok := s.Pop(); !ok || got != want {
					t.Errorf("Pop() #%d = (%d, %v), want (%d, true)",
						i, got, ok, want)
				}
			}

			if got, ok := s.Pop(); ok || got != 0 {
				t.Errorf("Pop() on empty = (%d, %v), want (0, false)",
					got, ok)
			}
		})
	}

	t.Run("zeroes vacated slot", func(t *testing.T) {
		s := New[*int]()
		v := 1
		s.Push(&v)
		data := s.data[:1]

		if _, ok := s.Pop(); !ok {
			t.Fatalf("Pop() ok = false, want true")
		}

		if got := data[0]; got != nil {
			t.Errorf("data[0] after Pop() = %p, "+
				"want nil (Pop must zero the slot)", got)
		}
	})
}

func TestStack_PopN(t *testing.T) {
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
			want: []int{5},
			left: []int{4, 3, 2, 1},
		},
		{
			name: "several in LIFO order",
			push: []int{1, 2, 3, 4, 5},
			n:    3,
			want: []int{5, 4, 3},
			left: []int{2, 1},
		},
		{
			name: "exactly Len",
			push: []int{1, 2, 3, 4, 5},
			n:    5,
			want: []int{5, 4, 3, 2, 1},
			left: nil,
		},
		{
			name: "more than Len drains",
			push: []int{1, 2, 3, 4, 5},
			n:    99,
			want: []int{5, 4, 3, 2, 1},
			left: nil,
		},
		{
			name: "zero",
			push: []int{1, 2, 3, 4, 5},
			n:    0,
			want: nil,
			left: []int{5, 4, 3, 2, 1},
		},
		{
			name: "negative",
			push: []int{1, 2, 3, 4, 5},
			n:    -3,
			want: nil,
			left: []int{5, 4, 3, 2, 1},
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
			s := New[int]()
			s.PushN(tt.push...)
			capBefore := s.Cap()

			got := s.PopN(tt.n)

			if !slices.Equal(got, tt.want) || (got == nil) != (tt.want == nil) {
				t.Errorf("PopN(%d) = %v, want %v", tt.n, got, tt.want)
			}

			if got := s.Slice(); !slices.Equal(got, tt.left) {
				t.Errorf("Slice() after PopN(%d) = %v, want %v",
					tt.n, got, tt.left)
			}

			if got := s.Cap(); got != capBefore {
				t.Errorf("Cap() after PopN(%d) = %d, want %d",
					tt.n, got, capBefore)
			}
		})
	}

	t.Run("zeroes vacated slots", func(t *testing.T) {
		s := New[*int]()
		a, b, c := 1, 2, 3
		s.PushN(&a, &b, &c)

		s.PopN(2)

		assertTailZeroed(t, s)

		if got := s.data[0]; got != &a {
			t.Errorf("data[0] after PopN(2) = %p, want %p", got, &a)
		}
	})

	t.Run("usable after PopN", func(t *testing.T) {
		s := New[int]()
		s.PushN(1, 2, 3)
		s.PopN(2)

		s.Push(9)

		if got, want := s.Slice(), []int{9, 1}; !slices.Equal(got, want) {
			t.Errorf("Slice() after PopN(2) and Push(9) = %v, want %v",
				got, want)
		}
	})
}

func TestStack_Clear(t *testing.T) {
	s := NewWithCap[int](8)
	s.PushN(0, 1, 2, 3, 4)

	s.Clear()

	if got := s.Len(); got != 0 {
		t.Errorf("Len() after Clear() = %d, want 0", got)
	}

	if got := s.Cap(); got != 0 {
		t.Errorf("Cap() after Clear() = %d, want 0", got)
	}

	if got, ok := s.Pop(); ok || got != 0 {
		t.Errorf("Pop() after Clear() = (%d, %v), want (0, false)", got, ok)
	}

	s.Push(99)

	if got, ok := s.Pop(); !ok || got != 99 {
		t.Errorf("Pop() after Clear() and Push(99) = (%d, %v), "+
			"want (99, true)", got, ok)
	}
}

func TestStack_Reset(t *testing.T) {
	t.Run("Reset keeps capacity", func(t *testing.T) {
		s := fill(10)
		capBefore := s.Cap()

		s.Reset()

		if got := s.Len(); got != 0 {
			t.Errorf("Len() after Reset() = %d, want 0", got)
		}

		if got := s.Cap(); got != capBefore {
			t.Errorf("Cap() after Reset() = %d, want %d", got, capBefore)
		}
	})

	t.Run("zeroes backing array", func(t *testing.T) {
		s := New[*int]()
		a, b, c := 1, 2, 3
		s.PushN(&a, &b, &c)

		s.Reset()

		assertTailZeroed(t, s)
	})

	t.Run("usable after Reset", func(t *testing.T) {
		s := fill(3)
		s.Reset()

		s.Push(42)

		if got, ok := s.Pop(); !ok || got != 42 {
			t.Errorf("Pop() after Reset() and Push(42) = (%d, %v), "+
				"want (42, true)", got, ok)
		}

		if got, ok := s.Pop(); ok || got != 0 {
			t.Errorf("Pop() on empty = (%d, %v), want (0, false)", got, ok)
		}
	})
}

func TestStack_Shrink(t *testing.T) {
	t.Run("reduces capacity to length and keeps contents", func(t *testing.T) {
		s := NewWithCap[int](64)
		s.PushN(1, 2, 3)

		s.Shrink()

		if got, want := s.Cap(), s.Len(); got != want {
			t.Errorf("Cap() after Shrink() = %d, want Len() = %d", got, want)
		}

		if got, want := s.Slice(), []int{3, 2, 1}; !slices.Equal(got, want) {
			t.Errorf("Slice() after Shrink() = %v, want %v", got, want)
		}
	})

	t.Run("allocates a new backing array", func(t *testing.T) {
		s := NewWithCap[int](64)
		s.PushN(1, 2, 3)
		old := &s.data[0]

		s.Shrink()

		if got := &s.data[0]; got == old {
			t.Errorf("&data[0] after Shrink() = %p, want a new backing array",
				got)
		}
	})

	t.Run("empty stack drops to zero capacity", func(t *testing.T) {
		s := NewWithCap[int](8)

		s.Shrink()

		if got := s.Cap(); got != 0 {
			t.Errorf("Cap() after Shrink() on empty = %d, want 0", got)
		}
	})

	t.Run("usable after Shrink", func(t *testing.T) {
		s := NewWithCap[int](16)
		s.Push(1)
		s.Shrink()

		s.Push(2)

		if got, want := s.Slice(), []int{2, 1}; !slices.Equal(got, want) {
			t.Errorf("Slice() after Shrink() and Push(2) = %v, want %v",
				got, want)
		}
	})
}

func TestStack_Clip(t *testing.T) {
	t.Run("reduces capacity to length and keeps contents", func(t *testing.T) {
		s := NewWithCap[int](64)
		s.PushN(1, 2, 3)

		s.Clip()

		if got, want := s.Cap(), s.Len(); got != want {
			t.Errorf("Cap() after Clip() = %d, want Len() = %d", got, want)
		}

		if got, want := s.Slice(), []int{3, 2, 1}; !slices.Equal(got, want) {
			t.Errorf("Slice() after Clip() = %v, want %v", got, want)
		}
	})

	t.Run("does not reallocate", func(t *testing.T) {
		s := NewWithCap[int](64)
		s.PushN(1, 2, 3)
		old := &s.data[0]

		s.Clip()

		if got := &s.data[0]; got != old {
			t.Errorf("&data[0] after Clip() = %p, want %p (no copy)", got, old)
		}
	})

	t.Run("empty stack drops to zero capacity", func(t *testing.T) {
		s := NewWithCap[int](8)

		s.Clip()

		if got := s.Cap(); got != 0 {
			t.Errorf("Cap() after Clip() on empty = %d, want 0", got)
		}
	})

	t.Run("usable after Clip", func(t *testing.T) {
		s := NewWithCap[int](16)
		s.Push(1)
		s.Clip()

		s.Push(2)

		if got, want := s.Slice(), []int{2, 1}; !slices.Equal(got, want) {
			t.Errorf("Slice() after Clip() and Push(2) = %v, want %v",
				got, want)
		}
	})
}

func TestStack_Grow(t *testing.T) {
	t.Run("reserves capacity without changing contents", func(t *testing.T) {
		s := New[int]()
		s.PushN(1, 2)

		s.Grow(100)

		if got := s.Cap(); got < 102 {
			t.Errorf("Cap() after Grow(100) = %d, want >= 102", got)
		}

		if got, want := s.Slice(), []int{2, 1}; !slices.Equal(got, want) {
			t.Errorf("Slice() after Grow(100) = %v, want %v", got, want)
		}
	})

	t.Run("no reallocation within grown capacity", func(t *testing.T) {
		s := New[int]()
		s.Grow(8)
		capBefore := s.Cap()

		for i := range 8 {
			s.Push(i)
		}

		if got := s.Cap(); got != capBefore {
			t.Errorf("Cap() after 8 Push() = %d, want %d", got, capBefore)
		}
	})

	t.Run("non-positive n is a no-op", func(t *testing.T) {
		s := fill(1)
		capBefore := s.Cap()

		s.Grow(0)

		if got := s.Cap(); got != capBefore {
			t.Errorf("Cap() after Grow(0) = %d, want %d", got, capBefore)
		}

		s.Grow(-1)

		if got := s.Cap(); got != capBefore {
			t.Errorf("Cap() after Grow(-1) = %d, want %d", got, capBefore)
		}
	})
}

// TestStack_RandomOps runs a seeded random sequence of operations against a
// slice model and checks that both agree after every step.
func TestStack_RandomOps(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	s := New[int]()
	var model []int // bottom to top

	for i := range 2000 {
		switch op := r.IntN(10); {
		case op < 3:
			s.Push(i)
			model = append(model, i)
		case op < 4:
			s.PushN(i, i+1)
			model = append(model, i, i+1)
		case op < 6:
			want, wantOK := 0, false
			if len(model) > 0 {
				want, wantOK = model[len(model)-1], true
				model = model[:len(model)-1]
			}

			if got, ok := s.Pop(); ok != wantOK || got != want {
				t.Fatalf("step %d: Pop() = (%d, %v), want (%d, %v)",
					i, got, ok, want, wantOK)
			}
		case op < 7:
			n := r.IntN(5) - 1
			k := min(max(n, 0), len(model))
			want := slices.Clone(model[len(model)-k:])
			slices.Reverse(want)
			model = model[:len(model)-k]

			if got := s.PopN(n); !slices.Equal(got, want) {
				t.Fatalf("step %d: PopN(%d) = %v, want %v", i, n, got, want)
			}
		case op < 8:
			want, wantOK := 0, false
			if len(model) > 0 {
				want, wantOK = model[len(model)-1], true
			}

			if got, ok := s.Peek(); ok != wantOK || got != want {
				t.Fatalf("step %d: Peek() = (%d, %v), want (%d, %v)",
					i, got, ok, want, wantOK)
			}
		default:
			switch r.IntN(5) {
			case 0:
				s.Clear()
				model = nil
			case 1:
				s.Reset()
				model = nil
			case 2:
				s.Shrink()
			case 3:
				s.Clip()
			default:
				s.Grow(r.IntN(16))
			}
		}

		want := slices.Clone(model)
		slices.Reverse(want)

		if got := s.Slice(); !slices.Equal(got, want) {
			t.Fatalf("step %d: Slice() = %v, want %v", i, got, want)
		}

		if got := s.Len(); got != len(model) {
			t.Fatalf("step %d: Len() = %d, want %d", i, got, len(model))
		}
	}
}

// TestStack_NilReceiver checks that every read-only and removal method treats
// a nil receiver as an empty stack.
func TestStack_NilReceiver(t *testing.T) {
	var s *Stack[int]

	if got := s.Len(); got != 0 {
		t.Errorf("nil.Len() = %d, want 0", got)
	}

	if got := s.Cap(); got != 0 {
		t.Errorf("nil.Cap() = %d, want 0", got)
	}

	if got := s.IsEmpty(); !got {
		t.Errorf("nil.IsEmpty() = %v, want true", got)
	}

	if got, ok := s.Peek(); ok || got != 0 {
		t.Errorf("nil.Peek() = (%d, %v), want (0, false)", got, ok)
	}

	count := 0
	for range s.All() {
		count++
	}

	if count != 0 {
		t.Errorf("nil.All() count = %d, want 0", count)
	}

	c := s.Clone()
	if c == nil {
		t.Fatalf("nil.Clone() = nil, want a usable empty stack")
	}

	if got := c.Len(); got != 0 {
		t.Errorf("nil.Clone().Len() = %d, want 0", got)
	}

	c.Push(42)

	if got, ok := c.Pop(); !ok || got != 42 {
		t.Errorf("Pop() after Push(42) on nil.Clone() = (%d, %v), "+
			"want (42, true)", got, ok)
	}

	if got := s.Slice(); got != nil {
		t.Errorf("nil.Slice() = %v, want nil", got)
	}

	if got, ok := s.Pop(); ok || got != 0 {
		t.Errorf("nil.Pop() = (%d, %v), want (0, false)", got, ok)
	}

	if got := s.PopN(3); got != nil {
		t.Errorf("nil.PopN(3) = %v, want nil", got)
	}
}

// TestStack_NilReceiverMutatorsPanic checks that every method that stores,
// clears, resets, or resizes data panics on a nil receiver.
func TestStack_NilReceiverMutatorsPanic(t *testing.T) {
	tests := []struct {
		name string
		op   string
		call func(*Stack[int])
	}{
		{
			name: "Push",
			op:   "Push",
			call: func(s *Stack[int]) { s.Push(1) },
		},
		{
			name: "PushN",
			op:   "PushN",
			call: func(s *Stack[int]) { s.PushN(1, 2) },
		},
		{
			name: "PushN with no arguments",
			op:   "PushN",
			call: func(s *Stack[int]) { s.PushN() },
		},
		{
			name: "Clear",
			op:   "Clear",
			call: func(s *Stack[int]) { s.Clear() },
		},
		{
			name: "Reset",
			op:   "Reset",
			call: func(s *Stack[int]) { s.Reset() },
		},
		{
			name: "Shrink",
			op:   "Shrink",
			call: func(s *Stack[int]) { s.Shrink() },
		},
		{
			name: "Clip",
			op:   "Clip",
			call: func(s *Stack[int]) { s.Clip() },
		},
		{
			name: "Grow",
			op:   "Grow",
			call: func(s *Stack[int]) { s.Grow(8) },
		},
		{
			name: "Grow zero",
			op:   "Grow",
			call: func(s *Stack[int]) { s.Grow(0) },
		},
		{
			name: "Grow negative",
			op:   "Grow",
			call: func(s *Stack[int]) { s.Grow(-1) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s *Stack[int]
			sharedtest.Panics(t, tt.name, "stack: "+tt.op+" on nil receiver",
				func() { tt.call(s) })
		})
	}
}

func TestStack_ZeroValue(t *testing.T) {
	var s Stack[int]

	if got := s.Len(); got != 0 {
		t.Errorf("Len() on zero value = %d, want 0", got)
	}

	if got := s.Cap(); got != 0 {
		t.Errorf("Cap() on zero value = %d, want 0", got)
	}

	if got := s.IsEmpty(); !got {
		t.Errorf("IsEmpty() on zero value = %v, want true", got)
	}

	if got, ok := s.Peek(); ok || got != 0 {
		t.Errorf("Peek() on zero value = (%d, %v), want (0, false)", got, ok)
	}

	if got, ok := s.Pop(); ok || got != 0 {
		t.Errorf("Pop() on zero value = (%d, %v), want (0, false)", got, ok)
	}

	s.Clear()
	s.Reset()
	s.Shrink()
	s.Clip()
	s.Grow(4)

	if got := s.Cap(); got < 4 {
		t.Errorf("Cap() after Grow(4) = %d, want >= 4", got)
	}

	s.PushN(1, 2)

	if got, ok := s.Pop(); !ok || got != 2 {
		t.Errorf("Pop() after PushN(1, 2) on zero value = (%d, %v), "+
			"want (2, true)", got, ok)
	}
}
