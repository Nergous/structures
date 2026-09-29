package binheap_test

import (
	"cmp"
	"fmt"

	"github.com/Nergous/structures/binheap"
)

// A min-heap always hands out its smallest element first, regardless of the
// order in which elements were pushed.
func Example() {
	h := binheap.New[int]()
	h.PushN(5, 1, 4, 2, 3)

	for !h.IsEmpty() {
		v, _ := h.Pop()
		fmt.Println(v)
	}
	// Output:
	// 1
	// 2
	// 3
	// 4
	// 5
}

// NewMax builds a max-heap: the largest element is on top.
func ExampleNewMax() {
	h := binheap.NewMax[int]()
	h.PushN(3, 1, 2)

	fmt.Println(h.PopN(h.Len()))
	// Output: [3 2 1]
}

// NewFunc accepts any comparator. Here tasks are ordered by priority, so the
// most urgent task (lowest number) comes out first.
func ExampleNewFunc() {
	type task struct {
		name     string
		priority int
	}

	h := binheap.NewFunc(func(a, b task) int {
		return cmp.Compare(a.priority, b.priority)
	})
	h.Push(task{name: "write docs", priority: 3})
	h.Push(task{name: "fix outage", priority: 1})
	h.Push(task{name: "review PR", priority: 2})

	for !h.IsEmpty() {
		t, _ := h.Pop()
		fmt.Println(t.priority, t.name)
	}
	// Output:
	// 1 fix outage
	// 2 review PR
	// 3 write docs
}

// From copies and heapifies a slice in O(n). The heap never aliases the input,
// so later changes to the slice do not affect it.
func ExampleFrom() {
	s := []int{5, 3, 8, 1}
	h := binheap.From(s)
	s[0] = 100

	top, _ := h.Peek()
	fmt.Println(top)
	fmt.Println(h.Sorted())
	fmt.Println(s)
	// Output:
	// 1
	// [1 3 5 8]
	// [100 3 8 1]
}

// NewWithCap preallocates the backing array. Use it when the final size is
// known up front to avoid reallocations during Push.
func ExampleNewWithCap() {
	h := binheap.NewWithCap[string](3)
	h.PushN("b", "c", "a")

	fmt.Println(h.Len(), h.Cap())
	// Output: 3 3
}

// Pop removes and returns the least element. The second value reports whether
// the heap was non-empty; on an empty heap it is false and the value is the
// zero value.
func ExampleHeap_Pop() {
	h := binheap.From([]string{"pear", "apple"})

	v, ok := h.Pop()
	fmt.Println(v, ok)

	h.Pop() // removes "pear"

	_, ok = h.Pop() // heap is now empty
	fmt.Println(ok)
	// Output:
	// apple true
	// false
}

// PopN removes several least elements at once and returns them in priority
// order.
func ExampleHeap_PopN() {
	h := binheap.From([]int{7, 3, 9, 1, 5})

	fmt.Println(h.PopN(3))
	fmt.Println("remaining:", h.Len())
	// Output:
	// [1 3 5]
	// remaining: 2
}

// PushPop pushes and pops in one sift. When the new value is not greater than
// the least element, it comes straight back and the heap is untouched.
func ExampleHeap_PushPop() {
	h := binheap.From([]int{5, 10, 20})

	fmt.Println(h.PushPop(1)) // 1 would be popped first anyway
	fmt.Println(h.PushPop(7)) // 7 is stored, 5 is popped
	fmt.Println(h.Sorted())
	// Output:
	// 1
	// 5
	// [7 10 20]
}

// Replace pops first and pushes second: the old least element is returned even
// when the new value is smaller. On an empty heap it only stores the value.
func ExampleHeap_Replace() {
	h := binheap.From([]int{5, 10, 20})

	old, ok := h.Replace(30)
	fmt.Println(old, ok, h.Sorted())

	empty := binheap.New[int]()
	old, ok = empty.Replace(1)
	fmt.Println(old, ok, empty.Len())
	// Output:
	// 5 true [10 20 30]
	// 0 false 1
}

