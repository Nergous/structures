package ring

import (
	"math/rand/v2"
	"slices"
	"testing"
)

// assertZeroed fails the test unless every slot of data is nil. op names the
// method that must have zeroed the slots.
func assertZeroed(t *testing.T, data []*int, op string) {
	t.Helper()

	for i, got := range data {
		if got != nil {
			t.Errorf("data[%d] after %s() = %p, "+
				"want nil (%s must zero the slot)", i, op, got, op)
		}
	}
}

// contents returns the live elements of b in front-to-back order.
func contents[T any](b *Buffer[T]) []T {
	out := make([]T, b.Len())
	b.CopyTo(out)

	return out
}

// spare returns a buffer holding 1, 2, 3 at the start of a 16-slot backing
// array, so it has spare capacity and does not wrap.
func spare() Buffer[int] {
	b := WithCap[int](16)
	b.PushBackN([]int{1, 2, 3})

	return b
}

// wrapped returns a full buffer of capacity 4 holding 3, 4, 5, 6, whose
// elements wrap around the end of the backing array.
func wrapped() Buffer[int] {
	b := WithCap[int](4)
	b.PushBackN([]int{1, 2, 3, 4})
	b.PopFrontN(2)
	b.PushBackN([]int{5, 6})

	return b
}

func TestBuffer_WithCap(t *testing.T) {
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
			b := WithCap[int](tt.n)

			if got := b.Cap(); got != tt.want {
				t.Errorf("WithCap(%d).Cap() = %d, want %d", tt.n, got, tt.want)
			}

			b.PushBack(1)

			if got, ok := b.PopFront(); !ok || got != 1 {
				t.Errorf("PopFront() after PushBack(1) = (%d, %v), "+
					"want (1, true)", got, ok)
			}
		})
	}
}

func TestBuffer_Front(t *testing.T) {
	t.Run("returns first element", func(t *testing.T) {
		b := wrapped()

		if got, ok := b.Front(); !ok || got != 3 {
			t.Errorf("Front() = (%d, %v), want (3, true)", got, ok)
		}

		if got := b.Len(); got != 4 {
			t.Errorf("Len() after Front() = %d, want 4", got)
		}
	})

	t.Run("drained buffer returns (0, false)", func(t *testing.T) {
		b := wrapped()
		b.PopFrontN(4)

		if got, ok := b.Front(); ok || got != 0 {
			t.Errorf("Front() on empty = (%d, %v), want (0, false)", got, ok)
		}
	})
}

func TestBuffer_Back(t *testing.T) {
	t.Run("returns last element", func(t *testing.T) {
		b := wrapped()

		if got, ok := b.Back(); !ok || got != 6 {
			t.Errorf("Back() = (%d, %v), want (6, true)", got, ok)
		}

		if got := b.Len(); got != 4 {
			t.Errorf("Len() after Back() = %d, want 4", got)
		}
	})

	t.Run("drained buffer returns (0, false)", func(t *testing.T) {
		b := wrapped()
		b.PopBackN(4)

		if got, ok := b.Back(); ok || got != 0 {
			t.Errorf("Back() on empty = (%d, %v), want (0, false)", got, ok)
		}
	})
}

func TestBuffer_Segments(t *testing.T) {
	t.Run("contiguous elements return nil second", func(t *testing.T) {
		b := spare()

		first, second := b.Segments()

		if want := []int{1, 2, 3}; !slices.Equal(first, want) {
			t.Errorf("Segments() first = %v, want %v", first, want)
		}

		if second != nil {
			t.Errorf("Segments() second = %v, want nil", second)
		}
	})

	t.Run("wrapped elements split in two", func(t *testing.T) {
		b := wrapped()

		first, second := b.Segments()

		if got := b.Cap(); got != 4 {
			t.Errorf("Cap() after wrapping = %d, want 4 (no growth)", got)
		}

		if want := []int{3, 4}; !slices.Equal(first, want) {
			t.Errorf("Segments() first = %v, want %v", first, want)
		}

		if want := []int{5, 6}; !slices.Equal(second, want) {
			t.Errorf("Segments() second = %v, want %v", second, want)
		}
	})

	t.Run("empty buffer returns nil slices", func(t *testing.T) {
		b := WithCap[int](4)

		if first, second := b.Segments(); first != nil || second != nil {
			t.Errorf("Segments() on empty = (%v, %v), want (nil, nil)",
				first, second)
		}
	})
}

