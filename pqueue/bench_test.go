package pqueue

import (
	"container/heap"
	"testing"

	"github.com/Nergous/structures/binheap"
)

// Package-level sinks prevent the compiler from optimizing away benchmarked
// results (and the work that produces them).
var (
	intSink  int
	boolSink bool
)

// benchSize is the steady-state queue size for benchmarks that keep the queue
// length constant.
const benchSize = 1 << 12

// pseudo returns a deterministic pseudo-random sequence, so benchmark runs are
// comparable across packages and commits.
func pseudo(n int) []int {
	s := make([]int, n)
	x := uint32(2463534242)
	for i := range s {
		x ^= x << 13
		x ^= x >> 17
		x ^= x << 5
		s[i] = int(x % 1_000_000)
	}
	return s
}

func filled(n int) *PriorityQueue[int, int] {
	q := NewWithCap[int, int](n)
	for k, p := range pseudo(n) {
		q.Push(k, p)
	}
	return q
}

// BenchmarkPush measures inserting new keys into a growing queue.
func BenchmarkPush(b *testing.B) {
	ps := pseudo(benchSize)
	b.ReportAllocs()

	q := NewWithCap[int, int](benchSize)
	k := 0
	for b.Loop() {
		if q.Len() == benchSize {
			b.StopTimer()
			q.Reset()
			b.StartTimer()
		}
		boolSink = q.Push(k, ps[k%benchSize])
		k++
	}
}

// BenchmarkPushUpdate measures changing the priority of keys already queued,
// the decrease-key step of Dijkstra's algorithm.
func BenchmarkPushUpdate(b *testing.B) {
	q := filled(benchSize)
	ps := pseudo(2 * benchSize)[benchSize:]
	b.ReportAllocs()

	i := 0
	for b.Loop() {
		boolSink = q.Push(i%benchSize, ps[i%benchSize])
		i++
	}
}

// BenchmarkPushPopCycle keeps the queue at benchSize by pushing a new key for
// every popped one, and compares against binheap on the same workload, which
// shows the cost of keyed lookup.
func BenchmarkPushPopCycle(b *testing.B) {
	ps := pseudo(benchSize)

	b.Run("pqueue", func(b *testing.B) {
		q := filled(benchSize)
		b.ReportAllocs()
		k := benchSize
		for b.Loop() {
			_, p, _ := q.Pop()
			q.Push(k, p+ps[k%benchSize])
			k++
		}
		intSink = q.Len()
	})

	b.Run("binheap", func(b *testing.B) {
		h := binheap.From(ps)
		b.ReportAllocs()
		k := benchSize
		for b.Loop() {
			v, _ := h.Pop()
			h.Push(v + ps[k%benchSize])
			k++
		}
		intSink = h.Len()
	})

	b.Run("container-heap", func(b *testing.B) {
		h := newIndexedHeap(benchSize)
		b.ReportAllocs()
		k := benchSize
		for b.Loop() {
			it := heap.Pop(h).(*indexedItem)
			delete(h.byKey, it.key)
			h.push(k, it.priority+ps[k%benchSize])
			k++
		}
		intSink = h.Len()
	})
}

// BenchmarkRemove measures removing a key by identity and pushing it back.
func BenchmarkRemove(b *testing.B) {
	q := filled(benchSize)
	b.ReportAllocs()

	i := 0
	for b.Loop() {
		k := i % benchSize
		p, _ := q.Remove(k)
		q.Push(k, p)
		i++
	}
}

// BenchmarkDijkstraMix models a shortest-path search: every pop is followed by
// several decrease-key updates, compared against the hand-written indexed
// container/heap that pqueue replaces.
func BenchmarkDijkstraMix(b *testing.B) {
	const updates = 4
	ps := pseudo(benchSize)

	b.Run("pqueue", func(b *testing.B) {
		q := filled(benchSize)
		b.ReportAllocs()
		i := 0
		for b.Loop() {
			k, p, _ := q.Pop()
			for j := range updates {
				u := ps[(i+j)%benchSize] % benchSize
				if old, ok := q.Priority(u); ok && p < old {
					q.Push(u, old-1)
				}
			}
			q.Push(k, p+ps[i%benchSize])
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
				u := ps[(i+j)%benchSize] % benchSize
				if e, ok := h.byKey[u]; ok && it.priority < e.priority {
					e.priority--
					heap.Fix(h, e.index)
				}
			}
			h.push(it.key, it.priority+ps[i%benchSize])
			i++
		}
		intSink = h.Len()
	})
}

// indexedHeap is the textbook container/heap priority queue with a key index,
// the pattern pqueue replaces. See the container/heap package example.
type indexedItem struct {
	key, priority, index int
}

type indexedHeap struct {
	items []*indexedItem
	byKey map[int]*indexedItem
}

func newIndexedHeap(n int) *indexedHeap {
	h := &indexedHeap{byKey: make(map[int]*indexedItem, n)}
	for k, p := range pseudo(n) {
		it := &indexedItem{key: k, priority: p, index: len(h.items)}
		h.items = append(h.items, it)
		h.byKey[k] = it
	}
	heap.Init(h)
	return h
}

func (h *indexedHeap) push(k, p int) {
	it := &indexedItem{key: k, priority: p}
	h.byKey[k] = it
	heap.Push(h, it)
}

func (h *indexedHeap) Len() int           { return len(h.items) }
func (h *indexedHeap) Less(i, j int) bool { return h.items[i].priority < h.items[j].priority }
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
