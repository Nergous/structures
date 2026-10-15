package ring

import (
	"slices"
	"testing"
)

func contents[T any](b *Buffer[T]) []T {
	out := make([]T, b.Len())
	b.CopyTo(out)

	return out
}

func TestBuffer_Zero(t *testing.T) {
	var b Buffer[int]

	if got := b.Len(); got != 0 {
		t.Errorf("Len() on zero value = %d, want 0", got)
	}

	if got := b.Cap(); got != 0 {
		t.Errorf("Cap() on zero value = %d, want 0", got)
	}

	if got, ok := b.Front(); ok {
		t.Errorf("Front() on empty: got %d, ok = true, want ok = false", got)
	}

	if got, ok := b.Back(); ok {
		t.Errorf("Back() on empty: got %d, ok = true, want ok = false", got)
	}

	if got, ok := b.PopFront(); ok {
		t.Errorf("PopFront() on empty: got %d, ok = true, want ok = false", got)
	}

	if got, ok := b.PopBack(); ok {
		t.Errorf("PopBack() on empty: got %d, ok = true, want ok = false", got)
	}

	b.PushBack(1)

	if got, ok := b.Front(); !ok || got != 1 {
		t.Errorf("Front() after PushBack(1) = (%d, %v), want (1, true)", got, ok)
	}
}

func TestBuffer_WithCap(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want int
	}{
		{
			name: "negative clamps to zero",
			n:    -5,
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
		})
	}
}

func TestBuffer_PushPopBothEnds(t *testing.T) {
	var b Buffer[int]
	b.PushBack(2)
	b.PushBack(3)
	b.PushFront(1)
	b.PushFront(0)

	if got, want := contents(&b), []int{0, 1, 2, 3}; !slices.Equal(got, want) {
		t.Fatalf("contents(b) = %v, want %v", got, want)
	}

	if got, _ := b.Front(); got != 0 {
		t.Errorf("Front() = %d, want 0", got)
	}

	if got, _ := b.Back(); got != 3 {
		t.Errorf("Back() = %d, want 3", got)
	}

	if got, _ := b.PopBack(); got != 3 {
		t.Errorf("PopBack() = %d, want 3", got)
	}

	if got, _ := b.PopFront(); got != 0 {
		t.Errorf("PopFront() = %d, want 0", got)
	}

	if got, want := contents(&b), []int{1, 2}; !slices.Equal(got, want) {
		t.Errorf("contents(b) after PopBack(), PopFront() = %v, want %v", got, want)
	}
}

func TestBuffer_WrapAround(t *testing.T) {
	b := WithCap[int](4)
	for i := 1; i <= 4; i++ {
		b.PushBack(i)
	}

	b.PopFront()
	b.PopFront()
	b.PushBack(5)
	b.PushBack(6) // wraps without growing

	if got := b.Cap(); got != 4 {
		t.Fatalf("Cap() = %d, want 4 (no growth)", got)
	}

	first, second := b.Segments()
	if len(second) == 0 {
		t.Errorf("Segments() second = %v, want wrapped elements", second)
	}

	if got, want := slices.Concat(first, second), []int{3, 4, 5, 6}; !slices.Equal(got, want) {
		t.Errorf("Segments() = %v, want %v", got, want)
	}
}

func TestBuffer_PushN(t *testing.T) {
	t.Run("back keeps argument order", func(t *testing.T) {
		var b Buffer[int]
		b.PushBack(0)
		b.PushBackN([]int{1, 2, 3})

		if got, want := contents(&b), []int{0, 1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("contents(b) = %v, want %v", got, want)
		}
	})

	t.Run("front makes first argument the new front", func(t *testing.T) {
		var b Buffer[int]
		b.PushBack(9)
		b.PushFrontN([]int{1, 2, 3})

		if got, want := contents(&b), []int{1, 2, 3, 9}; !slices.Equal(got, want) {
			t.Errorf("contents(b) = %v, want %v", got, want)
		}
	})

	t.Run("front on wrapped buffer", func(t *testing.T) {
		b := WithCap[int](4)
		b.PushBackN([]int{1, 2, 3})
		b.PopFront()
		b.PushFrontN([]int{7, 8})

		if got, want := contents(&b), []int{7, 8, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("contents(b) = %v, want %v", got, want)
		}
	})

	t.Run("empty argument list is a no-op", func(t *testing.T) {
		var b Buffer[int]
		b.PushBackN(nil)
		b.PushFrontN(nil)

		if got := b.Len(); got != 0 {
			t.Errorf("Len() after PushBackN(nil)/PushFrontN(nil) = %d, want 0", got)
		}

		if got := b.Cap(); got != 0 {
			t.Errorf("Cap() after PushBackN(nil)/PushFrontN(nil) = %d, want 0", got)
		}
	})
}

