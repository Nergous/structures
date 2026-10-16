package pqueue

import (
	"container/heap"
	"testing"

	"github.com/Nergous/structures/binheap"
)

// benchSize is the number of keys in containers prepared for benchmarks.
const benchSize = 1 << 12

// Package-level sinks prevent the compiler from optimizing away benchmarked
// results (and the work that produces them).
var (
	intSink   int
	keySink   int
	boolSink  bool
	sliceSink []Item[int, int]
	queueSink *Queue[int, int]
)

// filled returns a min priority queue holding the items of pseudoItems(n),
// heapified with [From].
func filled(n int) *Queue[int, int] {
	return From(pseudoItems(n))
}

// pseudo returns a cheap, deterministic, well-mixed value for index i, so the
// benchmarks exercise real sifts instead of already-sorted input.
func pseudo(i int) int {
	return int(uint32(i) * 2654435761 >> 8)
}

// pseudoSlice returns the values pseudo(0), ..., pseudo(n-1) in a new slice.
func pseudoSlice(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = pseudo(i)
	}

	return out
}

// pseudoItems returns the keys 0..n-1, each with the priority pseudo(key), in
// a new slice.
func pseudoItems(n int) []Item[int, int] {
	out := make([]Item[int, int], n)
	for i := range out {
		out[i] = Item[int, int]{
			Key:      i,
			Priority: pseudo(i),
		}
	}

	return out
}

// BenchmarkFrom measures building a priority queue from benchSize keys: one
// copy plus an O(n) heapify, plus the key index.
func BenchmarkFrom(b *testing.B) {
	items := pseudoItems(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		queueSink = From(items)
	}
}

// BenchmarkPeek measures reading the top key, which must stay O(1) and
// allocation-free.
func BenchmarkPeek(b *testing.B) {
	q := filled(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		keySink, intSink, boolSink = q.Peek()
	}
}

// BenchmarkContains measures lookups of present ("hit") and absent ("miss")
// keys.
func BenchmarkContains(b *testing.B) {
	q := filled(benchSize)

	b.Run("hit", func(b *testing.B) {
		b.ReportAllocs()

		i := 0
		for b.Loop() {
			k := i % benchSize
			boolSink = q.Contains(k)
			i++
		}
	})

	b.Run("miss", func(b *testing.B) {
		b.ReportAllocs()

		i := 0
		for b.Loop() {
			k := benchSize + i%benchSize
			boolSink = q.Contains(k)
			i++
		}
	})
}

// BenchmarkAll measures a full iteration over benchSize keys via the
// range-over-func iterator, confirming iteration is allocation-free.
func BenchmarkAll(b *testing.B) {
	q := filled(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		sum := 0
		for _, p := range q.All() {
			sum += p
		}

		intSink = sum
	}
}

// BenchmarkClone measures cloning a priority queue of benchSize keys, which
// allocates the new priority queue, its backing array, and its key index per
// call.
func BenchmarkClone(b *testing.B) {
	q := filled(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		queueSink = q.Clone()
	}
}

// BenchmarkSlice measures copying benchSize keys out to a plain slice, which
// allocates exactly one backing array per call.
func BenchmarkSlice(b *testing.B) {
	q := filled(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		sliceSink = q.Slice()
	}
}

// BenchmarkSorted measures the ordered snapshot of benchSize keys: one copy
// plus a sort, plus the conversion to items.
func BenchmarkSorted(b *testing.B) {
	q := filled(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		sliceSink = q.Sorted()
	}
}

// BenchmarkPush measures Push into a priority queue that grows from empty to
// benchSize, so the amortized cost of reallocations is included. When the
// priority queue reaches benchSize it is cleared with the timer stopped.
func BenchmarkPush(b *testing.B) {
	q := New[int, int]()

	b.ReportAllocs()

	i := 0
	for b.Loop() {
		if q.Len() == benchSize {
			b.StopTimer()
			q.Clear()
			b.StartTimer()
		}

		boolSink = q.Push(i, pseudo(i))
		i++
	}

	intSink = q.Len()
}

