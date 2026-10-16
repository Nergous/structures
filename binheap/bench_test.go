package binheap

import (
	"container/heap"
	"testing"
)

// benchSize is the number of elements in containers prepared for benchmarks.
const benchSize = 1 << 12

// Package-level sinks prevent the compiler from optimizing away benchmarked
// results (and the work that produces them).
var (
	intSink   int
	boolSink  bool
	sliceSink []int
	heapSink  *Heap[int]
)

// filled returns a min-heap holding the values of pseudoSlice(n), heapified
// with [From].
func filled(n int) *Heap[int] {
	return From(pseudoSlice(n))
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

// BenchmarkFrom measures building a heap from benchSize elements: one copy
// plus an O(n) heapify.
func BenchmarkFrom(b *testing.B) {
	vs := pseudoSlice(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		heapSink = From(vs)
	}
}

// BenchmarkPeek measures reading the top element, which must stay O(1) and
// allocation-free.
func BenchmarkPeek(b *testing.B) {
	h := filled(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		intSink, boolSink = h.Peek()
	}
}

// BenchmarkContains measures lookups of present ("hit") and absent ("miss")
// elements. Contains scans the heap linearly, so "hit" averages over every
// element and "miss" is the worst case.
func BenchmarkContains(b *testing.B) {
	vs := pseudoSlice(benchSize)
	h := From(vs)

	b.Run("hit", func(b *testing.B) {
		b.ReportAllocs()

		i := 0
		for b.Loop() {
			v := vs[i%benchSize]
			boolSink = h.Contains(v)
			i++
		}
	})

	b.Run("miss", func(b *testing.B) {
		b.ReportAllocs()

		i := 0
		for b.Loop() {
			v := -1 - i%benchSize
			boolSink = h.Contains(v)
			i++
		}
	})
}

// BenchmarkAll measures a full iteration over benchSize elements via the
// range-over-func iterator, confirming iteration is allocation-free.
func BenchmarkAll(b *testing.B) {
	h := filled(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		sum := 0
		for v := range h.All() {
			sum += v
		}

		intSink = sum
	}
}

// BenchmarkClone measures cloning a heap of benchSize elements, which
// allocates the new heap and its backing array per call.
func BenchmarkClone(b *testing.B) {
	h := filled(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		heapSink = h.Clone()
	}
}

// BenchmarkSlice measures copying benchSize elements out to a plain slice,
// which allocates exactly one backing array per call.
func BenchmarkSlice(b *testing.B) {
	h := filled(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		sliceSink = h.Slice()
	}
}

// BenchmarkSorted measures the ordered snapshot of benchSize elements: one
// copy plus a sort.
func BenchmarkSorted(b *testing.B) {
	h := filled(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		sliceSink = h.Sorted()
	}
}

// BenchmarkPush measures Push into a heap that grows from empty to benchSize,
// so the amortized cost of reallocations is included. When the heap reaches
// benchSize it is cleared with the timer stopped.
func BenchmarkPush(b *testing.B) {
	h := New[int]()

	b.ReportAllocs()

	i := 0
	for b.Loop() {
		if h.Len() == benchSize {
			b.StopTimer()
			h.Clear()
			b.StartTimer()
		}

		h.Push(pseudo(i))
		i++
	}

	intSink = h.Len()
}

// BenchmarkPushN measures PushN in two workloads. In "small-into-large",
// batches of 8 elements are sifted up into a heap that grows from benchSize to
// 2*benchSize elements; it is refilled with the timer stopped only when full.
// In "into-empty", benchSize elements are pushed into an empty heap, which
// PushN rebuilds with one heapify; it is emptied with Reset inside the timed
// section, which is negligible next to the push.
func BenchmarkPushN(b *testing.B) {
	b.Run("small-into-large", func(b *testing.B) {
		const batch = 8

		base := pseudoSlice(benchSize)
		extra := pseudoSlice(2 * benchSize)[benchSize:]
		h := NewWithCap[int](2 * benchSize)
		h.PushN(base...)

		b.ReportAllocs()

		for b.Loop() {
			if h.Len() == 2*benchSize {
				b.StopTimer()
				// retain capacity so the next PushN does not reallocate
				h.Reset()
				h.PushN(base...)
				b.StartTimer()
			}

			off := h.Len() - benchSize
			h.PushN(extra[off : off+batch]...)
		}

		intSink = h.Len()
	})

	b.Run("into-empty", func(b *testing.B) {
		vs := pseudoSlice(benchSize)
		h := NewWithCap[int](benchSize)

		b.ReportAllocs()

		for b.Loop() {
			// retain capacity so the next PushN does not reallocate
			h.Reset()
			h.PushN(vs...)
		}

		intSink = h.Len()
	})
}

// BenchmarkPop measures one Pop, which must not allocate. When the heap drains
// it is refilled to benchSize elements with the timer stopped.
func BenchmarkPop(b *testing.B) {
	vs := pseudoSlice(benchSize)
	h := NewWithCap[int](benchSize)

	b.ReportAllocs()

	for b.Loop() {
		if h.IsEmpty() {
			b.StopTimer()
			h.PushN(vs...)
			b.StartTimer()
		}

		intSink, boolSink = h.Pop()
	}

	intSink = h.Len()
}

// BenchmarkPopN measures one PopN of a 128-element batch, which allocates the
// result slice once. When fewer than a batch remain the heap is refilled with
// the timer stopped.
func BenchmarkPopN(b *testing.B) {
	const batch = 128

	vs := pseudoSlice(benchSize)
	h := NewWithCap[int](benchSize)

	b.ReportAllocs()

	for b.Loop() {
		if h.Len() < batch {
			b.StopTimer()
			// retain capacity so the next PushN does not reallocate
			h.Reset()
			h.PushN(vs...)
			b.StartTimer()
		}

		sliceSink = h.PopN(batch)
	}

	intSink = h.Len()
}

// BenchmarkPushPop measures PushPop at a steady size of benchSize. Values keep
// increasing, so each call replaces the root and sifts it down: the expensive
// path.
func BenchmarkPushPop(b *testing.B) {
	h := filled(benchSize)

	b.ReportAllocs()

	i := 0
	for b.Loop() {
		v := i + 1<<30
		intSink = h.PushPop(v)
		i++
	}

	intSink = h.Len()
}

// BenchmarkReplace measures Replace at a steady size of benchSize in a
// scheduler-like workload: the popped deadline is pushed back later by a
// varying interval, so the new root sifts part of the way down.
func BenchmarkReplace(b *testing.B) {
	h := filled(benchSize)

	b.ReportAllocs()

	last := 0
	i := 0
	for b.Loop() {
		v := last + pseudo(i)%(1<<16)
		last, boolSink = h.Replace(v)
		i++
	}

	intSink = h.Len()
}

// BenchmarkRemove measures one Remove of a present element: a linear scan plus
// restoring the heap order. When the heap drains it is refilled to benchSize
// elements with the timer stopped.
func BenchmarkRemove(b *testing.B) {
	vs := pseudoSlice(benchSize)
	h := NewWithCap[int](benchSize)

	b.ReportAllocs()

	i := 0
	for b.Loop() {
		if h.IsEmpty() {
			b.StopTimer()
			h.PushN(vs...)
			b.StartTimer()
		}

		v := vs[i%benchSize]
		boolSink = h.Remove(v)
		i++
	}

	intSink = h.Len()
}

// BenchmarkPushPopChurn measures one Pop plus one Push at a steady size of
// benchSize, against container/heap, which boxes values in any and dispatches
// through an interface.
func BenchmarkPushPopChurn(b *testing.B) {
	b.Run("binheap", func(b *testing.B) {
		h := filled(benchSize)

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
		h := intHeap(pseudoSlice(benchSize))
		heap.Init(&h)

		b.ReportAllocs()

		i := benchSize
		for b.Loop() {
			intSink = heap.Pop(&h).(int)
			v := intSink + pseudo(i)
			heap.Push(&h, v)
			i++
		}

		intSink = h.Len()
	})
}

// intHeap adapts a slice to container/heap for the comparison benchmarks.
type intHeap []int

func (h intHeap) Len() int           { return len(h) }
func (h intHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h intHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *intHeap) Push(x any)        { *h = append(*h, x.(int)) }
func (h *intHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]

	return x
}