func TestBuffer_CopyTo(t *testing.T) {
	t.Run("copies wrapped elements in order", func(t *testing.T) {
		b := wrapped()
		dst := make([]int, 4)

		if got := b.CopyTo(dst); got != 4 {
			t.Errorf("CopyTo(dst) = %d, want 4", got)
		}

		if want := []int{3, 4, 5, 6}; !slices.Equal(dst, want) {
			t.Errorf("dst after CopyTo(dst) = %v, want %v", dst, want)
		}
	})

	t.Run("short dst receives a prefix", func(t *testing.T) {
		b := wrapped()
		dst := make([]int, 3)

		if got := b.CopyTo(dst); got != 3 {
			t.Errorf("CopyTo(dst) = %d, want 3", got)
		}

		if want := []int{3, 4, 5}; !slices.Equal(dst, want) {
			t.Errorf("dst after CopyTo(dst) = %v, want %v", dst, want)
		}
	})
}

func TestBuffer_Clone(t *testing.T) {
	t.Run("copies wrapped elements to exact size", func(t *testing.T) {
		b := wrapped()

		c := b.Clone()

		want := []int{3, 4, 5, 6}
		if got := contents(&c); !slices.Equal(got, want) {
			t.Fatalf("contents(c) = %v, want %v", got, want)
		}

		if got, want := c.Cap(), c.Len(); got != want {
			t.Errorf("Cap() of clone = %d, want Len() = %d", got, want)
		}
	})

	t.Run("clone is independent", func(t *testing.T) {
		b := wrapped()
		c := b.Clone()

		c.PopFront()
		c.PushBack(9)

		want := []int{3, 4, 5, 6}
		if got := contents(&b); !slices.Equal(got, want) {
			t.Errorf("contents(b) after mutating clone = %v, want %v",
				got, want)
		}
	})

	t.Run("empty buffer clones to empty", func(t *testing.T) {
		b := WithCap[int](8)

		c := b.Clone()

		if got := c.Len(); got != 0 {
			t.Errorf("Len() of clone = %d, want 0", got)
		}

		if got := c.Cap(); got != 0 {
			t.Errorf("Cap() of clone = %d, want 0", got)
		}
	})
}

func TestBuffer_PushFront(t *testing.T) {
	t.Run("prepends at the front", func(t *testing.T) {
		var b Buffer[int]
		b.PushFront(3)
		b.PushFront(2)
		b.PushFront(1)

		if got, want := contents(&b), []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("contents(b) = %v, want %v", got, want)
		}

		if got, ok := b.Front(); !ok || got != 1 {
			t.Errorf("Front() = (%d, %v), want (1, true)", got, ok)
		}
	})

	t.Run("wraps head below index 0", func(t *testing.T) {
		b := WithCap[int](4)
		b.PushBack(1)
		b.PushFront(0)

		if got, want := contents(&b), []int{0, 1}; !slices.Equal(got, want) {
			t.Errorf("contents(b) = %v, want %v", got, want)
		}

		if got := b.Cap(); got != 4 {
			t.Errorf("Cap() after PushFront(0) = %d, want 4 (no growth)", got)
		}
	})
}