func TestBuffer_PopN(t *testing.T) {
	var b Buffer[int]
	b.PushBackN([]int{1, 2, 3, 4, 5})

	if got, want := b.PopFrontN(2), []int{1, 2}; !slices.Equal(got, want) {
		t.Errorf("PopFrontN(2) = %v, want %v", got, want)
	}

	if got, want := b.PopBackN(2), []int{5, 4}; !slices.Equal(got, want) {
		t.Errorf("PopBackN(2) = %v, want %v", got, want)
	}

	if got := b.PopFrontN(0); got != nil {
		t.Errorf("PopFrontN(0) = %v, want nil", got)
	}

	if got := b.PopBackN(-1); got != nil {
		t.Errorf("PopBackN(-1) = %v, want nil", got)
	}

	if got, want := b.PopFrontN(10), []int{3}; !slices.Equal(got, want) {
		t.Errorf("PopFrontN(10) = %v, want %v", got, want)
	}

	if got := b.PopBackN(1); got != nil {
		t.Errorf("PopBackN(1) on empty = %v, want nil", got)
	}
}

func TestBuffer_PopZeroesSlot(t *testing.T) {
	x, y := 1, 2
	b := WithCap[*int](2)
	b.PushBack(&x)
	b.PushBack(&y)
	backing := b.Data()

	b.PopFront()
	b.PopBack()

	for i, got := range backing {
		if got != nil {
			t.Errorf("data[%d] = %p, want nil (Pop must zero the slot)", i, got)
		}
	}
}

func TestBuffer_Growth(t *testing.T) {
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
			name:    "doubles from one",
			current: 1,
			need:    2,
			want:    2,
		},
		{
			name:    "doubles below 1024",
			current: 4,
			need:    5,
			want:    8,
		},
		{
			name:    "doubles up to 1024",
			current: 512,
			need:    513,
			want:    1024,
		},
		{
			name:    "grows by a quarter above 1024",
			current: 1024,
			need:    1025,
			want:    1280,
		},
		{
			name:    "negative starts from one",
			current: -3,
			need:    2,
			want:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nextCap(tt.current, tt.need); got != tt.want {
				t.Errorf("nextCap(%d, %d) = %d, want %d", tt.current, tt.need, got, tt.want)
			}
		})
	}
}

func TestBuffer_Grow(t *testing.T) {
	var b Buffer[int]
	b.PushBackN([]int{1, 2, 3})
	b.Grow(0)
	b.Grow(-4)
	capBefore := b.Cap()

	b.Grow(100)

	if got := b.Cap(); got < 103 {
		t.Errorf("Cap() after Grow(100) = %d, want >= 103", got)
	}

	if got := b.Cap(); got == capBefore {
		t.Errorf("Cap() after Grow(100) = %d, want != %d (must reallocate)", got, capBefore)
	}

	if got, want := contents(&b), []int{1, 2, 3}; !slices.Equal(got, want) {
		t.Errorf("contents(b) after Grow(100) = %v, want %v", got, want)
	}
}

func TestBuffer_ShrinkAndClip(t *testing.T) {
	filled := func() Buffer[int] {
		b := WithCap[int](16)
		b.PushBackN([]int{1, 2, 3})

		return b
	}

	wrapped := func() Buffer[int] {
		b := WithCap[int](4)
		b.PushBackN([]int{1, 2, 3, 4})
		b.PopFrontN(2)
		b.PushBackN([]int{5, 6})

		return b
	}

	t.Run("shrink always copies to exact size", func(t *testing.T) {
		b := filled()
		old := &b.Data()[0]

		b.Shrink()

		if got := b.Cap(); got != 3 {
			t.Errorf("Cap() after Shrink() = %d, want 3", got)
		}

		if got := &b.Data()[0]; got == old {
			t.Errorf("&data[0] after Shrink() = %p, want new backing array (not %p)", got, old)
		}

		if got, want := contents(&b), []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("contents(b) after Shrink() = %v, want %v", got, want)
		}
	})

	t.Run("clip reslices contiguous elements", func(t *testing.T) {
		b := filled()
		old := &b.Data()[0]

		b.Clip()

		if got := b.Cap(); got != 3 {
			t.Errorf("Cap() after Clip() = %d, want 3", got)
		}

		if got := &b.Data()[0]; got != old {
			t.Errorf("&data[0] after Clip() = %p, want %p (same backing array, no copy)", got, old)
		}
	})

	t.Run("clip copies wrapped elements", func(t *testing.T) {
		b := wrapped()

		b.Clip()

		if got := b.Cap(); got != 4 {
			t.Errorf("Cap() after Clip() = %d, want 4", got)
		}

		if got, want := contents(&b), []int{3, 4, 5, 6}; !slices.Equal(got, want) {
			t.Errorf("contents(b) after Clip() = %v, want %v", got, want)
		}
	})

	t.Run("shrink keeps wrapped order", func(t *testing.T) {
		b := wrapped()
		b.PopBack()

		b.Shrink()

		if got, want := contents(&b), []int{3, 4, 5}; !slices.Equal(got, want) {
			t.Errorf("contents(b) after Shrink() = %v, want %v", got, want)
		}

		if got := b.Cap(); got != 3 {
			t.Errorf("Cap() after Shrink() = %d, want 3", got)
		}
	})

	t.Run("empty drops the array", func(t *testing.T) {
		tests := []struct {
			name string
			call func(*Buffer[int])
		}{
			{
				name: "Shrink",
				call: (*Buffer[int]).Shrink,
			},
			{
				name: "Clip",
				call: (*Buffer[int]).Clip,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				b := WithCap[int](8)
				tt.call(&b)

				if got := b.Cap(); got != 0 {
					t.Errorf("Cap() after %s() on empty = %d, want 0", tt.name, got)
				}
			})
		}
	})

	t.Run("already right-sized is a no-op", func(t *testing.T) {
		b := WithCap[int](2)
		b.PushBackN([]int{1, 2})
		old := &b.Data()[0]

		b.Shrink()
		b.Clip()

		if got := &b.Data()[0]; got != old {
			t.Errorf("&data[0] after Shrink(), Clip() = %p, want %p (right-sized buffer must not reallocate)", got, old)
		}
	})
}

