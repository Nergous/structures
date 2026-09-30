package binheap

import (
	"container/heap"
	"testing"

	"github.com/Nergous/structures/internal/heapcore"
)

// Package-level sinks prevent the compiler from optimizing away benchmarked
// results (and the work that produces them).
var (
	intSink   int
	boolSink  bool
	sliceSink []int
	heapSink  *Heap[int]
)

// benchSize is the steady-state heap size for benchmarks that keep the heap
// length constant.
const benchSize = 1 << 12

// pseudo returns a cheap, deterministic, well-mixed value for index i, so the
// benchmarks exercise real sifts instead of already-sorted input.
func pseudo(i int) int {
	return int(uint32(i) * 2654435761 >> 8)
}

func pseudoSlice(n int) []int {
	s := make([]int, n)
	for i := range s {
		s[i] = pseudo(i)
	}
	return s
}

// BenchmarkPush measures the amortized cost of a single Push, including the
// occasional reallocation as the backing array grows. The heap grows across the
// whole run, so reallocations are amortized honestly over every push.
func BenchmarkPush(b *testing.B) {
	b.ReportAllocs()

	h := New[int]()
	i := 0
	for b.Loop() {
		h.Push(pseudo(i))
		i++
	}
	intSink = h.Len()
}

// BenchmarkPushN measures a bulk push in both of its strategies: a small batch
// into a large heap (sift-up per element) and a batch into an empty heap (full
// rebuild). The heap is restored outside the timed section so every iteration
// starts from the same state.
func BenchmarkPushN(b *testing.B) {
	cases := []struct {
		name  string
		base  int
		batch int
	}{
		{name: "small-into-large", base: benchSize, batch: 8},
		{name: "into-empty", base: 0, batch: benchSize},
	}

	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			base := pseudoSlice(c.base)
			vs := pseudoSlice(c.base + c.batch)[c.base:]

			b.ReportAllocs()

			h := NewWithCap[int](c.base + c.batch)
			for b.Loop() {
				b.StopTimer()
				h.Reset()
				h.data = append(h.data, base...)
				heapcore.Heapify(h.data, h.compare)
				b.StartTimer()

				h.PushN(vs...)
			}
			intSink = h.Len()
		})
	}
}

// BenchmarkPop measures Pop in isolation. Refilling happens outside the timed
// section: whenever the heap drains, a fresh chunk is loaded with the timer
// stopped, so only the Pop calls are measured.
func BenchmarkPop(b *testing.B) {
	chunk := pseudoSlice(benchSize)

	b.ReportAllocs()

	h := NewWithCap[int](benchSize)
	for b.Loop() {
		if h.IsEmpty() {
			b.StopTimer()
			h.PushN(chunk...)
			b.StartTimer()
		}
		intSink, boolSink = h.Pop()
	}
}

// BenchmarkPushPop measures the combined push-and-pop at a steady heap size.
// Values keep increasing, so each call replaces the root and sifts it down: the
// expensive path.
func BenchmarkPushPop(b *testing.B) {
	h := From(pseudoSlice(benchSize))

	b.ReportAllocs()
	i := 0
	for b.Loop() {
		intSink = h.PushPop(i + 1<<30)
		i++
	}
}

// BenchmarkReplace measures Replace at a steady heap size in a scheduler-like
// workload: the popped deadline is pushed back later by a varying interval, so
// the new root sifts part of the way down.
func BenchmarkReplace(b *testing.B) {
	h := From(pseudoSlice(benchSize))

	b.ReportAllocs()
	last := 0
	i := 0
	for b.Loop() {
		last, boolSink = h.Replace(last + pseudo(i)%(1<<16))
		i++
	}
	intSink = last
}

// BenchmarkPeek measures reading the least element, which must stay O(1) and
// allocation-free.
func BenchmarkPeek(b *testing.B) {
	h := From(pseudoSlice(benchSize))

	b.ReportAllocs()
	for b.Loop() {
		intSink, boolSink = h.Peek()
	}
}

// BenchmarkFrom measures building a heap from a slice: one copy plus an O(n)
// heapify.
func BenchmarkFrom(b *testing.B) {
	s := pseudoSlice(benchSize)

	b.ReportAllocs()
	for b.Loop() {
		heapSink = From(s)
	}
}

// BenchmarkContains measures the linear scan used by Contains in the worst
// case, a miss.
func BenchmarkContains(b *testing.B) {
	h := From(pseudoSlice(benchSize))

	b.ReportAllocs()
	for b.Loop() {
		boolSink = h.Contains(-1)
	}
}

// BenchmarkAll measures a full iteration via the range-over-func iterator,
// confirming iteration is allocation-free.
func BenchmarkAll(b *testing.B) {
	h := From(pseudoSlice(benchSize))

	b.ReportAllocs()
	for b.Loop() {
		sum := 0
		for v := range h.All() {
			sum += v
		}
		intSink = sum
	}
}

// BenchmarkSlice measures copying the heap out to a plain slice, which
// allocates exactly one backing array per call.
func BenchmarkSlice(b *testing.B) {
	h := From(pseudoSlice(benchSize))

	b.ReportAllocs()
	for b.Loop() {
		sliceSink = h.Slice()
	}
}

// BenchmarkSorted measures the ordered snapshot: one copy plus a sort.
func BenchmarkSorted(b *testing.B) {
	h := From(pseudoSlice(benchSize))

	b.ReportAllocs()
	for b.Loop() {
		sliceSink = h.Sorted()
	}
}

// intHeap adapts a slice to container/heap for the comparison benchmark.
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

// BenchmarkPushPopCycle compares one Push plus one Pop at a steady heap size
// against the standard library's container/heap, which boxes values in any and
// dispatches through an interface.
func BenchmarkPushPopCycle(b *testing.B) {
	b.Run("binheap", func(b *testing.B) {
		h := From(pseudoSlice(benchSize))

		b.ReportAllocs()
		i := 0
		for b.Loop() {
			h.Push(pseudo(i))
			intSink, boolSink = h.Pop()
			i++
		}
	})

	b.Run("container-heap", func(b *testing.B) {
		h := intHeap(pseudoSlice(benchSize))
		heap.Init(&h)

		b.ReportAllocs()
		i := 0
		for b.Loop() {
			heap.Push(&h, pseudo(i))
			intSink = heap.Pop(&h).(int)
			i++
		}
	})
}
