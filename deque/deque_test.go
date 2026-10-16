package deque

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Nergous/structures/internal/shared/sharedtest"
)

// assertNewEmpty fails the test unless New returns an empty deque with no
// preallocated capacity for element type T.
func assertNewEmpty[T any](t *testing.T) {
	t.Helper()

	d := New[T]()

	if got := d.Len(); got != 0 {
		t.Errorf("Len() = %d, want 0", got)
	}

	if got := d.Cap(); got != 0 {
		t.Errorf("Cap() = %d, want 0", got)
	}
}

// assertTailZeroed fails the test unless every slot of the backing array
// past Len holds the zero value of T. The live elements of d must start at
// index 0 of the backing array.
func assertTailZeroed[T comparable](t *testing.T, d *Deque[T]) {
	t.Helper()

	var zero T
	n := d.Len()
	for i, v := range d.buf.Data()[n:] {
		if v != zero {
			t.Errorf("data[%d] = %v, want zero value", n+i, v)
		}
	}
}

// fill returns a deque holding the elements 0..n-1 from front to back.
func fill(n int) *Deque[int] {
	d := New[int]()
	for i := range n {
		d.PushBack(i)
	}

	return d
}

// wrapped returns a full deque of capacity 4 holding 3, 4, 5, 6 from front to
// back, whose live elements wrap around the end of the backing array.
func wrapped() *Deque[int] {
	d := NewWithCap[int](4)
	d.PushBackN(1, 2, 3, 4)
	d.PopFront()
	d.PopFront()
	d.PushBackN(5, 6)

	return d
}

func TestDeque_New(t *testing.T) {
	t.Run("int", assertNewEmpty[int])
	t.Run("string", assertNewEmpty[string])
	t.Run("float32", assertNewEmpty[float32])
}

func TestDeque_NewWithCap(t *testing.T) {
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
			d := NewWithCap[int](tt.n)

			if got := d.Len(); got != 0 {
				t.Errorf("Len() = %d, want 0", got)
			}

			if got := d.Cap(); got != tt.want {
				t.Errorf("Cap() = %d, want %d", got, tt.want)
			}

			d.PushBack(1)

			if got, ok := d.PopFront(); !ok || got != 1 {
				t.Errorf("PopFront() after PushBack(1) = (%d, %v), "+
					"want (1, true)", got, ok)
			}
		})
	}
}

func TestDeque_Len(t *testing.T) {
	d := New[int]()
	d.PushBackN(1, 2, 3)

	if got := d.Len(); got != 3 {
		t.Errorf("Len() after PushBackN(1, 2, 3) = %d, want 3", got)
	}

	d.PopBack()

	if got := d.Len(); got != 2 {
		t.Errorf("Len() after PopBack() = %d, want 2", got)
	}
}

func TestDeque_Cap(t *testing.T) {
	t.Run("new deque has zero cap", func(t *testing.T) {
		d := New[int]()

		if got := d.Cap(); got != 0 {
			t.Errorf("Cap() = %d, want 0", got)
		}
	})

	t.Run("never less than Len while growing", func(t *testing.T) {
		d := New[int]()
		for i := range 100 {
			if i%2 == 0 {
				d.PushFront(i)
			} else {
				d.PushBack(i)
			}

			if c, l := d.Cap(), d.Len(); c < l {
				t.Fatalf("Cap() #%d = %d, want >= Len() = %d", i, c, l)
			}
		}
	})

	t.Run("no reallocation within preallocated capacity", func(t *testing.T) {
		d := NewWithCap[int](4)
		for i := range 4 {
			d.PushBack(i)
		}

		if got := d.Cap(); got != 4 {
			t.Errorf("Cap() after 4 PushBack() = %d, want 4", got)
		}
	})

	t.Run("stays stable when elements are popped", func(t *testing.T) {
		d := NewWithCap[int](4)
		d.PushBackN(1, 2, 3, 4)
		d.PopFront()
		d.PopBack()

		if got := d.Cap(); got != 4 {
			t.Errorf("Cap() after PopFront(), PopBack() = %d, want 4", got)
		}
	})

	t.Run("reuses freed slots without growing", func(t *testing.T) {
		d := NewWithCap[int](4)
		d.PushBackN(1, 2, 3, 4)
		d.PopFront()
		d.PopBack()
		d.PushFront(0)
		d.PushBack(5)

		if got := d.Cap(); got != 4 {
			t.Errorf("Cap() after refilling freed slots = %d, want 4", got)
		}

		if got, want := d.Slice(), []int{0, 2, 3, 5}; !slices.Equal(got, want) {
			t.Errorf("Slice() after refilling freed slots = %v, want %v",
				got, want)
		}
	})
}

func TestDeque_IsEmpty(t *testing.T) {
	d := New[int]()

	if got := d.IsEmpty(); !got {
		t.Errorf("IsEmpty() on empty = %v, want true", got)
	}

	d.PushBack(1)

	if got := d.IsEmpty(); got {
		t.Errorf("IsEmpty() after PushBack(1) = %v, want false", got)
	}

	d.PopFront()

	if got := d.IsEmpty(); !got {
		t.Errorf("IsEmpty() after PopFront() = %v, want true", got)
	}
}