func TestBuffer_ClearAndReset(t *testing.T) {
	x := 1
	b := WithCap[*int](4)
	b.PushBack(&x)
	b.PushBack(&x)
	backing := b.Data()

	b.Reset()

	if got := b.Len(); got != 0 {
		t.Errorf("Len() after Reset() = %d, want 0", got)
	}

	if got := b.Cap(); got != 4 {
		t.Errorf("Cap() after Reset() = %d, want 4", got)
	}

	for i, got := range backing {
		if got != nil {
			t.Errorf("data[%d] after Reset() = %p, want nil", i, got)
		}
	}

	b.PushBack(&x)
	b.Clear()

	if got := b.Len(); got != 0 {
		t.Errorf("Len() after Clear() = %d, want 0", got)
	}

	if got := b.Cap(); got != 0 {
		t.Errorf("Cap() after Clear() = %d, want 0", got)
	}

	b.PushBack(&x)

	if got := b.Len(); got != 1 {
		t.Errorf("Len() after PushBack() on cleared = %d, want 1 (buffer must stay usable)", got)
	}
}

func TestBuffer_Clone(t *testing.T) {
	b := WithCap[int](4)
	b.PushBackN([]int{1, 2, 3, 4})
	b.PopFrontN(2)
	b.PushBack(5) // wrapped

	c := b.Clone()
	if got, want := contents(&c), []int{3, 4, 5}; !slices.Equal(got, want) {
		t.Fatalf("contents(Clone()) = %v, want %v", got, want)
	}

	if got := c.Cap(); got != 3 {
		t.Errorf("Clone().Cap() = %d, want 3", got)
	}

	c.PopFront()

	if got := b.Len(); got != 3 {
		t.Errorf("original Len() = %d, want 3 (independent of clone)", got)
	}
}

func TestBuffer_Randomized(t *testing.T) {
	// Compare against a plain slice model through a deterministic mix of ops.
	var b Buffer[int]
	var model []int
	seed := uint32(1)
	next := func(n int) int {
		seed = seed*1664525 + 1013904223

		return int(seed>>8) % n
	}

	for i := range 5000 {
		switch next(8) {
		case 0, 1:
			b.PushBack(i)
			model = append(model, i)
		case 2, 3:
			b.PushFront(i)
			model = append([]int{i}, model...)
		case 4:
			got, ok := b.PopFront()
			if want := len(model) > 0; ok != want {
				t.Fatalf("PopFront() #%d: ok = %v, want %v", i, ok, want)
			}

			if ok {
				if want := model[0]; got != want {
					t.Fatalf("PopFront() #%d = %d, want %d", i, got, want)
				}

				model = model[1:]
			}
		case 5:
			got, ok := b.PopBack()
			if want := len(model) > 0; ok != want {
				t.Fatalf("PopBack() #%d: ok = %v, want %v", i, ok, want)
			}

			if ok {
				if want := model[len(model)-1]; got != want {
					t.Fatalf("PopBack() #%d = %d, want %d", i, got, want)
				}

				model = model[:len(model)-1]
			}
		case 6:
			vs := []int{i, i + 1, i + 2}
			b.PushFrontN(vs)
			model = append(slices.Clone(vs), model...)
		case 7:
			if next(4) == 0 {
				b.Clip()
			} else {
				b.PushBackN([]int{i, i + 1})
				model = append(model, i, i+1)
			}
		}

		if got := contents(&b); !slices.Equal(got, model) {
			t.Fatalf("contents(b) #%d = %v, want %v", i, got, model)
		}
	}
}