// BenchmarkPushExisting measures Push of a key that is already present, which
// changes its priority in place: the decrease-key step of Dijkstra's
// algorithm.
func BenchmarkPushExisting(b *testing.B) {
	q := filled(benchSize)

	b.ReportAllocs()

	i := 0
	for b.Loop() {
		k := i % benchSize
		boolSink = q.Push(k, pseudo(i+benchSize))
		i++
	}

	intSink = q.Len()
}

// BenchmarkPushN measures PushN in two workloads. In "small-into-large",
// batches of 8 new keys are sifted up into a priority queue that grows from
// benchSize to 2*benchSize keys; it is refilled with the timer stopped only
// when full. In "into-empty", benchSize keys are pushed into an empty priority
// queue; it is emptied with Reset inside the timed section, which is
// negligible next to the push.
func BenchmarkPushN(b *testing.B) {
	b.Run("small-into-large", func(b *testing.B) {
		const batch = 8

		base := pseudoItems(benchSize)
		extra := pseudoItems(2 * benchSize)[benchSize:]
		q := NewWithCap[int, int](2 * benchSize)
		q.PushN(base...)

		b.ReportAllocs()

		for b.Loop() {
			if q.Len() == 2*benchSize {
				b.StopTimer()
				// retain capacity so the next PushN does not reallocate
				q.Reset()
				q.PushN(base...)
				b.StartTimer()
			}

			off := q.Len() - benchSize
			q.PushN(extra[off : off+batch]...)
		}

		intSink = q.Len()
	})

	b.Run("into-empty", func(b *testing.B) {
		items := pseudoItems(benchSize)
		q := NewWithCap[int, int](benchSize)

		b.ReportAllocs()

		for b.Loop() {
			// retain capacity so the next PushN does not reallocate
			q.Reset()
			q.PushN(items...)
		}

		intSink = q.Len()
	})
}

// BenchmarkPop measures one Pop, which must not allocate. When the priority
// queue drains it is refilled to benchSize keys with the timer stopped.
func BenchmarkPop(b *testing.B) {
	items := pseudoItems(benchSize)
	q := NewWithCap[int, int](benchSize)

	b.ReportAllocs()

	for b.Loop() {
		if q.IsEmpty() {
			b.StopTimer()
			q.PushN(items...)
			b.StartTimer()
		}

		keySink, intSink, boolSink = q.Pop()
	}

	intSink = q.Len()
}

// BenchmarkPopN measures one PopN of a 128-key batch, which allocates the
// result slice once. When fewer than a batch remain the priority queue is
// refilled with the timer stopped.
func BenchmarkPopN(b *testing.B) {
	const batch = 128

	items := pseudoItems(benchSize)
	q := NewWithCap[int, int](benchSize)

	b.ReportAllocs()

	for b.Loop() {
		if q.Len() < batch {
			b.StopTimer()
			// retain capacity so the next PushN does not reallocate
			q.Reset()
			q.PushN(items...)
			b.StartTimer()
		}

		sliceSink = q.PopN(batch)
	}

	intSink = q.Len()
}

// BenchmarkRemove measures one Remove of a present key. When the priority
// queue drains it is refilled to benchSize keys with the timer stopped.
func BenchmarkRemove(b *testing.B) {
	items := pseudoItems(benchSize)
	q := NewWithCap[int, int](benchSize)

	b.ReportAllocs()

	i := 0
	for b.Loop() {
		if q.IsEmpty() {
			b.StopTimer()
			q.PushN(items...)
			b.StartTimer()
		}

		k := i % benchSize
		intSink, boolSink = q.Remove(k)
		i++
	}

	intSink = q.Len()
}

// BenchmarkPushPopChurn measures one Pop plus one Push of a new key at a
// steady size of benchSize, against binheap, which shows the cost of the key
// index, and against an indexed container/heap.
func BenchmarkPushPopChurn(b *testing.B) {
	b.Run("pqueue", func(b *testing.B) {
		q := filled(benchSize)

		b.ReportAllocs()

		i := benchSize
		for b.Loop() {
			keySink, intSink, boolSink = q.Pop()
			p := intSink + pseudo(i)
			boolSink = q.Push(i, p)
			i++
		}

		intSink = q.Len()
	})

	b.Run("binheap", func(b *testing.B) {
		h := binheap.From(pseudoSlice(benchSize))

		b.ReportAllocs()

		i := benchSize
		for b.Loop() {
			intSink, boolSink = h.Pop()
			v := intSink + pseudo(i)
			h.Push(v)
			i++
		}

		intSink = h.Len()
	})

	b.Run("container-heap", func(b *testing.B) {
		h := newIndexedHeap(benchSize)

		b.ReportAllocs()

		i := benchSize
		for b.Loop() {
			it := heap.Pop(h).(*indexedItem)
			delete(h.byKey, it.key)
			intSink = it.priority
			p := intSink + pseudo(i)
			h.push(i, p)
			i++
		}

		intSink = h.Len()
	})
}

