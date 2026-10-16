package deque

import (
	"container/list"
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
	dequeSink *Deque[int]
)

// filled returns a deque holding the elements 0..n-1 in insertion order.
func filled(n int) *Deque[int] {
	d := NewWithCap[int](n)
	for i := range n {
		d.PushBack(i)
	}

	return d
}

// BenchmarkFront measures reading the front element, which must stay O(1) and
// allocation-free.
func BenchmarkFront(b *testing.B) {
	d := filled(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		intSink, boolSink = d.Front()
	}
}

// BenchmarkBack measures reading the back element, which must stay O(1) and
// allocation-free.
func BenchmarkBack(b *testing.B) {
	d := filled(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		intSink, boolSink = d.Back()
	}
}

// BenchmarkAll measures a full iteration over benchSize elements via the
// range-over-func iterator, confirming iteration is allocation-free.
func BenchmarkAll(b *testing.B) {
	d := filled(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		sum := 0
		for v := range d.All() {
			sum += v
		}

		intSink = sum
	}
}

// BenchmarkBackward measures a full reverse iteration over benchSize elements
// via the range-over-func iterator, confirming iteration is allocation-free.
func BenchmarkBackward(b *testing.B) {
	d := filled(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		sum := 0
		for v := range d.Backward() {
			sum += v
		}

		intSink = sum
	}
}

// BenchmarkClone measures cloning a deque of benchSize elements, which
// allocates the new deque and its backing buffer per call.
func BenchmarkClone(b *testing.B) {
	d := filled(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		dequeSink = d.Clone()
	}
}

// BenchmarkSlice measures copying benchSize elements out to a plain slice,
// which allocates exactly one backing array per call.
func BenchmarkSlice(b *testing.B) {
	d := filled(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		sliceSink = d.Slice()
	}
}

// BenchmarkPushFront measures PushFront into a deque that grows from empty to
// benchSize, so the amortized cost of reallocations is included. When the
// deque reaches benchSize it is cleared with the timer stopped.
func BenchmarkPushFront(b *testing.B) {
	d := New[int]()

	b.ReportAllocs()

	i := 0
	for b.Loop() {
		if d.Len() == benchSize {
			b.StopTimer()
			d.Clear()
			b.StartTimer()
		}

		d.PushFront(i)
		i++
	}

	intSink = d.Len()
}

// BenchmarkPushBack measures PushBack into a deque that grows from empty to
// benchSize, so the amortized cost of reallocations is included. When the
// deque reaches benchSize it is cleared with the timer stopped.
func BenchmarkPushBack(b *testing.B) {
	d := New[int]()

	b.ReportAllocs()

	i := 0
	for b.Loop() {
		if d.Len() == benchSize {
			b.StopTimer()
			d.Clear()
			b.StartTimer()
		}

		d.PushBack(i)
		i++
	}

	intSink = d.Len()
}

// BenchmarkPushFrontN measures one PushFrontN of a 256-element batch into a
// deque with room for 1024 batches, so no reallocation happens. When the deque
// is full it is reset with the timer stopped.
func BenchmarkPushFrontN(b *testing.B) {
	const (
		batch  = 256
		rounds = 1024
	)

	vs := filled(batch).Slice()
	d := NewWithCap[int](rounds * batch)

	b.ReportAllocs()

	for b.Loop() {
		if d.Len() == rounds*batch {
			b.StopTimer()
			// retain capacity so the next PushFrontN does not reallocate
			d.Reset()
			b.StartTimer()
		}

		d.PushFrontN(vs...)
	}

	intSink = d.Len()
}