func TestDeque_Front(t *testing.T) {
	t.Run("empty returns zero value and false", func(t *testing.T) {
		d := New[int]()

		if got, ok := d.Front(); ok || got != 0 {
			t.Errorf("Front() on empty = (%d, %v), want (0, false)", got, ok)
		}
	})

	t.Run("returns front without removing it", func(t *testing.T) {
		d := New[int]()
		d.PushBackN(1, 2)

		if got, ok := d.Front(); !ok || got != 1 {
			t.Errorf("Front() = (%d, %v), want (1, true)", got, ok)
		}

		if got := d.Len(); got != 2 {
			t.Errorf("Len() after Front() = %d, want 2", got)
		}
	})

	t.Run("reflects PushFront", func(t *testing.T) {
		d := New[int]()
		d.PushBack(1)
		d.PushFront(0)

		if got, ok := d.Front(); !ok || got != 0 {
			t.Errorf("Front() after PushFront(0) = (%d, %v), want (0, true)",
				got, ok)
		}
	})
}

func TestDeque_Back(t *testing.T) {
	t.Run("empty returns zero value and false", func(t *testing.T) {
		d := New[int]()

		if got, ok := d.Back(); ok || got != 0 {
			t.Errorf("Back() on empty = (%d, %v), want (0, false)", got, ok)
		}
	})

	t.Run("returns back without removing it", func(t *testing.T) {
		d := New[int]()
		d.PushBackN(1, 2)

		if got, ok := d.Back(); !ok || got != 2 {
			t.Errorf("Back() = (%d, %v), want (2, true)", got, ok)
		}

		if got := d.Len(); got != 2 {
			t.Errorf("Len() after Back() = %d, want 2", got)
		}
	})

	t.Run("reflects PushBack across wraparound", func(t *testing.T) {
		d := wrapped()

		if got, ok := d.Back(); !ok || got != 6 {
			t.Errorf("Back() across wraparound = (%d, %v), want (6, true)",
				got, ok)
		}
	})
}

