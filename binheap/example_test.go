package binheap_test

import (
	"cmp"
	"fmt"

	"github.com/Nergous/structures/binheap"
)

// A heap is a priority structure that always hands out its least element
// first, regardless of the order in which elements were pushed. A heap built
// with New is a min-heap: the smallest value comes out first.
func Example() {
	h := binheap.New[int]()
	h.Push(3)
	h.Push(1)
	h.Push(2)

	for !h.IsEmpty() {
		v, _ := h.Pop()
		fmt.Println(v)
	}
	// Output:
	// 1
	// 2
	// 3
}

// NewWithCap preallocates room for a known number of elements. Use it when the
// final size is known up front to avoid reallocations during Push.
func ExampleNewWithCap() {
	h := binheap.NewWithCap[string](3)
	h.Push("b")
	h.Push("c")
	h.Push("a")

	fmt.Println(h.Len(), h.Cap())
	// Output: 3 3
}

// From builds a heap from a slice in O(n). The heap never aliases the slice,
// so later changes to the slice do not affect it.
func ExampleFrom() {
	vs := []int{5, 3, 8, 1}
	h := binheap.From(vs)
	vs[0] = 99 // does not affect h

	fmt.Println(h.Sorted())
	// Output: [1 3 5 8]
}

// NewMax returns a max-heap: the largest element is on top.
func ExampleNewMax() {
	h := binheap.NewMax[int]()
	h.Push(1)
	h.Push(3)
	h.Push(2)

	fmt.Println(h.PopN(h.Len()))
	// Output: [3 2 1]
}

// NewMaxWithCap preallocates a max-heap for a known number of elements.
func ExampleNewMaxWithCap() {
	h := binheap.NewMaxWithCap[int](3)
	h.Push(1)
	h.Push(3)
	h.Push(2)

	top, _ := h.Peek()
	fmt.Println(top, h.Cap())
	// Output: 3 3
}

// FromMax builds a max-heap from a slice in O(n), with the largest element on
// top.
func ExampleFromMax() {
	h := binheap.FromMax([]int{2, 9, 4})

	fmt.Println(h.PopN(h.Len()))
	// Output: [9 4 2]
}

// NewFunc orders elements with any comparator. Here tasks are ordered by
// priority, so the most urgent task (lowest number) comes out first.
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

// NewFuncWithCap combines a custom comparator with a preallocated backing
// array.
func ExampleNewFuncWithCap() {
	h := binheap.NewFuncWithCap(2, cmp.Compare[float64])
	h.Push(2.5)
	h.Push(0.5)

	top, _ := h.Peek()
	fmt.Println(top, h.Cap())
	// Output: 0.5 2
}

// FromFunc builds a heap from a slice in O(n), ordered by any comparator. Here
// words are ordered by length, shortest first.
func ExampleFromFunc() {
	h := binheap.FromFunc([]string{"ccc", "a", "bb"}, func(a, b string) int {
		return cmp.Compare(len(a), len(b))
	})

	fmt.Println(h.PopN(h.Len()))
	// Output: [a bb ccc]
}

// Peek returns the least element without removing it. The boolean result is
// false for an empty heap.
func ExampleHeap_Peek() {
	h := binheap.From([]int{4, 2, 9})

	top, ok := h.Peek()
	fmt.Println(top, ok)
	fmt.Println("len unchanged:", h.Len())

	empty := binheap.New[int]()
	_, ok = empty.Peek()
	fmt.Println(ok)
	// Output:
	// 2 true
	// len unchanged: 3
	// false
}

// Contains reports whether an element compares equal to the argument. It scans
// the heap in O(n).
func ExampleHeap_Contains() {
	h := binheap.From([]int{4, 8, 15})

	fmt.Println(h.Contains(8))
	fmt.Println(h.Contains(42)) // not present
	// Output:
	// true
	// false
}

// All returns a range-over-func iterator over the elements in internal layout
// order without consuming them. The heap is left intact after iteration. The
// order is not sorted, so All suits order-independent work such as sums and
// counts.
func ExampleHeap_All() {
	h := binheap.From([]int{4, 1, 3, 2})

	sum := 0
	for v := range h.All() {
		sum += v
	}

	fmt.Println("sum:", sum)
	fmt.Println("len unchanged:", h.Len())
	// Output:
	// sum: 10
	// len unchanged: 4
}

// Clone returns an independent shallow copy that keeps the comparator.
// Mutating one heap does not affect the other, since they share no backing
// array.
func ExampleHeap_Clone() {
	a := binheap.FromMax([]int{1, 2, 3})

	b := a.Clone()
	b.Pop() // removes 3 from b only

	fmt.Println(a.Sorted())
	fmt.Println(b.Sorted())
	// Output:
	// [3 2 1]
	// [2 1]
}