func TestBuffer_PushBack(t *testing.T) {
	t.Run("appends at the back", func(t *testing.T) {
		var b Buffer[int]
		b.PushBack(1)
		b.PushBack(2)
		b.PushBack(3)

		if got, want := contents(&b), []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("contents(b) = %v, want %v", got, want)
		}

		if got, ok := b.Back(); !ok || got != 3 {
			t.Errorf("Back() = (%d, %v), want (3, true)", got, ok)
		}
	})

	t.Run("grows by doubling from one", func(t *testing.T) {
		var b Buffer[int]
		var got []int
		for i := range 5 {
			b.PushBack(i)
			if len(got) == 0 || got[len(got)-1] != b.Cap() {
				got = append(got, b.Cap())
			}
		}

		if want := []int{1, 2, 4, 8}; !slices.Equal(got, want) {
			t.Errorf("Cap() sequence = %v, want %v", got, want)
		}
	})

	t.Run("growth keeps wrapped order", func(t *testing.T) {
		b := wrapped()
		b.PushBack(7)

		want := []int{3, 4, 5, 6, 7}
		if got := contents(&b); !slices.Equal(got, want) {
			t.Errorf("contents(b) = %v, want %v", got, want)
		}

		if got := b.Cap(); got != 8 {
			t.Errorf("Cap() after PushBack(7) = %d, want 8", got)
		}
	})
}

func TestBuffer_PushFrontN(t *testing.T) {
	t.Run("PushFrontN makes first argument the new front", func(t *testing.T) {
		var b Buffer[int]
		b.PushBack(9)
		b.PushFrontN([]int{1, 2, 3})

		want := []int{1, 2, 3, 9}
		if got := contents(&b); !slices.Equal(got, want) {
			t.Errorf("contents(b) = %v, want %v", got, want)
		}
	})

	t.Run("PushFrontN wraps head below index 0", func(t *testing.T) {
		b := WithCap[int](4)
		b.PushBackN([]int{1, 2, 3})
		b.PopFront()
		b.PushFrontN([]int{7, 8})

		want := []int{7, 8, 2, 3}
		if got := contents(&b); !slices.Equal(got, want) {
			t.Errorf("contents(b) = %v, want %v", got, want)
		}
	})

	t.Run("empty argument list is a no-op", func(t *testing.T) {
		var b Buffer[int]
		b.PushFrontN(nil)

		if got := b.Len(); got != 0 {
			t.Errorf("Len() after PushFrontN(nil) = %d, want 0", got)
		}

		if got := b.Cap(); got != 0 {
			t.Errorf("Cap() after PushFrontN(nil) = %d, want 0", got)
		}
	})
}

func TestBuffer_PushBackN(t *testing.T) {
	t.Run("PushBackN keeps argument order", func(t *testing.T) {
		var b Buffer[int]
		b.PushBack(0)
		b.PushBackN([]int{1, 2, 3})

		want := []int{0, 1, 2, 3}
		if got := contents(&b); !slices.Equal(got, want) {
			t.Errorf("contents(b) = %v, want %v", got, want)
		}
	})

	t.Run("empty argument list is a no-op", func(t *testing.T) {
		var b Buffer[int]
		b.PushBackN(nil)

		if got := b.Len(); got != 0 {
			t.Errorf("Len() after PushBackN(nil) = %d, want 0", got)
		}

		if got := b.Cap(); got != 0 {
			t.Errorf("Cap() after PushBackN(nil) = %d, want 0", got)
		}
	})
}

func TestBuffer_PopFront(t *testing.T) {
	t.Run("removes the front element", func(t *testing.T) {
		b := wrapped()

		if got, ok := b.PopFront(); !ok || got != 3 {
			t.Errorf("PopFront() = (%d, %v), want (3, true)", got, ok)
		}

		if got, want := contents(&b), []int{4, 5, 6}; !slices.Equal(got, want) {
			t.Errorf("contents(b) after PopFront() = %v, want %v", got, want)
		}
	})

	t.Run("drained buffer returns (0, false)", func(t *testing.T) {
		b := WithCap[int](2)
		b.PushBack(1)
		b.PopFront()

		if got, ok := b.PopFront(); ok || got != 0 {
			t.Errorf("PopFront() on empty = (%d, %v), want (0, false)", got, ok)
		}
	})

	t.Run("zeroes vacated slot", func(t *testing.T) {
		x := 1
		b := WithCap[*int](2)
		b.PushBack(&x)

		b.PopFront()

		assertZeroed(t, b.Data(), "PopFront")
	})
}