func TestDeque_All(t *testing.T) {
	t.Run("yields front to back", func(t *testing.T) {
		d := New[int]()
		d.PushBack(2)
		d.PushFront(1)
		d.PushBack(3)

		got := slices.Collect(d.All())
		if want := []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("All() = %v, want %v", got, want)
		}
	})

	t.Run("yields front to back across wraparound", func(t *testing.T) {
		d := wrapped()

		got := slices.Collect(d.All())
		if want := []int{3, 4, 5, 6}; !slices.Equal(got, want) {
			t.Errorf("All() across wraparound = %v, want %v", got, want)
		}
	})

	t.Run("empty deque yields nothing", func(t *testing.T) {
		d := New[int]()

		if got := slices.Collect(d.All()); len(got) != 0 {
			t.Errorf("All() on empty = %v, want no elements", got)
		}
	})

	t.Run("does not consume", func(t *testing.T) {
		d := fill(3)

		_ = slices.Collect(d.All())

		if got := d.Len(); got != 3 {
			t.Errorf("Len() after All() = %d, want 3", got)
		}

		if got, ok := d.PopFront(); !ok || got != 0 {
			t.Errorf("PopFront() after All() = (%d, %v), want (0, true)",
				got, ok)
		}
	})

	t.Run("break stops iteration early", func(t *testing.T) {
		d := fill(5)

		var got []int
		for v := range d.All() {
			got = append(got, v)
			if len(got) == 2 {
				break
			}
		}

		if want := []int{0, 1}; !slices.Equal(got, want) {
			t.Errorf("All() with early break = %v, want %v", got, want)
		}

		if got := d.Len(); got != 5 {
			t.Errorf("Len() after break = %d, want 5", got)
		}
	})

	t.Run("break across wraparound", func(t *testing.T) {
		d := wrapped()

		var got []int
		for v := range d.All() {
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

func TestDeque_Backward(t *testing.T) {
	t.Run("yields back to front", func(t *testing.T) {
		d := New[int]()
		d.PushBack(2)
		d.PushFront(1)
		d.PushBack(3)

		got := slices.Collect(d.Backward())
		if want := []int{3, 2, 1}; !slices.Equal(got, want) {
			t.Errorf("Backward() = %v, want %v", got, want)
		}
	})

	t.Run("yields back to front across wraparound", func(t *testing.T) {
		d := wrapped()

		got := slices.Collect(d.Backward())
		if want := []int{6, 5, 4, 3}; !slices.Equal(got, want) {
			t.Errorf("Backward() across wraparound = %v, want %v", got, want)
		}
	})

	t.Run("empty deque yields nothing", func(t *testing.T) {
		d := New[int]()

		if got := slices.Collect(d.Backward()); len(got) != 0 {
			t.Errorf("Backward() on empty = %v, want no elements", got)
		}
	})

	t.Run("does not consume", func(t *testing.T) {
		d := fill(3)

		_ = slices.Collect(d.Backward())

		if got := d.Len(); got != 3 {
			t.Errorf("Len() after Backward() = %d, want 3", got)
		}

		if got, ok := d.PopBack(); !ok || got != 2 {
			t.Errorf("PopBack() after Backward() = (%d, %v), want (2, true)",
				got, ok)
		}
	})

	t.Run("break stops iteration early", func(t *testing.T) {
		d := fill(5)

		var got []int
		for v := range d.Backward() {
			got = append(got, v)
			if len(got) == 2 {
				break
			}
		}

		if want := []int{4, 3}; !slices.Equal(got, want) {
			t.Errorf("Backward() with early break = %v, want %v", got, want)
		}

		if got := d.Len(); got != 5 {
			t.Errorf("Len() after break = %d, want 5", got)
		}
	})

	t.Run("break across wraparound", func(t *testing.T) {
		d := wrapped()

		// Backward walks the wrapped tail segment first, so the break must
		// land inside it.
		var got []int
		for v := range d.Backward() {
			got = append(got, v)
			if len(got) == 1 {
				break
			}
		}

		if want := []int{6}; !slices.Equal(got, want) {
			t.Errorf("Backward() with early break = %v, want %v", got, want)
		}
	})
}

func TestDeque_Clone(t *testing.T) {
	t.Run("copies all elements in order", func(t *testing.T) {
		d := fill(3)

		c := d.Clone()

		if got, want := c.Slice(), []int{0, 1, 2}; !slices.Equal(got, want) {
			t.Errorf("Slice() of clone = %v, want %v", got, want)
		}
	})

	t.Run("copies across wraparound", func(t *testing.T) {
		d := wrapped()

		c := d.Clone()

		want := []int{3, 4, 5, 6}
		if got := c.Slice(); !slices.Equal(got, want) {
			t.Errorf("Slice() of clone across wraparound = %v, want %v",
				got, want)
		}
	})

	t.Run("capacity equals length", func(t *testing.T) {
		d := NewWithCap[int](64)
		d.PushBackN(1, 2, 3, 4, 5)

		c := d.Clone()

		if got, want := c.Cap(), c.Len(); got != want {
			t.Errorf("Cap() of clone = %d, want Len() = %d", got, want)
		}
	})

	t.Run("is independent of the original", func(t *testing.T) {
		d := fill(3)

		c := d.Clone()
		c.PushBack(3)
		d.PopFront()

		if got, want := c.Slice(), []int{0, 1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("Slice() of clone = %v, want %v", got, want)
		}

		if got, want := d.Slice(), []int{1, 2}; !slices.Equal(got, want) {
			t.Errorf("Slice() of original = %v, want %v", got, want)
		}
	})

	t.Run("clone of empty deque is empty and usable", func(t *testing.T) {
		d := New[int]()

		c := d.Clone()

		if got := c.Len(); got != 0 {
			t.Errorf("Len() of clone = %d, want 0", got)
		}

		c.PushBack(1)

		if got, ok := c.PopFront(); !ok || got != 1 {
			t.Errorf("PopFront() of clone after PushBack(1) = (%d, %v), "+
				"want (1, true)", got, ok)
		}
	})
}

func TestDeque_Slice(t *testing.T) {
	t.Run("returns elements front to back", func(t *testing.T) {
		d := New[int]()
		d.PushBack(2)
		d.PushFront(1)
		d.PushBack(3)

		if got, want := d.Slice(), []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("Slice() = %v, want %v", got, want)
		}
	})

	t.Run("returns elements across wraparound", func(t *testing.T) {
		d := wrapped()

		if got, want := d.Slice(), []int{3, 4, 5, 6}; !slices.Equal(got, want) {
			t.Errorf("Slice() across wraparound = %v, want %v", got, want)
		}
	})

	t.Run("returns an independent copy", func(t *testing.T) {
		d := fill(2)

		out := d.Slice()
		for i := range out {
			out[i] = 999
		}

		if got, want := d.Slice(), []int{0, 1}; !slices.Equal(got, want) {
			t.Errorf("Slice() after mutating a copy = %v, want %v", got, want)
		}
	})

	t.Run("does not consume", func(t *testing.T) {
		d := fill(2)

		_ = d.Slice()

		if got := d.Len(); got != 2 {
			t.Errorf("Len() after Slice() = %d, want 2", got)
		}
	})

	t.Run("empty deque returns nil", func(t *testing.T) {
		drained := NewWithCap[int](8)
		drained.PushBack(1)
		drained.PopFront()

		reset := NewWithCap[int](8)
		reset.PushBack(1)
		reset.Reset()

		cleared := NewWithCap[int](8)
		cleared.PushBack(1)
		cleared.Clear()

		tests := []struct {
			name string
			d    *Deque[int]
		}{
			{
				name: "fresh New",
				d:    New[int](),
			},
			{
				name: "preallocated",
				d:    NewWithCap[int](8),
			},
			{
				name: "drained to zero",
				d:    drained,
			},
			{
				name: "after Reset",
				d:    reset,
			},
			{
				name: "after Clear",
				d:    cleared,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.d.Slice(); got != nil {
					t.Errorf("Slice() = %v, want nil", got)
				}
			})
		}
	})
}

func TestDeque_PushFront(t *testing.T) {
	t.Run("new elements become the front", func(t *testing.T) {
		d := New[int]()
		d.PushFront(1)
		d.PushFront(2)
		d.PushFront(3)

		if got, want := d.Slice(), []int{3, 2, 1}; !slices.Equal(got, want) {
			t.Errorf("Slice() after PushFront() = %v, want %v", got, want)
		}
	})

	t.Run("wraps around the backing array", func(t *testing.T) {
		d := NewWithCap[int](4)
		d.PushBackN(1, 2)

		d.PushFront(0)
		d.PushFront(-1)

		if got := d.Cap(); got != 4 {
			t.Errorf("Cap() after wrapping PushFront() = %d, want 4", got)
		}

		want := []int{-1, 0, 1, 2}
		if got := d.Slice(); !slices.Equal(got, want) {
			t.Errorf("Slice() after wrapping PushFront() = %v, want %v",
				got, want)
		}
	})

	t.Run("grows when full and keeps order", func(t *testing.T) {
		d := wrapped()

		d.PushFront(2)

		want := []int{2, 3, 4, 5, 6}
		if got := d.Slice(); !slices.Equal(got, want) {
			t.Errorf("Slice() after PushFront(2) on full deque = %v, want %v",
				got, want)
		}
	})
}

func TestDeque_PushBack(t *testing.T) {
	t.Run("new elements become the back", func(t *testing.T) {
		d := New[int]()
		d.PushBack(1)
		d.PushBack(2)
		d.PushBack(3)

		if got, want := d.Slice(), []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("Slice() after PushBack() = %v, want %v", got, want)
		}
	})

	t.Run("wraps around the backing array", func(t *testing.T) {
		d := wrapped()

		if got := d.Cap(); got != 4 {
			t.Errorf("Cap() after wrapping PushBack() = %d, want 4", got)
		}

		want := []int{3, 4, 5, 6}
		if got := d.Slice(); !slices.Equal(got, want) {
			t.Errorf("Slice() after wrapping PushBack() = %v, want %v",
				got, want)
		}
	})

	t.Run("grows when full and keeps order", func(t *testing.T) {
		d := wrapped()

		d.PushBack(7)

		want := []int{3, 4, 5, 6, 7}
		if got := d.Slice(); !slices.Equal(got, want) {
			t.Errorf("Slice() after PushBack(7) on full deque = %v, want %v",
				got, want)
		}
	})
}

