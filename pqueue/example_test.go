package pqueue_test

import (
	"cmp"
	"fmt"
	"time"

	"github.com/Nergous/structures/pqueue"
)

// A priority queue is a collection of unique keys, each with a priority, that
// hands out the key with the smallest priority first, regardless of the order
// in which keys were pushed.
func Example() {
	q := pqueue.New[string, int]()
	q.Push("write docs", 3)
	q.Push("fix outage", 1)
	q.Push("review PR", 2)

	for !q.IsEmpty() {
		key, priority, _ := q.Pop()
		fmt.Println(priority, key)
	}
	// Output:
	// 1 fix outage
	// 2 review PR
	// 3 write docs
}

// NewWithCap preallocates room for a known number of keys. Use it when the
// final size is known up front to avoid reallocations during Push.
func ExampleNewWithCap() {
	q := pqueue.NewWithCap[string, int](3)
	q.Push("b", 2)
	q.Push("c", 3)
	q.Push("a", 1)

	fmt.Println(q.Len(), q.Cap())
	// Output: 3 3
}

// From builds a priority queue from items in O(n). A key that appears more
// than once keeps the priority of its last occurrence. The priority queue
// never aliases the items, so later changes to the slice do not affect it.
func ExampleFrom() {
	items := []pqueue.Item[string, int]{
		{Key: "b", Priority: 2},
		{Key: "a", Priority: 5},
		{Key: "c", Priority: 3},
		{Key: "a", Priority: 1}, // last occurrence wins
	}
	q := pqueue.From(items)
	items[0].Priority = 99 // does not affect q

	fmt.Println(q.Sorted())
	// Output: [{a 1} {b 2} {c 3}]
}

// NewMax returns a max-priority queue: the largest priority is on top. Keys
// with equal priorities come out in the order they were first pushed.
func ExampleNewMax() {
	q := pqueue.NewMax[string, int]()
	q.Push("bronze", 1)
	q.Push("gold", 3)
	q.Push("silver", 2)
	q.Push("gold, tied", 3)

	for _, it := range q.PopN(q.Len()) {
		fmt.Println(it.Priority, it.Key)
	}
	// Output:
	// 3 gold
	// 3 gold, tied
	// 2 silver
	// 1 bronze
}

// NewMaxWithCap preallocates a max-priority queue for a known number of keys.
func ExampleNewMaxWithCap() {
	q := pqueue.NewMaxWithCap[string, int](3)
	q.Push("a", 1)
	q.Push("c", 3)
	q.Push("b", 2)

	key, priority, _ := q.Peek()
	fmt.Println(key, priority, q.Cap())
	// Output: c 3 3
}

// FromMax builds a priority queue from items in O(n), with the largest
// priority on top.
func ExampleFromMax() {
	q := pqueue.FromMax([]pqueue.Item[string, int]{
		{Key: "bronze", Priority: 1},
		{Key: "gold", Priority: 3},
		{Key: "silver", Priority: 2},
	})

	fmt.Println(q.PopN(q.Len()))
	// Output: [{gold 3} {silver 2} {bronze 1}]
}

// NewFunc orders priorities with any comparator. Here jobs are scheduled by
// deadline, so the earliest deadline comes out first.
func ExampleNewFunc() {
	start := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	q := pqueue.NewFunc[string](time.Time.Compare)
	q.Push("send report", start.Add(3*time.Hour))
	q.Push("backup", start.Add(1*time.Hour))
	q.Push("rotate logs", start.Add(2*time.Hour))

	for !q.IsEmpty() {
		key, priority, _ := q.Pop()
		fmt.Println(priority.Format("15:04"), key)
	}
	// Output:
	// 10:00 backup
	// 11:00 rotate logs
	// 12:00 send report
}

// NewFuncWithCap combines a custom comparator with a preallocated backing
// array.
func ExampleNewFuncWithCap() {
	q := pqueue.NewFuncWithCap[string](2, cmp.Compare[float64])
	q.Push("x", 2.5)
	q.Push("y", 0.5)

	key, priority, _ := q.Peek()
	fmt.Println(key, priority, q.Cap())
	// Output: y 0.5 2
}

// FromFunc builds a priority queue from items in O(n), ordered by any
// comparator. Here priorities are words ordered by length, shortest first.
func ExampleFromFunc() {
	q := pqueue.FromFunc([]pqueue.Item[int, string]{
		{Key: 1, Priority: "ccc"},
		{Key: 2, Priority: "a"},
		{Key: 3, Priority: "bb"},
	}, func(a, b string) int {
		return cmp.Compare(len(a), len(b))
	})

	fmt.Println(q.PopN(q.Len()))
	// Output: [{2 a} {3 bb} {1 ccc}]
}

