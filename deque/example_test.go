package deque_test

import (
	"fmt"

	"github.com/Nergous/structures/deque"
)

// A deque is double-ended: elements can be added and removed at both the
// front and the back.
func Example() {
	d := deque.New[int]()
	d.PushBack(2)
	d.PushFront(1)
	d.PushBack(3)

	for !d.IsEmpty() {
		v, _ := d.PopFront()
		fmt.Println(v)
	}
	// Output:
	// 1
	// 2
	// 3
}

// NewWithCap preallocates room for a known number of elements. Use it when the
// final size is known up front to avoid reallocations during PushBack.
func ExampleNewWithCap() {
	d := deque.NewWithCap[string](3)
	d.PushBack("a")
	d.PushBack("b")
	d.PushBack("c")

	fmt.Println(d.Len(), d.Cap())
	// Output: 3 3
}

// Front and Back return the elements at the ends without removing them. The
// boolean result is false for an empty deque.
func ExampleDeque_Front() {
	d := deque.New[int]()
	d.PushBackN(10, 20, 30)

	front, ok := d.Front()
	fmt.Println(front, ok)
	back, ok := d.Back()
	fmt.Println(back, ok)
	fmt.Println("len unchanged:", d.Len())

	empty := deque.New[int]()
	_, ok = empty.Front()
	fmt.Println(ok)
	// Output:
	// 10 true
	// 30 true
	// len unchanged: 3
	// false
}

// All returns a range-over-func iterator over the elements from front to back
// without consuming them. The deque is left intact after iteration.
func ExampleDeque_All() {
	d := deque.New[int]()
	d.PushBackN(1, 2, 3)

	for v := range d.All() {
		fmt.Println(v)
	}

	fmt.Println("len unchanged:", d.Len())
	// Output:
	// 1
	// 2
	// 3
	// len unchanged: 3
}

// Backward returns a range-over-func iterator over the elements from back to
// front without consuming them. The deque is left intact after iteration.
func ExampleDeque_Backward() {
	d := deque.New[int]()
	d.PushBackN(1, 2, 3)

	for v := range d.Backward() {
		fmt.Println(v)
	}

	fmt.Println("len unchanged:", d.Len())
	// Output:
	// 3
	// 2
	// 1
	// len unchanged: 3
}

// Clone returns an independent shallow copy. Mutating one deque does not
// affect the other, since they share no backing buffer.
func ExampleDeque_Clone() {
	a := deque.New[int]()
	a.PushBackN(1, 2, 3)

	b := a.Clone()
	b.PopFront() // removes 1 from b only

	fmt.Println(a.Slice())
	fmt.Println(b.Slice())
	// Output:
	// [1 2 3]
	// [2 3]
}

// Slice returns a copy of the elements from front to back, the same order All
// yields. The returned slice is independent of the deque.
func ExampleDeque_Slice() {
	d := deque.New[int]()
	d.PushBackN(1, 2, 3)

	out := d.Slice()
	fmt.Println(out)

	// The slice is detached: modifying it leaves the deque untouched.
	out[0] = 99
	front, _ := d.Front()
	fmt.Println("deque front:", front)

	// An empty deque returns nil.
	empty := deque.New[int]()
	fmt.Println(empty.Slice() == nil)
	// Output:
	// [1 2 3]
	// deque front: 1
	// true
}

// PushFront adds an element to the front, so it is the next one returned by
// PopFront.
func ExampleDeque_PushFront() {
	d := deque.New[string]()
	d.PushBack("normal")
	d.PushFront("urgent")

	fmt.Println(d.Slice())
	// Output: [urgent normal]
}

// PushBack adds an element to the back, so it is the next one returned by
// PopBack.
func ExampleDeque_PushBack() {
	d := deque.New[string]()
	d.PushBack("first")
	d.PushBack("second")

	fmt.Println(d.Slice())
	// Output: [first second]
}

// PushFrontN adds several elements to the front in argument order, so the
// first argument becomes the new front. It grows the backing buffer at most
// once.
func ExampleDeque_PushFrontN() {
	d := deque.New[int]()
	d.PushBackN(4, 5)
	d.PushFrontN(1, 2, 3)

	fmt.Println(d.Slice())
	// Output: [1 2 3 4 5]
}

// PushBackN adds several elements to the back in argument order, so the last
// argument becomes the new back. It grows the backing buffer at most once.
func ExampleDeque_PushBackN() {
	d := deque.New[int]()
	d.PushBack(1)
	d.PushBackN(2, 3, 4)

	fmt.Println(d.Slice())
	// Output: [1 2 3 4]
}