func TestDeque_PushFrontN(t *testing.T) {
	t.Run("first argument becomes the new front", func(t *testing.T) {
		d := New[int]()
		d.PushBackN(4, 5)

		d.PushFrontN(1, 2, 3)

		want := []int{1, 2, 3, 4, 5}
		if got := d.Slice(); !slices.Equal(got, want) {
			t.Errorf("Slice() after PushFrontN(1, 2, 3) = %v, want %v",
				got, want)
		}
	})

	t.Run("no arguments is a no-op", func(t *testing.T) {
		d := New[int]()
		d.PushFrontN()

		if got := d.Len(); got != 0 {
			t.Errorf("Len() after PushFrontN() = %d, want 0", got)
		}
	})

	t.Run("wraps around the backing array", func(t *testing.T) {
		d := NewWithCap[int](8)
		d.PushBackN(1, 2)

		d.PushFrontN(-1, 0)

		if got := d.Cap(); got != 8 {
			t.Errorf("Cap() after wrapping PushFrontN() = %d, want 8", got)
		}

		want := []int{-1, 0, 1, 2}
		if got := d.Slice(); !slices.Equal(got, want) {
			t.Errorf("Slice() after wrapping PushFrontN() = %v, want %v",
				got, want)
		}
	})

	t.Run("grows to hold all elements", func(t *testing.T) {
		d := wrapped()

		d.PushFrontN(0, 1, 2)

		want := []int{0, 1, 2, 3, 4, 5, 6}
		if got := d.Slice(); !slices.Equal(got, want) {
			t.Errorf("Slice() after PushFrontN(0, 1, 2) = %v, want %v",
				got, want)
		}
	})
}

func TestDeque_PushBackN(t *testing.T) {
	t.Run("last argument becomes the new back", func(t *testing.T) {
		d := New[int]()
		d.PushBack(1)

		d.PushBackN(2, 3)

		if got, want := d.Slice(), []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("Slice() after PushBackN(2, 3) = %v, want %v", got, want)
		}
	})

	t.Run("no arguments is a no-op", func(t *testing.T) {
		d := New[int]()
		d.PushBackN()

		if got := d.Len(); got != 0 {
			t.Errorf("Len() after PushBackN() = %d, want 0", got)
		}
	})

	t.Run("wraps around the backing array", func(t *testing.T) {
		d := NewWithCap[int](4)
		d.PushBackN(1, 2, 3)
		d.PopFront()
		d.PopFront()

		d.PushBackN(4, 5, 6)

		if got := d.Cap(); got != 4 {
			t.Errorf("Cap() after wrapping PushBackN() = %d, want 4", got)
		}

		want := []int{3, 4, 5, 6}
		if got := d.Slice(); !slices.Equal(got, want) {
			t.Errorf("Slice() after wrapping PushBackN() = %v, want %v",
				got, want)
		}
	})

	t.Run("grows to hold all elements", func(t *testing.T) {
		d := wrapped()

		d.PushBackN(7, 8, 9)

		want := []int{3, 4, 5, 6, 7, 8, 9}
		if got := d.Slice(); !slices.Equal(got, want) {
			t.Errorf("Slice() after PushBackN(7, 8, 9) = %v, want %v",
				got, want)
		}
	})
}

func TestDeque_PopFront(t *testing.T) {
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
			name: "front to back order",
			push: []int{1, 2, 3},
			want: []int{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New[int]()
			d.PushBackN(tt.push...)

			for i, want := range tt.want {
				if got, ok := d.PopFront(); !ok || got != want {
					t.Errorf("PopFront() #%d = (%d, %v), want (%d, true)",
						i, got, ok, want)
				}
			}

			if got, ok := d.PopFront(); ok || got != 0 {
				t.Errorf("PopFront() on empty = (%d, %v), want (0, false)",
					got, ok)
			}
		})
	}

	t.Run("pops across wraparound", func(t *testing.T) {
		d := wrapped()

		for i, want := range []int{3, 4, 5, 6} {
			if got, ok := d.PopFront(); !ok || got != want {
				t.Errorf("PopFront() #%d = (%d, %v), want (%d, true)",
					i, got, ok, want)
			}
		}
	})

	t.Run("pops elements pushed at both ends", func(t *testing.T) {
		d := New[int]()
		d.PushBack(2)
		d.PushFront(1)
		d.PushBack(3)

		if got, ok := d.PopFront(); !ok || got != 1 {
			t.Errorf("PopFront() = (%d, %v), want (1, true)", got, ok)
		}

		if got, want := d.Slice(), []int{2, 3}; !slices.Equal(got, want) {
			t.Errorf("Slice() after PopFront() = %v, want %v", got, want)
		}
	})

	t.Run("zeroes vacated slot", func(t *testing.T) {
		d := New[*int]()
		a, b := 1, 2
		d.PushBackN(&a, &b)
		data := d.buf.Data()

		if _, ok := d.PopFront(); !ok {
			t.Fatalf("PopFront() ok = false, want true")
		}

		if got := data[0]; got != nil {
			t.Errorf("data[0] after PopFront() = %p, "+
				"want nil (PopFront must zero the slot)", got)
		}
	})
}

