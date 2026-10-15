package deque

import (
	"strings"
	"testing"

	"github.com/Nergous/structures/internal/shared/sharedtest"
)

func TestDeque_PushPopBothEnds(t *testing.T) {
	d := New[int]()
	d.PushBack(2)
	d.PushFront(1)
	d.PushBack(3)

	if got, want := d.Slice(), []int{1, 2, 3}; !equalInts(got, want) {
		t.Fatalf("Slice() = %v, want %v", got, want)
	}

	if got, ok := d.Front(); !ok || got != 1 {
		t.Errorf("Front() = (%d, %v), want (1, true)", got, ok)
	}

	if got, ok := d.Back(); !ok || got != 3 {
		t.Errorf("Back() = (%d, %v), want (3, true)", got, ok)
	}

	if got, ok := d.PopFront(); !ok || got != 1 {
		t.Errorf("PopFront() = (%d, %v), want (1, true)", got, ok)
	}

	if got, ok := d.PopBack(); !ok || got != 3 {
		t.Errorf("PopBack() = (%d, %v), want (3, true)", got, ok)
	}

	if got, want := d.Slice(), []int{2}; !equalInts(got, want) {
		t.Errorf("Slice() after PopFront(), PopBack() = %v, want %v", got, want)
	}
}

func TestDeque_BulkOperations(t *testing.T) {
	d := New[int]()
	d.PushBackN(4, 5)
	d.PushFrontN(1, 2, 3)

	if got, want := d.Slice(), []int{1, 2, 3, 4, 5}; !equalInts(got, want) {
		t.Fatalf("Slice() after PushBackN(4, 5), PushFrontN(1, 2, 3) = %v, want %v", got, want)
	}

	if got, want := d.PopFrontN(2), []int{1, 2}; !equalInts(got, want) {
		t.Errorf("PopFrontN(2) = %v, want %v", got, want)
	}

	if got, want := d.PopBackN(2), []int{5, 4}; !equalInts(got, want) {
		t.Errorf("PopBackN(2) = %v, want %v", got, want)
	}

	if got, want := d.Slice(), []int{3}; !equalInts(got, want) {
		t.Errorf("Slice() after PopFrontN(2), PopBackN(2) = %v, want %v", got, want)
	}
}

func TestDeque_PopNDrainsAndHandlesEmpty(t *testing.T) {
	d := NewWithCap[int](8)
	d.PushBackN(1, 2, 3)

	if got, want := d.PopFrontN(10), []int{1, 2, 3}; !equalInts(got, want) {
		t.Fatalf("PopFrontN(10) = %v, want %v", got, want)
	}

	if got := d.Len(); got != 0 {
		t.Errorf("Len() after PopFrontN(10) = %d, want 0", got)
	}

	if got := d.Cap(); got != 8 {
		t.Errorf("Cap() after PopFrontN(10) = %d, want 8", got)
	}

	if got := d.PopBackN(1); got != nil {
		t.Errorf("PopBackN(1) on empty = %v, want nil", got)
	}

	if got := d.PopFrontN(0); got != nil {
		t.Errorf("PopFrontN(0) = %v, want nil", got)
	}

	if got := d.PopBackN(-1); got != nil {
		t.Errorf("PopBackN(-1) = %v, want nil", got)
	}
}

func TestDeque_WrapAround(t *testing.T) {
	d := NewWithCap[int](4)
	d.PushBackN(1, 2, 3, 4)
	d.PopFront()
	d.PopFront()
	d.PushBackN(5, 6)

	if got, want := d.Slice(), []int{3, 4, 5, 6}; !equalInts(got, want) {
		t.Fatalf("Slice() across wraparound = %v, want %v", got, want)
	}

	if got, want := d.PopBackN(3), []int{6, 5, 4}; !equalInts(got, want) {
		t.Errorf("PopBackN(3) across wraparound = %v, want %v", got, want)
	}

	if got, want := d.Slice(), []int{3}; !equalInts(got, want) {
		t.Errorf("Slice() after PopBackN(3) = %v, want %v", got, want)
	}
}

func TestDeque_AllCloneAndSlice(t *testing.T) {
	d := New[int]()
	d.PushBackN(1, 2, 3)

	var got []int
	for v := range d.All() {
		got = append(got, v)
	}

	if want := []int{1, 2, 3}; !equalInts(got, want) {
		t.Fatalf("All() = %v, want %v", got, want)
	}

	cloned := d.Clone()
	d.PopFront()
	cloned.PushBack(4)

	if got, want := d.Slice(), []int{2, 3}; !equalInts(got, want) {
		t.Errorf("original Slice() = %v, want %v (independent of clone)", got, want)
	}

	if got, want := cloned.Slice(), []int{1, 2, 3, 4}; !equalInts(got, want) {
		t.Errorf("clone Slice() = %v, want %v (independent of original)", got, want)
	}
}