// PopFront removes and returns the front element. The boolean result reports
// whether the deque was non-empty; on an empty deque it is false and the
// returned value is the zero value.
func ExampleDeque_PopFront() {
	d := deque.New[string]()
	d.PushBack("first")
	d.PushBack("last")

	v, ok := d.PopFront()
	fmt.Println(v, ok)
	d.PopFront()         // removes "last"
	_, ok = d.PopFront() // deque is now empty
	fmt.Println(ok)
	// Output:
	// first true
	// false
}

// PopBack removes and returns the back element. The boolean result reports
// whether the deque was non-empty; on an empty deque it is false and the
// returned value is the zero value.
func ExampleDeque_PopBack() {
	d := deque.New[string]()
	d.PushBack("first")
	d.PushBack("last")

	v, ok := d.PopBack()
	fmt.Println(v, ok)
	d.PopBack()         // removes "first"
	_, ok = d.PopBack() // deque is now empty
	fmt.Println(ok)
	// Output:
	// last true
	// false
}

// PopFrontN removes up to n elements from the front and returns them in
// front-to-back order.
func ExampleDeque_PopFrontN() {
	d := deque.New[int]()
	d.PushBackN(1, 2, 3, 4)

	fmt.Println(d.PopFrontN(3))
	fmt.Println("remaining:", d.Slice())
	// Output:
	// [1 2 3]
	// remaining: [4]
}

// PopBackN removes up to n elements from the back and returns them in
// back-to-front order.
func ExampleDeque_PopBackN() {
	d := deque.New[int]()
	d.PushBackN(1, 2, 3, 4)

	fmt.Println(d.PopBackN(3))
	fmt.Println("remaining:", d.Slice())
	// Output:
	// [4 3 2]
	// remaining: [1]
}

// Clear empties the deque and releases its backing buffer: capacity drops back
// to zero, and the deque stays usable.
func ExampleDeque_Clear() {
	d := deque.NewWithCap[int](8)
	d.PushBackN(1, 2)

	d.Clear()
	fmt.Println(d.Len(), d.Cap())

	d.PushBack(3)
	fmt.Println(d.Slice())
	// Output:
	// 0 0
	// [3]
}

// Reset, unlike Clear, empties the deque but keeps the allocated backing
// buffer, so it can be reused without a new allocation.
func ExampleDeque_Reset() {
	d := deque.NewWithCap[int](8)
	d.PushBackN(1, 2)

	d.Reset()
	fmt.Println(d.Len(), d.Cap())
	// Output: 0 8
}

// Shrink copies the elements into a right-sized buffer and frees the old one
// immediately. Grow reserves capacity up front, and Clip trims spare capacity
// without copying.
func ExampleDeque_Shrink() {
	d := deque.NewWithCap[int](1024)
	d.PushBackN(1, 2, 3)

	d.Grow(2000) // reserve room for 2000 more elements
	fmt.Println(d.Cap() >= 2003)

	d.Shrink() // copy into a right-sized array, free the old one now
	fmt.Println(d.Cap())

	d.Clip() // already tight: no-op
	fmt.Println(d.Cap())
	// Output:
	// true
	// 3
	// 3
}

// A nil *Deque[T] behaves as an empty deque for every read-only method and for
// removals, so it is safe to query. Mutating methods still require a deque
// created with New (or a zero value).
func Example_nilReceiver() {
	var d *deque.Deque[int] // nil, never initialized

	fmt.Println(d.Len(), d.IsEmpty(), d.Cap())

	v, ok := d.PopFront()
	fmt.Println(v, ok)
	// Output:
	// 0 true 0
	// 0 false
}

// The zero value of Deque[T] is usable as an empty deque, so New is optional.
func Example_zeroValue() {
	var d deque.Deque[string]
	d.PushBack("b")
	d.PushFront("a")

	fmt.Println(d.Len(), d.Slice())
	// Output: 2 [a b]
}

// A practical example: process urgent items from the front while normal items
// continue to queue at the back.
func Example_urgentQueue() {
	tasks := deque.New[string]()
	tasks.PushBack("write tests")
	tasks.PushBack("run benchmarks")
	tasks.PushFront("fix build")

	for !tasks.IsEmpty() {
		task, _ := tasks.PopFront()
		fmt.Println(task)
	}
	// Output:
	// fix build
	// write tests
	// run benchmarks
}