func TestDeque_PopBack(t *testing.T) {
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
			name: "back to front order",
			push: []int{1, 2, 3},
			want: []int{3, 2, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New[int]()
			d.PushBackN(tt.push...)

			for i, want := range tt.want {
				if got, ok := d.PopBack(); !ok || got != want {
					t.Errorf("PopBack() #%d = (%d, %v), want (%d, true)",
						i, got, ok, want)
				}
			}

			if got, ok := d.PopBack(); ok || got != 0 {
				t.Errorf("PopBack() on empty = (%d, %v), want (0, false)",
					got, ok)
			}
		})
	}

	t.Run("pops across wraparound", func(t *testing.T) {
		d := wrapped()

		for i, want := range []int{6, 5, 4, 3} {
			if got, ok := d.PopBack(); !ok || got != want {
				t.Errorf("PopBack() #%d = (%d, %v), want (%d, true)",
					i, got, ok, want)
			}
		}
	})

	t.Run("pops elements pushed at both ends", func(t *testing.T) {
		d := New[int]()
		d.PushBack(2)
		d.PushFront(1)
		d.PushBack(3)

		if got, ok := d.PopBack(); !ok || got != 3 {
			t.Errorf("PopBack() = (%d, %v), want (3, true)", got, ok)
		}

		if got, want := d.Slice(), []int{1, 2}; !slices.Equal(got, want) {
			t.Errorf("Slice() after PopBack() = %v, want %v", got, want)
		}
	})

	t.Run("zeroes vacated slot", func(t *testing.T) {
		d := New[*int]()
		a, b := 1, 2
		d.PushBackN(&a, &b)
		data := d.buf.Data()

		if _, ok := d.PopBack(); !ok {
			t.Fatalf("PopBack() ok = false, want true")
		}

		if got := data[1]; got != nil {
			t.Errorf("data[1] after PopBack() = %p, "+
				"want nil (PopBack must zero the slot)", got)
		}
	})
}

func TestDeque_PopFrontN(t *testing.T) {
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
			name: "several in front to back order",
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
			d := New[int]()
			d.PushBackN(tt.push...)
			capBefore := d.Cap()

			got := d.PopFrontN(tt.n)

			if !slices.Equal(got, tt.want) || (got == nil) != (tt.want == nil) {
				t.Errorf("PopFrontN(%d) = %v, want %v", tt.n, got, tt.want)
			}

			if got := d.Slice(); !slices.Equal(got, tt.left) {
				t.Errorf("Slice() after PopFrontN(%d) = %v, want %v",
					tt.n, got, tt.left)
			}

			if got := d.Cap(); got != capBefore {
				t.Errorf("Cap() after PopFrontN(%d) = %d, want %d",
					tt.n, got, capBefore)
			}
		})
	}

	t.Run("works across wraparound", func(t *testing.T) {
		d := wrapped()

		want := []int{3, 4, 5}
		if got := d.PopFrontN(3); !slices.Equal(got, want) {
			t.Errorf("PopFrontN(3) across wraparound = %v, want %v",
				got, want)
		}

		if got, want := d.Slice(), []int{6}; !slices.Equal(got, want) {
			t.Errorf("Slice() after PopFrontN(3) = %v, want %v", got, want)
		}
	})

	t.Run("zeroes vacated slots", func(t *testing.T) {
		d := NewWithCap[*int](4)
		a, b, c := 1, 2, 3
		d.PushBackN(&a, &b, &c)

		d.PopFrontN(3)

		assertTailZeroed(t, d)
	})
}

func TestDeque_PopBackN(t *testing.T) {
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
			left: []int{1, 2, 3, 4},
		},
		{
			name: "several in back to front order",
			push: []int{1, 2, 3, 4, 5},
			n:    3,
			want: []int{5, 4, 3},
			left: []int{1, 2},
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
			d := New[int]()
			d.PushBackN(tt.push...)
			capBefore := d.Cap()

			got := d.PopBackN(tt.n)

			if !slices.Equal(got, tt.want) || (got == nil) != (tt.want == nil) {
				t.Errorf("PopBackN(%d) = %v, want %v", tt.n, got, tt.want)
			}

			if got := d.Slice(); !slices.Equal(got, tt.left) {
				t.Errorf("Slice() after PopBackN(%d) = %v, want %v",
					tt.n, got, tt.left)
			}

			if got := d.Cap(); got != capBefore {
				t.Errorf("Cap() after PopBackN(%d) = %d, want %d",
					tt.n, got, capBefore)
			}
		})
	}

	t.Run("works across wraparound", func(t *testing.T) {
		d := wrapped()

		want := []int{6, 5, 4}
		if got := d.PopBackN(3); !slices.Equal(got, want) {
			t.Errorf("PopBackN(3) across wraparound = %v, want %v", got, want)
		}

		if got, want := d.Slice(), []int{3}; !slices.Equal(got, want) {
			t.Errorf("Slice() after PopBackN(3) = %v, want %v", got, want)
		}
	})

	t.Run("zeroes vacated slots", func(t *testing.T) {
		d := NewWithCap[*int](4)
		a, b, c := 1, 2, 3
		d.PushBackN(&a, &b, &c)

		d.PopBackN(2)

		assertTailZeroed(t, d)
	})
}