func TestDeque_MemoryControls(t *testing.T) {
	t.Run("Reset keeps capacity", func(t *testing.T) {
		d := NewWithCap[int](8)
		d.PushBackN(1, 2, 3)
		d.Reset()

		if got := d.Len(); got != 0 {
			t.Errorf("Len() after Reset() = %d, want 0", got)
		}

		if got := d.Cap(); got != 8 {
			t.Errorf("Cap() after Reset() = %d, want 8", got)
		}

		d.PushBack(4)

		if got, ok := d.PopFront(); !ok || got != 4 {
			t.Errorf("PopFront() after Reset(), PushBack(4) = (%d, %v), want (4, true)", got, ok)
		}
	})

	t.Run("Clear releases capacity", func(t *testing.T) {
		d := NewWithCap[int](8)
		d.PushBackN(1, 2, 3)
		d.Clear()

		if got := d.Len(); got != 0 {
			t.Errorf("Len() after Clear() = %d, want 0", got)
		}

		if got := d.Cap(); got != 0 {
			t.Errorf("Cap() after Clear() = %d, want 0", got)
		}
	})

	t.Run("Shrink and Clip trim to length", func(t *testing.T) {
		d := NewWithCap[int](16)
		d.PushBackN(1, 2, 3)
		d.Shrink()

		if got := d.Cap(); got != 3 {
			t.Errorf("Cap() after Shrink() = %d, want 3", got)
		}

		d.Grow(10)

		if got := d.Cap(); got < 13 {
			t.Errorf("Cap() after Grow(10) = %d, want >= 13", got)
		}

		d.Clip()

		if got := d.Cap(); got != 3 {
			t.Errorf("Cap() after Clip() = %d, want 3", got)
		}

		if got, want := d.Slice(), []int{1, 2, 3}; !equalInts(got, want) {
			t.Errorf("Slice() after Shrink(), Grow(10), Clip() = %v, want %v", got, want)
		}
	})
}

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

	if got, ok := d.PopFront(); ok || got != 0 {
		t.Errorf("nil.PopFront() = (%d, %v), want (0, false)", got, ok)
	}

	if got, ok := d.PopBack(); ok || got != 0 {
		t.Errorf("nil.PopBack() = (%d, %v), want (0, false)", got, ok)
	}

	if got := d.PopFrontN(1); got != nil {
		t.Errorf("nil.PopFrontN(1) = %v, want nil", got)
	}

	if got := d.PopBackN(1); got != nil {
		t.Errorf("nil.PopBackN(1) = %v, want nil", got)
	}

	if got := d.Slice(); got != nil {
		t.Errorf("nil.Slice() = %v, want nil", got)
	}

	c := d.Clone()
	if c == nil {
		t.Fatalf("nil.Clone() = nil, want a usable empty deque")
	}

	if got := c.Len(); got != 0 {
		t.Errorf("nil.Clone().Len() = %d, want 0", got)
	}
}

func TestDeque_ZeroValue(t *testing.T) {
	var c Deque[int]

	if got := c.Len(); got != 0 {
		t.Errorf("Len() on zero value = %d, want 0", got)
	}

	if got := c.Cap(); got != 0 {
		t.Errorf("Cap() on zero value = %d, want 0", got)
	}

	if got := c.IsEmpty(); !got {
		t.Errorf("IsEmpty() on zero value = %v, want true", got)
	}

	if got, ok := c.PopFront(); ok {
		t.Errorf("PopFront() on zero value: got %d, ok = true, want ok = false", got)
	}

	c.PushBack(1)
	c.PushBack(2)

	if got := c.Len(); got != 2 {
		t.Errorf("Len() after pushes = %d, want 2", got)
	}

	if got, ok := c.PopFront(); !ok || got != 1 {
		t.Errorf("PopFront() = (%d, %v), want (1, true)", got, ok)
	}

	c.Reset()
	c.Clear()
	c.Shrink()
	c.Clip()
	c.Grow(4)

	if got := c.Cap(); got < 4 {
		t.Errorf("Cap() after Grow(4) = %d, want >= 4", got)
	}
}

func TestDeque_NilReceiverMutatorsPanic(t *testing.T) {
	tests := []struct {
		name string
		call func(d *Deque[int])
	}{
		{
			name: "PushFront",
			call: func(d *Deque[int]) { d.PushFront(1) },
		},
		{
			name: "PushBack",
			call: func(d *Deque[int]) { d.PushBack(1) },
		},
		{
			name: "PushFrontN",
			call: func(d *Deque[int]) { d.PushFrontN(1, 2) },
		},
		{
			name: "PushBackN",
			call: func(d *Deque[int]) { d.PushBackN(1, 2) },
		},
		{
			name: "Grow",
			call: func(d *Deque[int]) { d.Grow(1) },
		},
		{
			name: "Clear",
			call: func(d *Deque[int]) { d.Clear() },
		},
		{
			name: "Reset",
			call: func(d *Deque[int]) { d.Reset() },
		},
		{
			name: "Shrink",
			call: func(d *Deque[int]) { d.Shrink() },
		},
		{
			name: "Clip",
			call: func(d *Deque[int]) { d.Clip() },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d *Deque[int]
			sharedtest.Panics(t, tt.name, "deque: "+strings.Fields(tt.name)[0]+" on nil receiver", func() { tt.call(d) })
		})
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