func TestBuffer_PopBack(t *testing.T) {
	t.Run("removes the back element", func(t *testing.T) {
		b := wrapped()

		if got, ok := b.PopBack(); !ok || got != 6 {
			t.Errorf("PopBack() = (%d, %v), want (6, true)", got, ok)
		}

		if got, want := contents(&b), []int{3, 4, 5}; !slices.Equal(got, want) {
			t.Errorf("contents(b) after PopBack() = %v, want %v", got, want)
		}
	})

	t.Run("drained buffer returns (0, false)", func(t *testing.T) {
		b := WithCap[int](2)
		b.PushBack(1)
		b.PopBack()

		if got, ok := b.PopBack(); ok || got != 0 {
			t.Errorf("PopBack() on empty = (%d, %v), want (0, false)", got, ok)
		}
	})

	t.Run("zeroes vacated slot", func(t *testing.T) {
		x := 1
		b := WithCap[*int](2)
		b.PushBack(&x)

		b.PopBack()

		assertZeroed(t, b.Data(), "PopBack")
	})
}

func TestBuffer_PopFrontN(t *testing.T) {
	t.Run("returns front elements in order", func(t *testing.T) {
		var b Buffer[int]
		b.PushBackN([]int{1, 2, 3, 4, 5})

		if got, want := b.PopFrontN(2), []int{1, 2}; !slices.Equal(got, want) {
			t.Errorf("PopFrontN(2) = %v, want %v", got, want)
		}

		if got, want := contents(&b), []int{3, 4, 5}; !slices.Equal(got, want) {
			t.Errorf("contents(b) after PopFrontN(2) = %v, want %v", got, want)
		}
	})

	t.Run("n greater than Len drains the buffer", func(t *testing.T) {
		b := spare()

		want := []int{1, 2, 3}
		if got := b.PopFrontN(10); !slices.Equal(got, want) {
			t.Errorf("PopFrontN(10) = %v, want %v", got, want)
		}

		if got := b.Len(); got != 0 {
			t.Errorf("Len() after PopFrontN(10) = %d, want 0", got)
		}
	})

	t.Run("non-positive n returns nil", func(t *testing.T) {
		b := spare()

		if got := b.PopFrontN(0); got != nil {
			t.Errorf("PopFrontN(0) = %v, want nil", got)
		}

		if got := b.PopFrontN(-1); got != nil {
			t.Errorf("PopFrontN(-1) = %v, want nil", got)
		}

		if got := b.Len(); got != 3 {
			t.Errorf("Len() after PopFrontN(0), PopFrontN(-1) = %d, want 3",
				got)
		}
	})

	t.Run("empty buffer returns nil", func(t *testing.T) {
		b := WithCap[int](4)

		if got := b.PopFrontN(1); got != nil {
			t.Errorf("PopFrontN(1) on empty = %v, want nil", got)
		}
	})

	t.Run("zeroes vacated slots", func(t *testing.T) {
		x, y := 1, 2
		b := WithCap[*int](2)
		b.PushBackN([]*int{&x, &y})

		b.PopFrontN(2)

		assertZeroed(t, b.Data(), "PopFrontN")
	})
}

func TestBuffer_PopBackN(t *testing.T) {
	t.Run("returns back elements in removal order", func(t *testing.T) {
		var b Buffer[int]
		b.PushBackN([]int{1, 2, 3, 4, 5})

		if got, want := b.PopBackN(2), []int{5, 4}; !slices.Equal(got, want) {
			t.Errorf("PopBackN(2) = %v, want %v", got, want)
		}

		if got, want := contents(&b), []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("contents(b) after PopBackN(2) = %v, want %v", got, want)
		}
	})

	t.Run("n greater than Len drains the buffer", func(t *testing.T) {
		b := spare()

		want := []int{3, 2, 1}
		if got := b.PopBackN(10); !slices.Equal(got, want) {
			t.Errorf("PopBackN(10) = %v, want %v", got, want)
		}

		if got := b.Len(); got != 0 {
			t.Errorf("Len() after PopBackN(10) = %d, want 0", got)
		}
	})

	t.Run("non-positive n returns nil", func(t *testing.T) {
		b := spare()

		if got := b.PopBackN(0); got != nil {
			t.Errorf("PopBackN(0) = %v, want nil", got)
		}

		if got := b.PopBackN(-1); got != nil {
			t.Errorf("PopBackN(-1) = %v, want nil", got)
		}

		if got := b.Len(); got != 3 {
			t.Errorf("Len() after PopBackN(0), PopBackN(-1) = %d, want 3", got)
		}
	})

	t.Run("empty buffer returns nil", func(t *testing.T) {
		b := WithCap[int](4)

		if got := b.PopBackN(1); got != nil {
			t.Errorf("PopBackN(1) on empty = %v, want nil", got)
		}
	})

	t.Run("zeroes vacated slots", func(t *testing.T) {
		x, y := 1, 2
		b := WithCap[*int](2)
		b.PushBackN([]*int{&x, &y})

		b.PopBackN(2)

		assertZeroed(t, b.Data(), "PopBackN")
	})
}