func TestDeque_Clear(t *testing.T) {
	d := NewWithCap[int](8)
	d.PushBackN(0, 1, 2, 3, 4)

	d.Clear()

	if got := d.Len(); got != 0 {
		t.Errorf("Len() after Clear() = %d, want 0", got)
	}

	if got := d.Cap(); got != 0 {
		t.Errorf("Cap() after Clear() = %d, want 0", got)
	}

	if got, ok := d.PopFront(); ok || got != 0 {
		t.Errorf("PopFront() after Clear() = (%d, %v), want (0, false)",
			got, ok)
	}

	d.PushBack(99)

	if got, ok := d.PopFront(); !ok || got != 99 {
		t.Errorf("PopFront() after Clear() and PushBack(99) = (%d, %v), "+
			"want (99, true)", got, ok)
	}
}

func TestDeque_Reset(t *testing.T) {
	t.Run("Reset keeps capacity", func(t *testing.T) {
		d := fill(10)
		capBefore := d.Cap()

		d.Reset()

		if got := d.Len(); got != 0 {
			t.Errorf("Len() after Reset() = %d, want 0", got)
		}

		if got := d.Cap(); got != capBefore {
			t.Errorf("Cap() after Reset() = %d, want %d", got, capBefore)
		}
	})

	t.Run("keeps full capacity after pops", func(t *testing.T) {
		d := NewWithCap[int](8)
		d.PushBackN(1, 2, 3, 4, 5)
		d.PopFront()
		d.PopBack()

		d.Reset()

		if got := d.Cap(); got != 8 {
			t.Errorf("Cap() after PopFront(), PopBack(), Reset() = %d, "+
				"want 8", got)
		}
	})

	t.Run("zeroes backing array", func(t *testing.T) {
		d := NewWithCap[*int](4)
		a, b, c, e := 1, 2, 3, 4
		d.PushBackN(&a, &b, &c)
		d.PopFront()
		d.PushBackN(&e, &a)

		d.Reset()

		assertTailZeroed(t, d)
	})

	t.Run("usable after Reset", func(t *testing.T) {
		d := fill(3)
		d.Reset()

		d.PushBack(42)

		if got, ok := d.PopFront(); !ok || got != 42 {
			t.Errorf("PopFront() after Reset() and PushBack(42) = (%d, %v), "+
				"want (42, true)", got, ok)
		}

		if got, ok := d.PopFront(); ok || got != 0 {
			t.Errorf("PopFront() on empty = (%d, %v), want (0, false)",
				got, ok)
		}
	})
}

func TestDeque_Shrink(t *testing.T) {
	t.Run("reduces capacity to length and keeps contents", func(t *testing.T) {
		d := NewWithCap[int](64)
		d.PushBackN(1, 2, 3)

		d.Shrink()

		if got, want := d.Cap(), d.Len(); got != want {
			t.Errorf("Cap() after Shrink() = %d, want Len() = %d", got, want)
		}

		if got, want := d.Slice(), []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("Slice() after Shrink() = %v, want %v", got, want)
		}
	})

	t.Run("allocates a new backing array", func(t *testing.T) {
		d := NewWithCap[int](64)
		d.PushBackN(1, 2, 3)
		old := &d.buf.Data()[0]

		d.Shrink()

		if got := &d.buf.Data()[0]; got == old {
			t.Errorf("&data[0] after Shrink() = %p, want a new backing array",
				got)
		}
	})

	t.Run("compacts wrapped elements", func(t *testing.T) {
		d := wrapped()
		d.PopBack()

		d.Shrink()

		if got, want := d.Cap(), d.Len(); got != want {
			t.Errorf("Cap() after Shrink() = %d, want Len() = %d", got, want)
		}

		if got, want := d.Slice(), []int{3, 4, 5}; !slices.Equal(got, want) {
			t.Errorf("Slice() after Shrink() = %v, want %v", got, want)
		}
	})

	t.Run("empty deque drops to zero capacity", func(t *testing.T) {
		d := NewWithCap[int](8)

		d.Shrink()

		if got := d.Cap(); got != 0 {
			t.Errorf("Cap() after Shrink() on empty = %d, want 0", got)
		}
	})

	t.Run("usable after Shrink", func(t *testing.T) {
		d := NewWithCap[int](16)
		d.PushBack(1)
		d.Shrink()

		d.PushBack(2)

		if got, want := d.Slice(), []int{1, 2}; !slices.Equal(got, want) {
			t.Errorf("Slice() after Shrink() and PushBack(2) = %v, want %v",
				got, want)
		}
	})
}

func TestDeque_Clip(t *testing.T) {
	t.Run("reduces capacity to length and keeps contents", func(t *testing.T) {
		d := NewWithCap[int](64)
		d.PushBackN(1, 2, 3)

		d.Clip()

		if got, want := d.Cap(), d.Len(); got != want {
			t.Errorf("Cap() after Clip() = %d, want Len() = %d", got, want)
		}

		if got, want := d.Slice(), []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("Slice() after Clip() = %v, want %v", got, want)
		}
	})

	t.Run("does not reallocate", func(t *testing.T) {
		d := NewWithCap[int](64)
		d.PushBackN(1, 2, 3)
		old := &d.buf.Data()[0]

		d.Clip()

		if got := &d.buf.Data()[0]; got != old {
			t.Errorf("&data[0] after Clip() = %p, want %p (no copy)", got, old)
		}
	})

	t.Run("compacts wrapped elements", func(t *testing.T) {
		d := wrapped()
		d.PopBack()

		d.Clip()

		if got, want := d.Cap(), d.Len(); got != want {
			t.Errorf("Cap() after Clip() = %d, want Len() = %d", got, want)
		}

		if got, want := d.Slice(), []int{3, 4, 5}; !slices.Equal(got, want) {
			t.Errorf("Slice() after Clip() = %v, want %v", got, want)
		}
	})

	t.Run("empty deque drops to zero capacity", func(t *testing.T) {
		d := NewWithCap[int](8)

		d.Clip()

		if got := d.Cap(); got != 0 {
			t.Errorf("Cap() after Clip() on empty = %d, want 0", got)
		}
	})

	t.Run("usable after Clip", func(t *testing.T) {
		d := NewWithCap[int](16)
		d.PushBack(1)
		d.Clip()

		d.PushBack(2)

		if got, want := d.Slice(), []int{1, 2}; !slices.Equal(got, want) {
			t.Errorf("Slice() after Clip() and PushBack(2) = %v, want %v",
				got, want)
		}
	})
}

