package stack_test

import (
	"fmt"

	"github.com/Nergous/structures/stack"
)

// A stack is LIFO (last in, first out): elements come out in the reverse
// order they went in.
func Example() {
	s := stack.New[int]()
	s.Push(1)
	s.Push(2)
	s.Push(3)

	for !s.IsEmpty() {
		v, _ := s.Pop()
		fmt.Println(v)
	}
	// Output:
	// 3
	// 2
	// 1
}

// NewWithCap preallocates room for a known number of elements. Use it when the
// final size is known up front to avoid reallocations during Push.
func ExampleNewWithCap() {
	s := stack.NewWithCap[string](3)
	s.Push("a")
	s.Push("b")
	s.Push("c")

	fmt.Println(s.Len(), s.Cap())
	// Output: 3 3
}

// Peek returns the top element without removing it. The boolean result is
// false for an empty stack.
func ExampleStack_Peek() {
	s := stack.New[int]()
	s.Push(10)
	s.Push(20)

	top, ok := s.Peek()
	fmt.Println(top, ok)
	fmt.Println("len unchanged:", s.Len())

	empty := stack.New[int]()
	_, ok = empty.Peek()
	fmt.Println(ok)
	// Output:
	// 20 true
	// len unchanged: 2
	// false
}

// All returns a range-over-func iterator over the elements from top to bottom
// without consuming them. The stack is left intact after iteration.
func ExampleStack_All() {
	s := stack.New[int]()
	s.PushN(1, 2, 3)

	for v := range s.All() {
		fmt.Println(v)
	}

	fmt.Println("len unchanged:", s.Len())
	// Output:
	// 3
	// 2
	// 1
	// len unchanged: 3
}

// Clone returns an independent shallow copy. Mutating one stack does not
// affect the other, since they share no backing array.
func ExampleStack_Clone() {
	a := stack.New[int]()
	a.PushN(1, 2, 3)

	b := a.Clone()
	b.Pop() // removes 3 from b only

	fmt.Println(a.Slice())
	fmt.Println(b.Slice())
	// Output:
	// [3 2 1]
	// [2 1]
}

// Slice returns a copy of the elements from top to bottom (LIFO), the same
// order All yields. The returned slice is independent of the stack.
func ExampleStack_Slice() {
	s := stack.New[int]()
	s.PushN(1, 2, 3)

	out := s.Slice()
	fmt.Println(out)

	// The slice is detached: modifying it leaves the stack untouched.
	out[0] = 99
	top, _ := s.Peek()
	fmt.Println("stack top:", top)

	// An empty stack returns nil.
	empty := stack.New[int]()
	fmt.Println(empty.Slice() == nil)
	// Output:
	// [3 2 1]
	// stack top: 3
	// true
}

// Push adds an element to the top of the stack, so the most recent push is
// the next one to come out.
func ExampleStack_Push() {
	s := stack.New[string]()
	s.Push("a")
	s.Push("b")

	top, _ := s.Peek()
	fmt.Println(top, s.Len())
	// Output: b 2
}

// PushN adds several elements at once, in argument order, so the last argument
// ends up on top. It is equivalent to calling Push for each value but grows the
// backing array at most once.
func ExampleStack_PushN() {
	s := stack.New[int]()
	s.PushN(1, 2, 3) // 3 ends up on top

	for !s.IsEmpty() {
		v, _ := s.Pop()
		fmt.Println(v)
	}
	// Output:
	// 3
	// 2
	// 1
}

// Pop removes and returns the top element. The boolean result reports whether
// the stack was non-empty; on an empty stack it is false and the returned
// value is the zero value.
func ExampleStack_Pop() {
	s := stack.New[string]()
	s.Push("first")
	s.Push("last")

	v, ok := s.Pop()
	fmt.Println(v, ok)
	s.Pop()         // removes "first"
	_, ok = s.Pop() // stack is now empty
	fmt.Println(ok)
	// Output:
	// last true
	// false
}

// PopN removes up to n elements from the top and returns them in LIFO order,
// so the first element of the result is the former top.
func ExampleStack_PopN() {
	s := stack.New[int]()
	s.PushN(1, 2, 3, 4, 5)

	fmt.Println(s.PopN(3))
	fmt.Println("remaining:", s.Slice())
	// Output:
	// [5 4 3]
	// remaining: [2 1]
}

// Clear empties the stack and releases its backing array: capacity drops back
// to zero, and the stack stays usable.
func ExampleStack_Clear() {
	s := stack.NewWithCap[int](8)
	s.PushN(1, 2)

	s.Clear()
	fmt.Println(s.Len(), s.Cap())

	s.Push(3)
	fmt.Println(s.Slice())
	// Output:
	// 0 0
	// [3]
}

// Reset, unlike Clear, empties the stack but keeps the allocated backing
// array, so it can be reused without a new allocation.
func ExampleStack_Reset() {
	s := stack.NewWithCap[int](8)
	s.PushN(1, 2)

	s.Reset()
	fmt.Println(s.Len(), s.Cap())
	// Output: 0 8
}

// Shrink copies the elements into a right-sized array and frees the old one
// immediately. Grow reserves capacity up front, and Clip trims spare capacity
// without copying.
func ExampleStack_Shrink() {
	s := stack.NewWithCap[int](1024)
	s.PushN(1, 2, 3)

	s.Grow(2000) // reserve room for 2000 more elements
	fmt.Println(s.Cap() >= 2003)

	s.Shrink() // copy into a right-sized array, free the old one now
	fmt.Println(s.Cap())

	s.Clip() // already tight: no-op
	fmt.Println(s.Cap())
	// Output:
	// true
	// 3
	// 3
}

// A nil *Stack[T] behaves as an empty stack for every read-only method and for
// removals, so it is safe to query. Mutating methods still require a stack
// created with New (or a zero value).
func Example_nilReceiver() {
	var s *stack.Stack[int] // nil, never initialized

	fmt.Println(s.Len(), s.IsEmpty(), s.Cap())

	v, ok := s.Pop()
	fmt.Println(v, ok)
	// Output:
	// 0 true 0
	// 0 false
}

// The zero value of Stack[T] is usable as an empty stack, so New is optional.
func Example_zeroValue() {
	var s stack.Stack[string]
	s.Push("a")
	s.Push("b")

	fmt.Println(s.Len(), s.Slice())
	// Output: 2 [b a]
}

// A practical example: checking whether brackets are balanced. Push each
// opening bracket; on a closing one, pop and verify the pair matches.
func Example_balancedBrackets() {
	balanced := func(input string) bool {
		pairs := map[rune]rune{')': '(', ']': '[', '}': '{'}
		s := stack.New[rune]()

		for _, r := range input {
			switch r {
			case '(', '[', '{':
				s.Push(r)
			case ')', ']', '}':
				top, ok := s.Pop()
				if !ok || top != pairs[r] {
					return false
				}
			}
		}

		return s.IsEmpty()
	}

	fmt.Println(balanced("([]{})"))
	fmt.Println(balanced("([)]"))
	// Output:
	// true
	// false
}