// Peek returns the top key and its priority without removing them. The
// boolean result is false for an empty priority queue.
func ExampleQueue_Peek() {
	q := pqueue.New[string, int]()
	q.Push("b", 2)
	q.Push("a", 1)

	key, priority, ok := q.Peek()
	fmt.Println(key, priority, ok)
	fmt.Println("len unchanged:", q.Len())

	empty := pqueue.New[string, int]()
	_, _, ok = empty.Peek()
	fmt.Println(ok)
	// Output:
	// a 1 true
	// len unchanged: 2
	// false
}

// Contains reports whether a key is queued. It looks the key up in O(1) on
// average, without scanning the priority queue.
func ExampleQueue_Contains() {
	q := pqueue.New[string, int]()
	q.Push("a", 1)
	q.Push("b", 2)

	fmt.Println(q.Contains("b"))
	fmt.Println(q.Contains("zzz")) // not present
	// Output:
	// true
	// false
}

// All returns a range-over-func iterator over the keys and priorities in
// internal layout order without consuming them. The priority queue is left
// intact after iteration. The order is not sorted, so All suits
// order-independent work such as sums and counts.
func ExampleQueue_All() {
	q := pqueue.New[string, int]()
	q.Push("a", 4)
	q.Push("b", 1)
	q.Push("c", 3)

	sum := 0
	for _, priority := range q.All() {
		sum += priority
	}

	fmt.Println("sum:", sum)
	fmt.Println("len unchanged:", q.Len())
	// Output:
	// sum: 8
	// len unchanged: 3
}

// Clone returns an independent shallow copy that keeps the comparator.
// Mutating one priority queue does not affect the other, since they share no
// backing storage.
func ExampleQueue_Clone() {
	a := pqueue.FromMax([]pqueue.Item[string, int]{
		{Key: "x", Priority: 1},
		{Key: "y", Priority: 2},
		{Key: "z", Priority: 3},
	})

	b := a.Clone()
	b.Pop() // removes "z" from b only

	fmt.Println(a.Sorted())
	fmt.Println(b.Sorted())
	// Output:
	// [{z 3} {y 2} {x 1}]
	// [{y 2} {x 1}]
}

// Slice returns a copy of the keys and priorities in internal layout order,
// the same order All yields: the first item has the least priority, the rest
// are unordered. The returned slice is independent of the priority queue.
func ExampleQueue_Slice() {
	q := pqueue.New[string, int]()
	q.Push("a", 4)
	q.Push("b", 1)
	q.Push("c", 3)

	out := q.Slice()
	fmt.Println(len(out), out[0])

	// The slice is detached: modifying it leaves the priority queue untouched.
	out[0].Priority = 99
	key, priority, _ := q.Peek()
	fmt.Println("priority queue top:", key, priority)

	// An empty priority queue returns nil.
	empty := pqueue.New[string, int]()
	fmt.Println(empty.Slice() == nil)
	// Output:
	// 3 {b 1}
	// priority queue top: b 1
	// true
}

// Sorted returns a copy of the keys and priorities in priority order without
// consuming the priority queue. For a max-priority queue the copy is in
// descending order.
func ExampleQueue_Sorted() {
	q := pqueue.New[string, float64]()
	q.Push("x", 2.5)
	q.Push("y", 0.5)
	q.Push("z", 1.5)

	fmt.Println(q.Sorted())
	fmt.Println("len unchanged:", q.Len())
	// Output:
	// [{y 0.5} {z 1.5} {x 2.5}]
	// len unchanged: 3
}

// Push reports whether the key was new. Pushing an existing key changes its
// priority and keeps the priority queue ordered.
func ExampleQueue_Push() {
	q := pqueue.New[string, int]()

	fmt.Println(q.Push("task", 5))
	fmt.Println(q.Push("other", 3))
	fmt.Println(q.Push("task", 1)) // existing: priority changes

	key, priority, _ := q.Peek()
	fmt.Println(key, priority, q.Len())
	// Output:
	// true
	// true
	// false
	// task 1 2
}

// PushN pushes several items with the semantics of Push: a key that is
// already queued, or repeated in the batch, has its priority changed.
func ExampleQueue_PushN() {
	q := pqueue.New[string, int]()
	q.Push("a", 5)
	q.PushN(
		pqueue.Item[string, int]{Key: "b", Priority: 2},
		pqueue.Item[string, int]{Key: "a", Priority: 1}, // updates "a"
		pqueue.Item[string, int]{Key: "c", Priority: 3},
	)

	fmt.Println(q.Sorted())
	// Output: [{a 1} {b 2} {c 3}]
}