// BenchmarkPushRemoveChurn measures one Remove plus one Push of the same key at
// a steady size of benchSize.
func BenchmarkPushRemoveChurn(b *testing.B) {
	q := filled(benchSize)

	b.ReportAllocs()

	i := 0
	for b.Loop() {
		k := i % benchSize
		intSink, boolSink = q.Remove(k)
		boolSink = q.Push(k, intSink)
		i++
	}

	intSink = q.Len()
}

// BenchmarkDijkstraChurn measures a shortest-path workload at a steady size of
// benchSize, where every Pop is followed by up to four decrease-key updates
// and one Push, against an indexed container/heap.
func BenchmarkDijkstraChurn(b *testing.B) {
	const updates = 4

	b.Run("pqueue", func(b *testing.B) {
		q := filled(benchSize)

		b.ReportAllocs()

		i := 0
		for b.Loop() {
			k, p, _ := q.Pop()
			for j := range updates {
				u := pseudo(i+j) % benchSize
				if old, ok := q.Priority(u); ok && p < old {
					boolSink = q.Push(u, old-1)
				}
			}

			next := p + pseudo(i)
			boolSink = q.Push(k, next)
			i++
		}

		intSink = q.Len()
	})

	b.Run("container-heap", func(b *testing.B) {
		h := newIndexedHeap(benchSize)

		b.ReportAllocs()

		i := 0
		for b.Loop() {
			it := heap.Pop(h).(*indexedItem)
			delete(h.byKey, it.key)
			for j := range updates {
				u := pseudo(i+j) % benchSize
				if e, ok := h.byKey[u]; ok && it.priority < e.priority {
					e.priority--
					heap.Fix(h, e.index)
				}
			}

			next := it.priority + pseudo(i)
			h.push(it.key, next)
			i++
		}

		intSink = h.Len()
	})
}

// indexedItem is one entry of indexedHeap: a key, its priority, and its
// current position in the heap.
type indexedItem struct {
	key, priority, index int
}

// indexedHeap is the textbook container/heap priority queue with a key index,
// the pattern pqueue replaces. See the container/heap package example.
type indexedHeap struct {
	items []*indexedItem
	byKey map[int]*indexedItem
}

// newIndexedHeap returns an indexedHeap holding the keys 0..n-1, each with the
// priority pseudo(key).
func newIndexedHeap(n int) *indexedHeap {
	h := &indexedHeap{byKey: make(map[int]*indexedItem, n)}
	for k, p := range pseudoSlice(n) {
		it := &indexedItem{key: k, priority: p, index: len(h.items)}
		h.items = append(h.items, it)
		h.byKey[k] = it
	}

	heap.Init(h)

	return h
}

// push adds the key k with the priority p and records it in the key index.
func (h *indexedHeap) push(k, p int) {
	it := &indexedItem{key: k, priority: p}
	h.byKey[k] = it
	heap.Push(h, it)
}

func (h *indexedHeap) Len() int { return len(h.items) }

func (h *indexedHeap) Less(i, j int) bool {
	return h.items[i].priority < h.items[j].priority
}

func (h *indexedHeap) Swap(i, j int) {
	h.items[i], h.items[j] = h.items[j], h.items[i]
	h.items[i].index = i
	h.items[j].index = j
}

func (h *indexedHeap) Push(x any) {
	it := x.(*indexedItem)
	it.index = len(h.items)
	h.items = append(h.items, it)
}

func (h *indexedHeap) Pop() any {
	old := h.items
	n := len(old)
	it := old[n-1]
	old[n-1] = nil
	h.items = old[:n-1]
	it.index = -1

	return it
}