func TestDeque_Grow(t *testing.T) {
	t.Run("reserves capacity without changing contents", func(t *testing.T) {
		d := wrapped()

		d.Grow(100)

		if got := d.Cap(); got < 104 {
			t.Errorf("Cap() after Grow(100) = %d, want >= 104", got)
		}

		if got, want := d.Slice(), []int{3, 4, 5, 6}; !slices.Equal(got, want) {
			t.Errorf("Slice() after Grow(100) = %v, want %v", got, want)
		}
	})

	t.Run("no reallocation within grown capacity", func(t *testing.T) {
		d := New[int]()
		d.Grow(8)
		capBefore := d.Cap()

		for i := range 8 {
			d.PushFront(i)
		}

		if got := d.Cap(); got != capBefore {
			t.Errorf("Cap() after 8 PushFront() = %d, want %d",
				got, capBefore)
		}
	})

	t.Run("non-positive n is a no-op", func(t *testing.T) {
		d := fill(3)
		capBefore := d.Cap()

		d.Grow(0)

		if got := d.Cap(); got != capBefore {
			t.Errorf("Cap() after Grow(0) = %d, want %d", got, capBefore)
		}

		d.Grow(-1)

		if got := d.Cap(); got != capBefore {
			t.Errorf("Cap() after Grow(-1) = %d, want %d", got, capBefore)
		}
	})
}

// TestDeque_RandomOps runs a seeded random sequence of operations against a
// slice model and checks that both agree after every step.
func TestDeque_RandomOps(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	d := New[int]()
	var model []int // front to back

	for i := range 5000 {
		switch op := r.IntN(11); {
		case op < 2:
			d.PushFront(i)
			model = slices.Insert(model, 0, i)
		case op < 4:
			d.PushBack(i)
			model = append(model, i)
		case op < 5:
			d.PushFrontN(i, i+1)
			model = slices.Insert(model, 0, i, i+1)
		case op < 6:
			d.PushBackN(i, i+1)
			model = append(model, i, i+1)
		case op < 7:
			want, wantOK := 0, false
			if len(model) > 0 {
				want, wantOK = model[0], true
				model = model[1:]
			}

			if got, ok := d.PopFront(); ok != wantOK || got != want {
				t.Fatalf("step %d: PopFront() = (%d, %v), want (%d, %v)",
					i, got, ok, want, wantOK)
			}
		case op < 8:
			want, wantOK := 0, false
			if len(model) > 0 {
				want, wantOK = model[len(model)-1], true
				model = model[:len(model)-1]
			}

			if got, ok := d.PopBack(); ok != wantOK || got != want {
				t.Fatalf("step %d: PopBack() = (%d, %v), want (%d, %v)",
					i, got, ok, want, wantOK)
			}
		case op < 9:
			n := r.IntN(5) - 1
			k := min(max(n, 0), len(model))
			want := slices.Clone(model[:k])
			model = model[k:]

			if got := d.PopFrontN(n); !slices.Equal(got, want) {
				t.Fatalf("step %d: PopFrontN(%d) = %v, want %v",
					i, n, got, want)
			}
		case op < 10:
			n := r.IntN(5) - 1
			k := min(max(n, 0), len(model))
			want := slices.Clone(model[len(model)-k:])
			slices.Reverse(want)
			model = model[:len(model)-k]

			if got := d.PopBackN(n); !slices.Equal(got, want) {
				t.Fatalf("step %d: PopBackN(%d) = %v, want %v",
					i, n, got, want)
			}
		default:
			switch r.IntN(5) {
			case 0:
				d.Clear()
				model = nil
			case 1:
				d.Reset()
				model = nil
			case 2:
				d.Shrink()
			case 3:
				d.Clip()
			default:
				d.Grow(r.IntN(16))
			}
		}

		if got := d.Slice(); !slices.Equal(got, model) {
			t.Fatalf("step %d: Slice() = %v, want %v", i, got, model)
		}

		if got := d.Len(); got != len(model) {
			t.Fatalf("step %d: Len() = %d, want %d", i, got, len(model))
		}
	}

	want := slices.Clone(model)
	slices.Reverse(want)

	if got := slices.Collect(d.Backward()); !slices.Equal(got, want) {
		t.Errorf("Backward() = %v, want %v", got, want)
	}
}