// Pop removes and returns the top key and its priority. The boolean result
// reports whether the priority queue was non-empty; on an empty priority queue
// it is false and the returned key and priority are zero values.
func ExampleQueue_Pop() {
	q := pqueue.New[string, int]()
	q.Push("pear", 2)
	q.Push("apple", 1)

	key, priority, ok := q.Pop()
	fmt.Println(key, priority, ok)
	q.Pop()            // removes "pear"
	_, _, ok = q.Pop() // priority queue is now empty
	fmt.Println(ok)
	// Output:
	// apple 1 true
	// false
}

// PopN removes up to n keys from the top and returns them in priority order.
func ExampleQueue_PopN() {
	q := pqueue.From([]pqueue.Item[string, int]{
		{Key: "e", Priority: 5},
		{Key: "a", Priority: 1},
		{Key: "d", Priority: 4},
		{Key: "b", Priority: 2},
		{Key: "c", Priority: 3},
	})

	fmt.Println(q.PopN(3))
	fmt.Println("remaining:", q.Len())
	// Output:
	// [{a 1} {b 2} {c 3}]
	// remaining: 2
}

// Remove deletes a key and returns its priority, looking the key up by
// identity without scanning the priority queue.
func ExampleQueue_Remove() {
	q := pqueue.New[int, string]()
	q.Push(101, "b")
	q.Push(102, "a")
	q.Push(103, "c")

	priority, ok := q.Remove(102)
	fmt.Println(priority, ok)
	_, ok = q.Remove(999) // not present
	fmt.Println(ok)
	fmt.Println(q.Sorted())
	// Output:
	// a true
	// false
	// [{101 b} {103 c}]
}

// Clear empties the priority queue and releases its backing storage: capacity
// drops back to zero, and the priority queue stays usable.
func ExampleQueue_Clear() {
	q := pqueue.NewWithCap[string, int](8)
	q.PushN(
		pqueue.Item[string, int]{Key: "a", Priority: 2},
		pqueue.Item[string, int]{Key: "b", Priority: 1},
	)

	q.Clear()
	fmt.Println(q.Len(), q.Cap())

	q.Push("c", 3)
	key, priority, _ := q.Peek()
	fmt.Println(key, priority)
	// Output:
	// 0 0
	// c 3
}

// Reset, unlike Clear, empties the priority queue but keeps the allocated
// backing storage, so it can be reused without a new allocation.
func ExampleQueue_Reset() {
	q := pqueue.NewWithCap[string, int](8)
	q.PushN(
		pqueue.Item[string, int]{Key: "a", Priority: 2},
		pqueue.Item[string, int]{Key: "b", Priority: 1},
	)

	q.Reset()
	fmt.Println(q.Len(), q.Cap())
	// Output: 0 8
}

// Shrink copies the keys into a right-sized array, rebuilds the index, and
// frees the old storage immediately. Grow reserves capacity up front, and Clip
// trims spare capacity without copying.
func ExampleQueue_Shrink() {
	q := pqueue.NewWithCap[int, int](1024)
	q.PushN(
		pqueue.Item[int, int]{Key: 1, Priority: 1},
		pqueue.Item[int, int]{Key: 2, Priority: 2},
		pqueue.Item[int, int]{Key: 3, Priority: 3},
	)

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

// A nil *Queue[K, P] behaves as an empty priority queue for every read-only
// method and for removals, so it is safe to query. Mutating methods still
// require a priority queue created with a constructor such as New. Clone of a
// nil priority queue is nil, because there is no comparator to copy.
func Example_nilReceiver() {
	var q *pqueue.Queue[int, int] // nil, never initialized

	fmt.Println(q.Len(), q.IsEmpty(), q.Cap())

	key, priority, ok := q.Pop()
	fmt.Println(key, priority, ok)
	fmt.Println(q.Clone() == nil)
	// Output:
	// 0 true 0
	// 0 0 false
	// true
}

// A practical example: shortest paths with Dijkstra's algorithm. Pushing a key
// that is already queued changes its priority in place instead of adding a
// duplicate, so lowering a vertex's tentative distance is a single Push.
func Example_dijkstra() {
	type edge struct {
		to     string
		weight int
	}

	graph := map[string][]edge{
		"A": {{"B", 4}, {"C", 1}},
		"C": {{"B", 2}, {"D", 5}},
		"B": {{"D", 1}},
	}

	dist := map[string]int{"A": 0}
	q := pqueue.New[string, int]()
	q.Push("A", 0)

	for !q.IsEmpty() {
		v, d, _ := q.Pop()
		for _, e := range graph[v] {
			nd := d + e.weight
			if old, seen := dist[e.to]; !seen || nd < old {
				dist[e.to] = nd
				q.Push(e.to, nd) // inserts, or lowers the queued distance
			}
		}
	}

	for _, v := range []string{"A", "B", "C", "D"} {
		fmt.Println(v, dist[v])
	}
	// Output:
	// A 0
	// B 3
	// C 1
	// D 4
}