// Slice returns a copy of the elements in internal layout order, the same
// order All yields: the first one is the least, the rest are unordered. The
// returned slice is independent of the heap.
func ExampleHeap_Slice() {
	h := binheap.From([]int{4, 1, 3, 2})

	out := h.Slice()
	fmt.Println(len(out), out[0])

	// The slice is detached: modifying it leaves the heap untouched.
	out[0] = 99
	top, _ := h.Peek()
	fmt.Println("heap top:", top)

	// An empty heap returns nil.
	empty := binheap.New[int]()
	fmt.Println(empty.Slice() == nil)
	// Output:
	// 4 1
	// heap top: 1
	// true
}

// Sorted returns a copy of the elements in priority order without consuming
// the heap. For a max-heap the copy is in descending order.
func ExampleHeap_Sorted() {
	h := binheap.FromMax([]int{2, 9, 4, 7})

	fmt.Println(h.Sorted())
	fmt.Println("len unchanged:", h.Len())
	// Output:
	// [9 7 4 2]
	// len unchanged: 4
}

// Push adds an element to the heap, so the least element is always on top.
func ExampleHeap_Push() {
	h := binheap.New[int]()
	h.Push(3)
	h.Push(1)
	h.Push(2)

	top, _ := h.Peek()
	fmt.Println(top, h.Len())
	// Output: 1 3
}

// PushN adds several elements at once and grows the backing array at most
// once.
func ExampleHeap_PushN() {
	h := binheap.New[int]()
	h.PushN(7, 3, 9)

	fmt.Println(h.Sorted())
	// Output: [3 7 9]
}

// Pop removes and returns the least element. The boolean result reports
// whether the heap was non-empty; on an empty heap it is false and the
// returned value is the zero value.
func ExampleHeap_Pop() {
	h := binheap.From([]string{"pear", "apple"})

	v, ok := h.Pop()
	fmt.Println(v, ok)
	h.Pop()         // removes "pear"
	_, ok = h.Pop() // heap is now empty
	fmt.Println(ok)
	// Output:
	// apple true
	// false
}

// PopN removes up to n elements from the top and returns them in priority
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

// Remove deletes one element that compares equal to the argument. It scans the
// heap in O(n).
func ExampleHeap_Remove() {
	h := binheap.From([]int{1, 2, 3, 4})

	fmt.Println(h.Remove(3))
	fmt.Println(h.Remove(42)) // not present
	fmt.Println(h.Sorted())
	// Output:
	// true
	// false
	// [1 2 4]
}

// Clear empties the heap and releases its backing array: capacity drops back
// to zero, and the heap stays usable.
func ExampleHeap_Clear() {
	h := binheap.NewWithCap[int](8)
	h.PushN(2, 1)

	h.Clear()
	fmt.Println(h.Len(), h.Cap())

	h.Push(3)
	top, _ := h.Peek()
	fmt.Println(top)
	// Output:
	// 0 0
	// 3
}

// Reset, unlike Clear, empties the heap but keeps the allocated backing array,
// so it can be reused without a new allocation.
func ExampleHeap_Reset() {
	h := binheap.NewWithCap[int](8)
	h.PushN(2, 1)

	h.Reset()
	fmt.Println(h.Len(), h.Cap())
	// Output: 0 8
}

// Shrink copies the elements into a right-sized array and frees the old one
// immediately. Grow reserves capacity up front, and Clip trims spare capacity
// without copying.
func ExampleHeap_Shrink() {
	h := binheap.NewWithCap[int](1024)
	h.PushN(1, 2, 3)

	h.Grow(2000) // reserve room for 2000 more elements
	fmt.Println(h.Cap() >= 2003)

	h.Shrink() // copy into a right-sized array, free the old one now
	fmt.Println(h.Cap())

	h.Clip() // already tight: no-op
	fmt.Println(h.Cap())
	// Output:
	// true
	// 3
	// 3
}

// A nil *Heap[T] behaves as an empty heap for every read-only method and for
// removals, so it is safe to query. Mutating methods still require a heap
// created with a constructor such as New. Clone of a nil heap is nil, because
// there is no comparator to copy.
func Example_nilReceiver() {
	var h *binheap.Heap[int] // nil, never initialized

	fmt.Println(h.Len(), h.IsEmpty(), h.Cap())

	v, ok := h.Pop()
	fmt.Println(v, ok)
	fmt.Println(h.Clone() == nil)
	// Output:
	// 0 true 0
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