// TestDeque_NilReceiver checks that every read-only and removal method treats
// a nil receiver as an empty deque.
func TestDeque_NilReceiver(t *testing.T) {
	var d *Deque[int]

	if got := d.Len(); got != 0 {
		t.Errorf("nil.Len() = %d, want 0", got)
	}

	if got := d.Cap(); got != 0 {
		t.Errorf("nil.Cap() = %d, want 0", got)
	}

	if got := d.IsEmpty(); !got {
		t.Errorf("nil.IsEmpty() = %v, want true", got)
	}

	if got, ok := d.Front(); ok || got != 0 {
		t.Errorf("nil.Front() = (%d, %v), want (0, false)", got, ok)
	}

	if got, ok := d.Back(); ok || got != 0 {
		t.Errorf("nil.Back() = (%d, %v), want (0, false)", got, ok)
	}

	count := 0
	for range d.All() {
		count++
	}

	if count != 0 {
		t.Errorf("nil.All() count = %d, want 0", count)
	}

	count = 0
	for range d.Backward() {
		count++
	}

	if count != 0 {
		t.Errorf("nil.Backward() count = %d, want 0", count)
	}

	c := d.Clone()
	if c == nil {
		t.Fatalf("nil.Clone() = nil, want a usable empty deque")
	}

	if got := c.Len(); got != 0 {
		t.Errorf("nil.Clone().Len() = %d, want 0", got)
	}

	c.PushBack(42)

	if got, ok := c.PopFront(); !ok || got != 42 {
		t.Errorf("PopFront() after PushBack(42) on nil.Clone() = (%d, %v), "+
			"want (42, true)", got, ok)
	}

	if got := d.Slice(); got != nil {
		t.Errorf("nil.Slice() = %v, want nil", got)
	}

	if got, ok := d.PopFront(); ok || got != 0 {
		t.Errorf("nil.PopFront() = (%d, %v), want (0, false)", got, ok)
	}

	if got, ok := d.PopBack(); ok || got != 0 {
		t.Errorf("nil.PopBack() = (%d, %v), want (0, false)", got, ok)
	}

	if got := d.PopFrontN(3); got != nil {
		t.Errorf("nil.PopFrontN(3) = %v, want nil", got)
	}

	if got := d.PopBackN(3); got != nil {
		t.Errorf("nil.PopBackN(3) = %v, want nil", got)
	}
}

// TestDeque_NilReceiverMutatorsPanic checks that every method that stores,
// clears, resets, or resizes data panics on a nil receiver.
func TestDeque_NilReceiverMutatorsPanic(t *testing.T) {
	tests := []struct {
		name string
		op   string
		call func(*Deque[int])
	}{
		{
			name: "PushFront",
			op:   "PushFront",
			call: func(d *Deque[int]) { d.PushFront(1) },
		},
		{
			name: "PushBack",
			op:   "PushBack",
			call: func(d *Deque[int]) { d.PushBack(1) },
		},
		{
			name: "PushFrontN",
			op:   "PushFrontN",
			call: func(d *Deque[int]) { d.PushFrontN(1, 2) },
		},
		{
			name: "PushFrontN with no arguments",
			op:   "PushFrontN",
			call: func(d *Deque[int]) { d.PushFrontN() },
		},
		{
			name: "PushBackN",
			op:   "PushBackN",
			call: func(d *Deque[int]) { d.PushBackN(1, 2) },
		},
		{
			name: "PushBackN with no arguments",
			op:   "PushBackN",
			call: func(d *Deque[int]) { d.PushBackN() },
		},
		{
			name: "Clear",
			op:   "Clear",
			call: func(d *Deque[int]) { d.Clear() },
		},
		{
			name: "Reset",
			op:   "Reset",
			call: func(d *Deque[int]) { d.Reset() },
		},
		{
			name: "Shrink",
			op:   "Shrink",
			call: func(d *Deque[int]) { d.Shrink() },
		},
		{
			name: "Clip",
			op:   "Clip",
			call: func(d *Deque[int]) { d.Clip() },
		},
		{
			name: "Grow",
			op:   "Grow",
			call: func(d *Deque[int]) { d.Grow(8) },
		},
		{
			name: "Grow zero",
			op:   "Grow",
			call: func(d *Deque[int]) { d.Grow(0) },
		},
		{
			name: "Grow negative",
			op:   "Grow",
			call: func(d *Deque[int]) { d.Grow(-1) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d *Deque[int]
			sharedtest.Panics(t, tt.name, "deque: "+tt.op+" on nil receiver",
				func() { tt.call(d) })
		})
	}
}

func TestDeque_ZeroValue(t *testing.T) {
	var d Deque[int]

	if got := d.Len(); got != 0 {
		t.Errorf("Len() on zero value = %d, want 0", got)
	}

	if got := d.Cap(); got != 0 {
		t.Errorf("Cap() on zero value = %d, want 0", got)
	}

	if got := d.IsEmpty(); !got {
		t.Errorf("IsEmpty() on zero value = %v, want true", got)
	}

	if got, ok := d.Front(); ok || got != 0 {
		t.Errorf("Front() on zero value = (%d, %v), want (0, false)", got, ok)
	}

	if got, ok := d.Back(); ok || got != 0 {
		t.Errorf("Back() on zero value = (%d, %v), want (0, false)", got, ok)
	}

	if got, ok := d.PopFront(); ok || got != 0 {
		t.Errorf("PopFront() on zero value = (%d, %v), want (0, false)",
			got, ok)
	}

	if got, ok := d.PopBack(); ok || got != 0 {
		t.Errorf("PopBack() on zero value = (%d, %v), want (0, false)",
			got, ok)
	}

	d.Clear()
	d.Reset()
	d.Shrink()
	d.Clip()
	d.Grow(4)

	if got := d.Cap(); got < 4 {
		t.Errorf("Cap() after Grow(4) = %d, want >= 4", got)
	}

	d.PushBackN(1, 2)

	if got, ok := d.PopFront(); !ok || got != 1 {
		t.Errorf("PopFront() after PushBackN(1, 2) on zero value = "+
			"(%d, %v), want (1, true)", got, ok)
	}
}