func TestBuffer_Clear(t *testing.T) {
	b := spare()

	b.Clear()

	if got := b.Len(); got != 0 {
		t.Errorf("Len() after Clear() = %d, want 0", got)
	}

	if got := b.Cap(); got != 0 {
		t.Errorf("Cap() after Clear() = %d, want 0", got)
	}

	b.PushBack(1)

	if got, ok := b.Front(); !ok || got != 1 {
		t.Errorf("Front() after Clear(), PushBack(1) = (%d, %v), "+
			"want (1, true)", got, ok)
	}
}

func TestBuffer_Reset(t *testing.T) {
	x := 1
	b := WithCap[*int](4)
	b.PushBackN([]*int{&x, &x})
	data := b.Data()

	b.Reset()

	if got := b.Len(); got != 0 {
		t.Errorf("Len() after Reset() = %d, want 0", got)
	}

	if got := b.Cap(); got != 4 {
		t.Errorf("Cap() after Reset() = %d, want 4", got)
	}

	assertZeroed(t, data, "Reset")

	b.PushBack(&x)

	if got, ok := b.Front(); !ok || got != &x {
		t.Errorf("Front() after Reset(), PushBack(&x) = (%p, %v), "+
			"want (%p, true)", got, ok, &x)
	}
}

func TestBuffer_Shrink(t *testing.T) {
	t.Run("Shrink always copies to exact size", func(t *testing.T) {
		b := spare()
		old := &b.Data()[0]

		b.Shrink()

		if got, want := b.Cap(), b.Len(); got != want {
			t.Errorf("Cap() after Shrink() = %d, want Len() = %d", got, want)
		}

		if got := &b.Data()[0]; got == old {
			t.Errorf("&data[0] after Shrink() = %p, want new backing array",
				got)
		}

		if got, want := contents(&b), []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("contents(b) after Shrink() = %v, want %v", got, want)
		}
	})

	t.Run("Shrink keeps wrapped order", func(t *testing.T) {
		b := wrapped()
		b.PopBack()

		b.Shrink()

		if got, want := b.Cap(), b.Len(); got != want {
			t.Errorf("Cap() after Shrink() = %d, want Len() = %d", got, want)
		}

		if got, want := contents(&b), []int{3, 4, 5}; !slices.Equal(got, want) {
			t.Errorf("contents(b) after Shrink() = %v, want %v", got, want)
		}
	})

	t.Run("empty buffer drops the array", func(t *testing.T) {
		b := WithCap[int](8)

		b.Shrink()

		if got := b.Cap(); got != 0 {
			t.Errorf("Cap() after Shrink() on empty = %d, want 0", got)
		}
	})

	t.Run("right-sized buffer is a no-op", func(t *testing.T) {
		b := WithCap[int](2)
		b.PushBackN([]int{1, 2})
		old := &b.Data()[0]

		b.Shrink()

		if got := &b.Data()[0]; got != old {
			t.Errorf("&data[0] after Shrink() = %p, want %p (no reallocation)",
				got, old)
		}
	})
}

