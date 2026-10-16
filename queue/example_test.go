package queue_test

import (
	"fmt"

	"github.com/Nergous/structures/queue"
)

// A queue is FIFO (first in, first out): elements come out in the same order
// they went in.
func Example() {
	q := queue.New[int]()
	q.Push(1)
	q.Push(2)
	q.Push(3)

	for !q.IsEmpty() {
		v, _ := q.Pop()
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
	q := queue.NewWithCap[string](3)
	q.Push("a")
	q.Push("b")
	q.Push("c")

	fmt.Println(q.Len(), q.Cap())
	// Output: 3 3
}

// Peek returns the front element without removing it. The boolean result is
// false for an empty queue.
func ExampleQueue_Peek() {
	q := queue.New[int]()
	q.Push(10)
	q.Push(20)

	front, ok := q.Peek()
	fmt.Println(front, ok)
	fmt.Println("len unchanged:", q.Len())

	empty := queue.New[int]()
	_, ok = empty.Peek()
	fmt.Println(ok)
	// Output:
	// 10 true
	// len unchanged: 2
	// false
}

// All returns a range-over-func iterator over the elements from front to back
// without consuming them. The queue is left intact after iteration.
func ExampleQueue_All() {
	q := queue.New[int]()
	q.PushN(1, 2, 3)

	for v := range q.All() {
		fmt.Println(v)
	}

	fmt.Println("len unchanged:", q.Len())
	// Output:
	// 1
	// 2
	// 3
	// len unchanged: 3
}

// Clone returns an independent shallow copy. Mutating one queue does not
// affect the other, since they share no backing buffer.
func ExampleQueue_Clone() {
	a := queue.New[int]()
	a.PushN(1, 2, 3)

	b := a.Clone()
	b.Pop() // removes 1 from b only

	fmt.Println(a.Slice())
	fmt.Println(b.Slice())
	// Output:
	// [1 2 3]
	// [2 3]
}

// Slice returns a copy of the elements from front to back (FIFO), the same
// order All yields. The returned slice is independent of the queue.
func ExampleQueue_Slice() {
	q := queue.New[int]()
	q.PushN(1, 2, 3)

	out := q.Slice()
	fmt.Println(out)

	// The slice is detached: modifying it leaves the queue untouched.
	out[0] = 99
	front, _ := q.Peek()
	fmt.Println("queue front:", front)

	// An empty queue returns nil.
	empty := queue.New[int]()
	fmt.Println(empty.Slice() == nil)
	// Output:
	// [1 2 3]
	// queue front: 1
	// true
}

// Push adds an element to the back of the queue, so the oldest push is the
// next one to come out.
func ExampleQueue_Push() {
	q := queue.New[string]()
	q.Push("a")
	q.Push("b")

	front, _ := q.Peek()
	fmt.Println(front, q.Len())
	// Output: a 2
}

// PushN adds several elements at once, in argument order, so the last argument
// ends up at the back. It is equivalent to calling Push for each value but
// grows the backing buffer at most once.
func ExampleQueue_PushN() {
	q := queue.New[int]()
	q.PushN(1, 2, 3) // 3 ends up at the back

	for !q.IsEmpty() {
		v, _ := q.Pop()
		fmt.Println(v)
	}
	// Output:
	// 1
	// 2
	// 3
}

// Pop removes and returns the front element (the oldest). The boolean result
// reports whether the queue was non-empty; on an empty queue it is false and
// the returned value is the zero value.
func ExampleQueue_Pop() {
	q := queue.New[string]()
	q.Push("first")
	q.Push("last")

	v, ok := q.Pop()
	fmt.Println(v, ok)
	q.Pop()         // removes "last"
	_, ok = q.Pop() // queue is now empty
	fmt.Println(ok)
	// Output:
	// first true
	// false
}

// PopN removes up to n elements from the front and returns them in FIFO order,
// so the first element of the result is the former front.
func ExampleQueue_PopN() {
	q := queue.New[int]()
	q.PushN(1, 2, 3, 4)

	fmt.Println(q.PopN(3))
	fmt.Println("remaining:", q.Slice())
	// Output:
	// [1 2 3]
	// remaining: [4]
}

// Clear empties the queue and releases its backing buffer: capacity drops back
// to zero, and the queue stays usable.
func ExampleQueue_Clear() {
	q := queue.NewWithCap[int](8)
	q.PushN(1, 2)

	q.Clear()
	fmt.Println(q.Len(), q.Cap())

	q.Push(3)
	fmt.Println(q.Slice())
	// Output:
	// 0 0
	// [3]
}

// Reset, unlike Clear, empties the queue but keeps the allocated backing
// buffer, so it can be reused without a new allocation.
func ExampleQueue_Reset() {
	q := queue.NewWithCap[int](8)
	q.PushN(1, 2)

	q.Reset()
	fmt.Println(q.Len(), q.Cap())
	// Output: 0 8
}

// Shrink copies the elements into a right-sized buffer and frees the old one
// immediately. Grow reserves capacity up front, and Clip trims spare capacity
// without copying.
func ExampleQueue_Shrink() {
	q := queue.NewWithCap[int](1024)
	q.PushN(1, 2, 3)

	q.Grow(2000) // reserve room for 2000 more elements
	fmt.Println(q.Cap() >= 2003)

	q.Shrink() // copy into a right-sized array, free the old one now
	fmt.Println(q.Cap())

	q.Clip() // already tight: no-op
	fmt.Println(q.Cap())
	// Output:
	// true
	// 3
	// 3
}

// A nil *Queue[T] behaves as an empty queue for every read-only method and for
// removals, so it is safe to query. Mutating methods still require a queue
// created with New (or a zero value).
func Example_nilReceiver() {
	var q *queue.Queue[int] // nil, never initialized

	fmt.Println(q.Len(), q.IsEmpty(), q.Cap())

	v, ok := q.Pop()
	fmt.Println(v, ok)
	// Output:
	// 0 true 0
	// 0 false
}

// The zero value of Queue[T] is usable as an empty queue, so New is optional.
func Example_zeroValue() {
	var q queue.Queue[string]
	q.Push("a")
	q.Push("b")

	fmt.Println(q.Len(), q.Slice())
	// Output: 2 [a b]
}

// A practical example: breadth-first (level-order) traversal of a binary tree.
// A queue visits nodes in arrival order, so each level is processed left to
// right before the next level begins.
func Example_breadthFirstTraversal() {
	type node struct {
		value       int
		left, right *node
	}

	//        1
	//       / \
	//      2   3
	//     / \   \
	//    4   5   6
	tree := &node{
		value: 1,
		left: &node{
			value: 2,
			left:  &node{value: 4},
			right: &node{value: 5},
		},
		right: &node{
			value: 3,
			right: &node{value: 6},
		},
	}

	q := queue.New[*node]()
	q.Push(tree)

	for !q.IsEmpty() {
		n, _ := q.Pop()
		fmt.Println(n.value)

		if n.left != nil {
			q.Push(n.left)
		}

		if n.right != nil {
			q.Push(n.right)
		}
	}
	// Output:
	// 1
	// 2
	// 3
	// 4
	// 5
	// 6
}