// Sorted returns an ordered snapshot without consuming the heap. For a
// max-heap the snapshot is in descending order.
func ExampleHeap_Sorted() {
	h := binheap.FromMax([]int{2, 9, 4, 7})

	fmt.Println(h.Sorted())
	fmt.Println("len unchanged:", h.Len())
	// Output:
	// [9 7 4 2]
	// len unchanged: 4
}

// All iterates over the heap without consuming it. Elements come in internal
// layout order, which is not sorted, so All suits order-independent work such
// as sums and counts.
func ExampleHeap_All() {
	h := binheap.From([]int{4, 1, 3, 2})

	sum := 0
	for v := range h.All() {
		sum += v
	}
	fmt.Println(sum, h.Len())
	// Output: 10 4
}

// Remove deletes one element that compares equal to the argument. It scans the
// heap in O(n).
func ExampleHeap_Remove() {
	h := binheap.From([]int{1, 2, 3, 4})

	fmt.Println(h.Remove(3))
	fmt.Println(h.Remove(42))
	fmt.Println(h.Sorted())
	// Output:
	// true
	// false
	// [1 2 4]
}

// Clone returns an independent copy that keeps the comparator.
func ExampleHeap_Clone() {
	a := binheap.FromMax([]int{1, 2, 3})

	b := a.Clone()
	b.Pop() // removes 3 from b only

	fmt.Println(a.Sorted(), b.Sorted())
	// Output: [3 2 1] [2 1]
}

// Grow reserves capacity up front; Shrink and Clip both reduce the capacity to
// the current length. Shrink copies into a right-sized array and frees the old
// one immediately, whereas Clip only reslices in O(1).
func ExampleHeap_Shrink() {
	h := binheap.NewWithCap[int](1024)
	h.PushN(1, 2, 3)

	h.Grow(2000)
	fmt.Println(h.Cap() >= 2003)

	h.Shrink()
	fmt.Println(h.Len(), h.Cap())

	h.Clip() // already tight: no-op
	fmt.Println(h.Len(), h.Cap())
	// Output:
	// true
	// 3 3
	// 3 3
}

// Reset empties the heap but keeps the allocated backing array for reuse.
func ExampleHeap_Reset() {
	h := binheap.NewWithCap[int](8)
	h.PushN(2, 1)

	h.Reset()
	fmt.Println(h.Len(), h.Cap())
	// Output: 0 8
}

// Clear, unlike Reset, releases the backing array. The comparator is kept, so
// the heap stays usable.
func ExampleHeap_Clear() {
	h := binheap.NewMaxWithCap[int](8)
	h.PushN(2, 1)

	h.Clear()
	fmt.Println(h.Len(), h.Cap())

	h.PushN(1, 3)
	top, _ := h.Peek()
	fmt.Println(top)
	// Output:
	// 0 0
	// 3
}

// The nil *Heap[T] behaves as an empty heap for reads and removals. Clone of a
// nil heap is nil, because there is no comparator to copy.
func Example_nilReceiver() {
	var h *binheap.Heap[int] // nil, never initialized

	fmt.Println(h.Len(), h.IsEmpty())

	v, ok := h.Pop()
	fmt.Println(v, ok)
	fmt.Println(h.Clone() == nil)
	// Output:
	// 0 true
	// 0 false
	// true
}

// A practical example: keeping the k largest values of a stream. A min-heap of
// size k holds the current top k; PushPop drops the smallest of them whenever a
// larger value arrives.
func Example_topK() {
	const k = 3
	h := binheap.NewWithCap[int](k)

	for _, v := range []int{5, 1, 9, 3, 7, 2, 8} {
		if h.Len() < k {
			h.Push(v)
			continue
		}
		h.PushPop(v)
	}

	fmt.Println(h.Sorted())
	// Output: [7 8 9]
}
