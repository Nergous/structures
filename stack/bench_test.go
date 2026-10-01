package stack

import "testing"

// Package-level sinks prevent the compiler from optimizing away benchmarked
// results (and the work that produces them).
var (
	intSink   int
	boolSink  bool
	sliceSink []int
)

// BenchmarkPush measures the amortized cost of a single Push, including the
// occasional reallocation as the backing array grows. The stack grows across
// the whole run, so the cost of those reallocations is amortized honestly over
// every push rather than hidden behind a one-time prealloc.
func BenchmarkPush(b *testing.B) {
	b.ReportAllocs()

	s := New[int]()
	i := 0
	for b.Loop() {
		s.Push(i)
		i++
	}
	// Keep the final size observable so the pushes are not dead code.
	intSink = s.Len()
}

// BenchmarkPushN measures a single bulk push of a fixed batch, the cheaper
// alternative to repeated Push since it grows the backing array at most once.
// The batch is built once outside the timed loop; the stack is reset (capacity
// retained) each iteration so we measure the bulk append, not repeated growth.
func BenchmarkPushN(b *testing.B) {
	const batch = 1024

	vs := make([]int, batch)
	for i := range vs {
		vs[i] = i
	}

	b.ReportAllocs()

	s := NewWithCap[int](batch)
	for b.Loop() {
		s.PushN(vs...)
		b.StopTimer()
		s.Reset() // retain capacity so the next PushN does not reallocate
		b.StartTimer()
	}
	intSink = s.Len()
}

// BenchmarkPop measures Pop in isolation. Refilling happens outside the timed
// section: whenever the stack drains we push a fresh chunk with the timer
// stopped, so only the Pop calls are measured.
func BenchmarkPop(b *testing.B) {
	const chunk = 1024

	b.ReportAllocs()

	s := NewWithCap[int](chunk)
	for b.Loop() {
		if s.IsEmpty() {
			b.StopTimer()
			for j := range chunk {
				s.Push(j)
			}
			b.StartTimer()
		}
		_, boolSink = s.Pop()
	}
}

// BenchmarkPopN measures draining fixed-size batches. Refilling happens outside
// the timed section so the benchmark focuses on batch removal and result-slice
// allocation.
func BenchmarkPopN(b *testing.B) {
	const batch = 128

	b.ReportAllocs()

	s := NewWithCap[int](batch)
	for b.Loop() {
		if s.Len() < batch {
			b.StopTimer()
			s.Reset()
			for j := range batch {
				s.Push(j)
			}
			b.StartTimer()
		}
		sliceSink = s.PopN(batch)
	}
}

// BenchmarkPeek measures the cost of reading the top element, which must stay
// O(1) and allocation-free.
func BenchmarkPeek(b *testing.B) {
	b.ReportAllocs()

	s := New[int]()
	s.PushN(1, 2, 3)

	for b.Loop() {
		intSink, boolSink = s.Peek()
	}
}

// BenchmarkAll measures a full iteration over the stack via the range-over-func
// iterator, confirming iteration is allocation-free.
func BenchmarkAll(b *testing.B) {
	const n = 1024

	s := NewWithCap[int](n)
	for i := range n {
		s.Push(i)
	}

	b.ReportAllocs()
	for b.Loop() {
		sum := 0
		for v := range s.All() {
			sum += v
		}
		intSink = sum
	}
}

// BenchmarkSlice measures copying the stack out to a plain slice, which
// allocates exactly one backing array per call.
func BenchmarkSlice(b *testing.B) {
	const n = 1024

	s := NewWithCap[int](n)
	for i := range n {
		s.Push(i)
	}

	b.ReportAllocs()
	for b.Loop() {
		sliceSink = s.Slice()
	}
}
