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
	if b.Len() != 0 || b.Cap() != 0 {
		t.Fatalf("zero Buffer Len=%d Cap=%d, want 0, 0", b.Len(), b.Cap())
	}
	if _, ok := b.Front(); ok {
		t.Errorf("Front() on empty ok = true")
	}
	if _, ok := b.Back(); ok {
		t.Errorf("Back() on empty ok = true")
	}
	if _, ok := b.PopFront(); ok {
		t.Errorf("PopFront() on empty ok = true")
	}
	if _, ok := b.PopBack(); ok {
		t.Errorf("PopBack() on empty ok = true")
	}
	b.PushBack(1)
	if v, _ := b.Front(); v != 1 {
		t.Errorf("Front() after PushBack = %d, want 1", v)
	}
}

func TestWithCap(t *testing.T) {
	for _, tt := range []struct{ n, want int }{{-5, 0}, {0, 0}, {1, 1}, {10, 10}} {
		b := WithCap[int](tt.n)
		if b.Cap() != tt.want {
			t.Errorf("WithCap(%d).Cap() = %d, want %d", tt.n, b.Cap(), tt.want)
		}
	}
}

func TestBuffer_PushPopBothEnds(t *testing.T) {
	var b Buffer[int]
	b.PushBack(2)
	b.PushBack(3)
	b.PushFront(1)
	b.PushFront(0)

	if got, want := contents(&b), []int{0, 1, 2, 3}; !slices.Equal(got, want) {
		t.Fatalf("contents = %v, want %v", got, want)
	}
	if v, _ := b.Front(); v != 0 {
		t.Errorf("Front() = %d, want 0", v)
	}
	if v, _ := b.Back(); v != 3 {
		t.Errorf("Back() = %d, want 3", v)
	}
	if v, _ := b.PopBack(); v != 3 {
		t.Errorf("PopBack() = %d, want 3", v)
	}
	if v, _ := b.PopFront(); v != 0 {
		t.Errorf("PopFront() = %d, want 0", v)
	}
	if got, want := contents(&b), []int{1, 2}; !slices.Equal(got, want) {
		t.Errorf("contents = %v, want %v", got, want)
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

	if b.Cap() != 4 {
		t.Fatalf("Cap() = %d, want 4 (no growth)", b.Cap())
	}
	first, second := b.Segments()
	if len(second) == 0 {
		t.Errorf("Segments() second = empty, want wrapped elements")
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
			t.Errorf("contents = %v, want %v", got, want)
		}
	})

	t.Run("front makes first argument the new front", func(t *testing.T) {
		var b Buffer[int]
		b.PushBack(9)
		b.PushFrontN([]int{1, 2, 3})
		if got, want := contents(&b), []int{1, 2, 3, 9}; !slices.Equal(got, want) {
			t.Errorf("contents = %v, want %v", got, want)
		}
	})

	t.Run("front on wrapped buffer", func(t *testing.T) {
		b := WithCap[int](4)
		b.PushBackN([]int{1, 2, 3})
		b.PopFront()
		b.PushFrontN([]int{7, 8})
		if got, want := contents(&b), []int{7, 8, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("contents = %v, want %v", got, want)
		}
	})

	t.Run("empty argument list is a no-op", func(t *testing.T) {
		var b Buffer[int]
		b.PushBackN(nil)
		b.PushFrontN(nil)
		if b.Len() != 0 || b.Cap() != 0 {
			t.Errorf("Len=%d Cap=%d, want 0, 0", b.Len(), b.Cap())
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
		t.Errorf("PopBackN on empty = %v, want nil", got)
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
	for i, p := range backing {
		if p != nil {
			t.Errorf("slot %d = %p, want nil", i, p)
		}
	}
}

func TestBuffer_Growth(t *testing.T) {
	var b Buffer[int]
	var caps []int
	for i := range 5 {
		b.PushBack(i)
		if len(caps) == 0 || caps[len(caps)-1] != b.Cap() {
			caps = append(caps, b.Cap())
		}
	}
	if want := []int{1, 2, 4, 8}; !slices.Equal(caps, want) {
		t.Errorf("capacity sequence = %v, want %v", caps, want)
	}
}

func TestNextCap(t *testing.T) {
	tests := []struct{ current, need, want int }{
		{0, 1, 1},
		{0, 3, 4},
		{1, 2, 2},
		{4, 5, 8},
		{512, 513, 1024},
		{1024, 1025, 1280},
		{-3, 2, 2},
	}
	for _, tt := range tests {
		if got := nextCap(tt.current, tt.need); got != tt.want {
			t.Errorf("nextCap(%d, %d) = %d, want %d", tt.current, tt.need, got, tt.want)
		}
	}
}

func TestBuffer_Grow(t *testing.T) {
	var b Buffer[int]
	b.PushBackN([]int{1, 2, 3})
	b.Grow(0)
	b.Grow(-4)
	before := b.Cap()
	b.Grow(100)
	if b.Cap() < 103 {
		t.Errorf("Cap() after Grow(100) = %d, want >= 103", b.Cap())
	}
	if b.Cap() == before {
		t.Errorf("Grow(100) did not reallocate")
	}
	if got, want := contents(&b), []int{1, 2, 3}; !slices.Equal(got, want) {
		t.Errorf("contents = %v, want %v", got, want)
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
		if b.Cap() != 3 || &b.Data()[0] == old {
			t.Errorf("Shrink: Cap=%d, reused array=%v, want 3, false", b.Cap(), &b.Data()[0] == old)
		}
		if got, want := contents(&b), []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("contents = %v, want %v", got, want)
		}
	})

	t.Run("clip reslices contiguous elements", func(t *testing.T) {
		b := filled()
		old := &b.Data()[0]
		b.Clip()
		if b.Cap() != 3 || &b.Data()[0] != old {
			t.Errorf("Clip: Cap=%d, same array=%v, want 3, true", b.Cap(), &b.Data()[0] == old)
		}
	})

	t.Run("clip copies wrapped elements", func(t *testing.T) {
		b := wrapped()
		b.Clip()
		if b.Cap() != 4 {
			t.Errorf("Cap() = %d, want 4", b.Cap())
		}
		if got, want := contents(&b), []int{3, 4, 5, 6}; !slices.Equal(got, want) {
			t.Errorf("contents = %v, want %v", got, want)
		}
	})

	t.Run("shrink keeps wrapped order", func(t *testing.T) {
		b := wrapped()
		b.PopBack()
		b.Shrink()
		if got, want := contents(&b), []int{3, 4, 5}; !slices.Equal(got, want) || b.Cap() != 3 {
			t.Errorf("contents = %v (cap %d), want %v (cap 3)", got, b.Cap(), want)
		}
	})

	t.Run("empty drops the array", func(t *testing.T) {
		for name, f := range map[string]func(*Buffer[int]){
			"Shrink": (*Buffer[int]).Shrink,
			"Clip":   (*Buffer[int]).Clip,
		} {
			b := WithCap[int](8)
			f(&b)
			if b.Cap() != 0 {
				t.Errorf("%s on empty: Cap() = %d, want 0", name, b.Cap())
			}
		}
	})

	t.Run("already right-sized is a no-op", func(t *testing.T) {
		b := WithCap[int](2)
		b.PushBackN([]int{1, 2})
		old := &b.Data()[0]
		b.Shrink()
		b.Clip()
		if &b.Data()[0] != old {
			t.Errorf("right-sized buffer was reallocated")
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
	if b.Len() != 0 || b.Cap() != 4 {
		t.Errorf("Reset: Len=%d Cap=%d, want 0, 4", b.Len(), b.Cap())
	}
	for i, p := range backing {
		if p != nil {
			t.Errorf("Reset left slot %d = %p, want nil", i, p)
		}
	}

	b.PushBack(&x)
	b.Clear()
	if b.Len() != 0 || b.Cap() != 0 {
		t.Errorf("Clear: Len=%d Cap=%d, want 0, 0", b.Len(), b.Cap())
	}
	b.PushBack(&x)
	if b.Len() != 1 {
		t.Errorf("buffer unusable after Clear")
	}
}

func TestBuffer_Clone(t *testing.T) {
	b := WithCap[int](4)
	b.PushBackN([]int{1, 2, 3, 4})
	b.PopFrontN(2)
	b.PushBack(5) // wrapped

	c := b.Clone()
	if got, want := contents(&c), []int{3, 4, 5}; !slices.Equal(got, want) {
		t.Fatalf("clone contents = %v, want %v", got, want)
	}
	if c.Cap() != 3 {
		t.Errorf("clone Cap() = %d, want 3", c.Cap())
	}
	c.PopFront()
	if b.Len() != 3 {
		t.Errorf("mutating the clone changed the original")
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
			v, ok := b.PopFront()
			if ok != (len(model) > 0) {
				t.Fatalf("step %d: PopFront ok = %v", i, ok)
			}
			if ok {
				if v != model[0] {
					t.Fatalf("step %d: PopFront = %d, want %d", i, v, model[0])
				}
				model = model[1:]
			}
		case 5:
			v, ok := b.PopBack()
			if ok != (len(model) > 0) {
				t.Fatalf("step %d: PopBack ok = %v", i, ok)
			}
			if ok {
				if v != model[len(model)-1] {
					t.Fatalf("step %d: PopBack = %d, want %d", i, v, model[len(model)-1])
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
			t.Fatalf("step %d: contents = %v, want %v", i, got, model)
		}
	}
}