func TestBuffer_Clip(t *testing.T) {
	t.Run("Clip reslices contiguous elements", func(t *testing.T) {
		b := spare()
		old := &b.Data()[0]

		b.Clip()

		if got, want := b.Cap(), b.Len(); got != want {
			t.Errorf("Cap() after Clip() = %d, want Len() = %d", got, want)
		}

		if got := &b.Data()[0]; got != old {
			t.Errorf("&data[0] after Clip() = %p, want %p (no copy)", got, old)
		}

		if got, want := contents(&b), []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("contents(b) after Clip() = %v, want %v", got, want)
		}
	})

	t.Run("Clip copies wrapped elements", func(t *testing.T) {
		b := wrapped()

		b.Clip()

		if got, want := b.Cap(), b.Len(); got != want {
			t.Errorf("Cap() after Clip() = %d, want Len() = %d", got, want)
		}

		want := []int{3, 4, 5, 6}
		if got := contents(&b); !slices.Equal(got, want) {
			t.Errorf("contents(b) after Clip() = %v, want %v", got, want)
		}
	})

	t.Run("empty buffer drops the array", func(t *testing.T) {
		b := WithCap[int](8)

		b.Clip()

		if got := b.Cap(); got != 0 {
			t.Errorf("Cap() after Clip() on empty = %d, want 0", got)
		}
	})

	t.Run("right-sized buffer is a no-op", func(t *testing.T) {
		b := WithCap[int](2)
		b.PushBackN([]int{1, 2})
		old := &b.Data()[0]

		b.Clip()

		if got := &b.Data()[0]; got != old {
			t.Errorf("&data[0] after Clip() = %p, want %p (no reallocation)",
				got, old)
		}
	})
}

func TestBuffer_Grow(t *testing.T) {
	t.Run("Grow reallocates for n more elements", func(t *testing.T) {
		var b Buffer[int]
		b.PushBackN([]int{1, 2, 3})

		b.Grow(100)

		if got := b.Cap(); got < 103 {
			t.Errorf("Cap() after Grow(100) = %d, want >= 103", got)
		}

		if got, want := contents(&b), []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("contents(b) after Grow(100) = %v, want %v", got, want)
		}
	})

	t.Run("non-positive n is a no-op", func(t *testing.T) {
		b := spare()
		capBefore := b.Cap()

		b.Grow(0)
		b.Grow(-1)

		if got := b.Cap(); got != capBefore {
			t.Errorf("Cap() after Grow(0), Grow(-1) = %d, want %d",
				got, capBefore)
		}
	})

	t.Run("sufficient capacity is a no-op", func(t *testing.T) {
		b := spare()
		old := &b.Data()[0]

		b.Grow(13)

		if got := b.Cap(); got != 16 {
			t.Errorf("Cap() after Grow(13) = %d, want 16", got)
		}

		if got := &b.Data()[0]; got != old {
			t.Errorf("&data[0] after Grow(13) = %p, want %p (no reallocation)",
				got, old)
		}
	})
}

func TestNextCap(t *testing.T) {
	tests := []struct {
		name    string
		current int
		need    int
		want    int
	}{
		{
			name:    "zero starts from one",
			current: 0,
			need:    1,
			want:    1,
		},
		{
			name:    "zero doubles to fit",
			current: 0,
			need:    3,
			want:    4,
		},
		{
			name:    "negative starts from one",
			current: -3,
			need:    2,
			want:    2,
		},
		{
			name:    "doubles from one",
			current: 1,
			need:    2,
			want:    2,
		},
		{
			name:    "doubles below smallCap",
			current: 4,
			need:    5,
			want:    8,
		},
		{
			name:    "doubles up to smallCap",
			current: smallCap / 2,
			need:    smallCap/2 + 1,
			want:    smallCap,
		},
		{
			name:    "grows by a quarter at smallCap",
			current: smallCap,
			need:    smallCap + 1,
			want:    smallCap + smallCap/4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nextCap(tt.current, tt.need); got != tt.want {
				t.Errorf("nextCap(%d, %d) = %d, want %d",
					tt.current, tt.need, got, tt.want)
			}
		})
	}
}