// BenchmarkPushBackN measures one PushBackN of a 256-element batch into a
// deque with room for 1024 batches, so no reallocation happens. When the deque
// is full it is reset with the timer stopped.
func BenchmarkPushBackN(b *testing.B) {
	const (
		batch  = 256
		rounds = 1024
	)

	vs := filled(batch).Slice()
	d := NewWithCap[int](rounds * batch)

	b.ReportAllocs()

	for b.Loop() {
		if d.Len() == rounds*batch {
			b.StopTimer()
			// retain capacity so the next PushBackN does not reallocate
			d.Reset()
			b.StartTimer()
		}

		d.PushBackN(vs...)
	}

	intSink = d.Len()
}

// BenchmarkPopFront measures one PopFront, which must not allocate. When the
// deque drains it is refilled to benchSize elements with the timer stopped.
func BenchmarkPopFront(b *testing.B) {
	vs := filled(benchSize).Slice()
	d := NewWithCap[int](benchSize)

	b.ReportAllocs()

	for b.Loop() {
		if d.IsEmpty() {
			b.StopTimer()
			d.PushBackN(vs...)
			b.StartTimer()
		}

		intSink, boolSink = d.PopFront()
	}

	intSink = d.Len()
}

// BenchmarkPopBack measures one PopBack, which must not allocate. When the
// deque drains it is refilled to benchSize elements with the timer stopped.
func BenchmarkPopBack(b *testing.B) {
	vs := filled(benchSize).Slice()
	d := NewWithCap[int](benchSize)

	b.ReportAllocs()

	for b.Loop() {
		if d.IsEmpty() {
			b.StopTimer()
			d.PushBackN(vs...)
			b.StartTimer()
		}

		intSink, boolSink = d.PopBack()
	}

	intSink = d.Len()
}

// BenchmarkPopFrontN measures one PopFrontN of a 128-element batch, which
// allocates the result slice once. When fewer than a batch remain the deque is
// refilled with the timer stopped.
func BenchmarkPopFrontN(b *testing.B) {
	const batch = 128

	vs := filled(benchSize).Slice()
	d := NewWithCap[int](benchSize)

	b.ReportAllocs()

	for b.Loop() {
		if d.Len() < batch {
			b.StopTimer()
			// retain capacity so the next PushBackN does not reallocate
			d.Reset()
			d.PushBackN(vs...)
			b.StartTimer()
		}

		sliceSink = d.PopFrontN(batch)
	}

	intSink = d.Len()
}

// BenchmarkPopBackN measures one PopBackN of a 128-element batch, which
// allocates the result slice once. When fewer than a batch remain the deque is
// refilled with the timer stopped.
func BenchmarkPopBackN(b *testing.B) {
	const batch = 128

	vs := filled(benchSize).Slice()
	d := NewWithCap[int](benchSize)

	b.ReportAllocs()

	for b.Loop() {
		if d.Len() < batch {
			b.StopTimer()
			// retain capacity so the next PushBackN does not reallocate
			d.Reset()
			d.PushBackN(vs...)
			b.StartTimer()
		}

		sliceSink = d.PopBackN(batch)
	}

	intSink = d.Len()
}

// BenchmarkPushPopFrontBackChurn measures one PopFront plus one PushBack and
// one PopBack plus one PushFront at a steady size of 16 elements, which must
// not allocate for the deque, against container/list.
func BenchmarkPushPopFrontBackChurn(b *testing.B) {
	const live = 16

	b.Run("deque", func(b *testing.B) {
		d := filled(live)

		b.ReportAllocs()

		i := live
		for b.Loop() {
			intSink, boolSink = d.PopFront()
			d.PushBack(i)
			intSink, boolSink = d.PopBack()
			d.PushFront(i)
			i++
		}

		intSink = d.Len()
	})

	b.Run("container-list", func(b *testing.B) {
		l := list.New()
		for i := range live {
			l.PushBack(i)
		}

		b.ReportAllocs()

		i := live
		for b.Loop() {
			intSink = l.Remove(l.Front()).(int)
			l.PushBack(i)
			intSink = l.Remove(l.Back()).(int)
			l.PushFront(i)
			i++
		}

		intSink = l.Len()
	})
}