// TestBuffer_RandomOps runs a seeded random sequence of operations against a
// slice model and checks that both agree after every step.
func TestBuffer_RandomOps(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	var b Buffer[int]
	var model []int

	for i := range 5000 {
		switch op := r.IntN(9); {
		case op < 2:
			b.PushBack(i)
			model = append(model, i)
		case op < 4:
			b.PushFront(i)
			model = append([]int{i}, model...)
		case op < 5:
			got, ok := b.PopFront()
			if len(model) == 0 {
				if ok || got != 0 {
					t.Fatalf("step %d: PopFront() on empty = (%d, %v), "+
						"want (0, false)", i, got, ok)
				}

				break
			}

			want := model[0]
			model = model[1:]
			if !ok || got != want {
				t.Fatalf("step %d: PopFront() = (%d, %v), want (%d, true)",
					i, got, ok, want)
			}
		case op < 6:
			got, ok := b.PopBack()
			if len(model) == 0 {
				if ok || got != 0 {
					t.Fatalf("step %d: PopBack() on empty = (%d, %v), "+
						"want (0, false)", i, got, ok)
				}

				break
			}

			want := model[len(model)-1]
			model = model[:len(model)-1]
			if !ok || got != want {
				t.Fatalf("step %d: PopBack() = (%d, %v), want (%d, true)",
					i, got, ok, want)
			}
		case op < 7:
			vs := []int{i, i + 1, i + 2}
			b.PushFrontN(vs)
			model = append(slices.Clone(vs), model...)
		case op < 8:
			b.PushBackN([]int{i, i + 1})
			model = append(model, i, i+1)
		default:
			switch r.IntN(5) {
			case 0:
				b.Clear()
				model = nil
			case 1:
				b.Reset()
				model = nil
			case 2:
				b.Shrink()
			case 3:
				b.Clip()
			default:
				b.Grow(r.IntN(16))
			}
		}

		if got := contents(&b); !slices.Equal(got, model) {
			t.Fatalf("step %d: contents(b) = %v, want %v", i, got, model)
		}

		if got := b.Len(); got != len(model) {
			t.Fatalf("step %d: Len() = %d, want %d", i, got, len(model))
		}
	}
}

func TestBuffer_ZeroValue(t *testing.T) {
	var b Buffer[int]

	if got := b.Len(); got != 0 {
		t.Errorf("Len() on zero value = %d, want 0", got)
	}

	if got := b.Cap(); got != 0 {
		t.Errorf("Cap() on zero value = %d, want 0", got)
	}

	if got, ok := b.Front(); ok || got != 0 {
		t.Errorf("Front() on zero value = (%d, %v), want (0, false)", got, ok)
	}

	if got, ok := b.Back(); ok || got != 0 {
		t.Errorf("Back() on zero value = (%d, %v), want (0, false)", got, ok)
	}

	if first, second := b.Segments(); first != nil || second != nil {
		t.Errorf("Segments() on zero value = (%v, %v), want (nil, nil)",
			first, second)
	}

	if got, ok := b.PopFront(); ok || got != 0 {
		t.Errorf("PopFront() on zero value = (%d, %v), want (0, false)",
			got, ok)
	}

	if got, ok := b.PopBack(); ok || got != 0 {
		t.Errorf("PopBack() on zero value = (%d, %v), want (0, false)", got, ok)
	}

	if got := b.PopFrontN(1); got != nil {
		t.Errorf("PopFrontN(1) on zero value = %v, want nil", got)
	}

	if got := b.PopBackN(1); got != nil {
		t.Errorf("PopBackN(1) on zero value = %v, want nil", got)
	}

	b.Clear()
	b.Reset()
	b.Shrink()
	b.Clip()
	b.Grow(4)

	if got := b.Cap(); got < 4 {
		t.Errorf("Cap() after Grow(4) = %d, want >= 4", got)
	}

	b.PushBack(1)

	if got, ok := b.PopFront(); !ok || got != 1 {
		t.Errorf("PopFront() after PushBack(1) on zero value = (%d, %v), "+
			"want (1, true)", got, ok)
	}
}
