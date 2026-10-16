package queue

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
	queueSink *Queue[int]
)

// filled returns a queue holding the elements 0..n-1 in insertion order.
func filled(n int) *Queue[int] {
	q := NewWithCap[int](n)
	for i := range n {
		q.Push(i)
	}

	return q
}

// BenchmarkPeek measures reading the front element, which must stay O(1) and
// allocation-free.
func BenchmarkPeek(b *testing.B) {
	q := filled(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		intSink, boolSink = q.Peek()
	}
}

// BenchmarkAll measures a full iteration over benchSize elements via the
// range-over-func iterator, confirming iteration is allocation-free.
func BenchmarkAll(b *testing.B) {
	q := filled(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		sum := 0
		for v := range q.All() {
			sum += v
		}

		intSink = sum
	}
}

// BenchmarkClone measures cloning a queue of benchSize elements, which
// allocates the new queue and its backing buffer per call.
func BenchmarkClone(b *testing.B) {
	q := filled(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		queueSink = q.Clone()
	}
}

// BenchmarkSlice measures copying benchSize elements out to a plain slice,
// which allocates exactly one backing array per call.
func BenchmarkSlice(b *testing.B) {
	q := filled(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		sliceSink = q.Slice()
	}
}

// BenchmarkPush measures Push into a queue that grows from empty to benchSize,
// so the amortized cost of reallocations is included. When the queue reaches
// benchSize it is cleared with the timer stopped.
func BenchmarkPush(b *testing.B) {
	q := New[int]()

	b.ReportAllocs()

	i := 0
	for b.Loop() {
		if q.Len() == benchSize {
			b.StopTimer()
			q.Clear()
			b.StartTimer()
		}

		q.Push(i)
		i++
	}

	intSink = q.Len()
}

// BenchmarkPushN measures one PushN of a 256-element batch into a queue with
// room for 1024 batches, so no reallocation happens. When the queue is full it
// is reset with the timer stopped.
func BenchmarkPushN(b *testing.B) {
	const (
		batch  = 256
		rounds = 1024
	)

	vs := filled(batch).Slice()
	q := NewWithCap[int](rounds * batch)

	b.ReportAllocs()

	for b.Loop() {
		if q.Len() == rounds*batch {
			b.StopTimer()
			// retain capacity so the next PushN does not reallocate
			q.Reset()
			b.StartTimer()
		}

		q.PushN(vs...)
	}

	intSink = q.Len()
}

// BenchmarkPop measures one Pop, which must not allocate. When the queue
// drains it is refilled to benchSize elements with the timer stopped.
func BenchmarkPop(b *testing.B) {
	vs := filled(benchSize).Slice()
	q := NewWithCap[int](benchSize)

	b.ReportAllocs()

	for b.Loop() {
		if q.IsEmpty() {
			b.StopTimer()
			q.PushN(vs...)
			b.StartTimer()
		}

		intSink, boolSink = q.Pop()
	}

	intSink = q.Len()
}

// BenchmarkPopN measures one PopN of a 128-element batch, which allocates the
// result slice once. When fewer than a batch remain the queue is refilled with
// the timer stopped.
func BenchmarkPopN(b *testing.B) {
	const batch = 128

	vs := filled(benchSize).Slice()
	q := NewWithCap[int](benchSize)

	b.ReportAllocs()

	for b.Loop() {
		if q.Len() < batch {
			b.StopTimer()
			// retain capacity so the next PushN does not reallocate
			q.Reset()
			q.PushN(vs...)
			b.StartTimer()
		}

		sliceSink = q.PopN(batch)
	}

	intSink = q.Len()
}

// BenchmarkPushPopChurn measures one Pop plus one Push at a steady size of 16
// elements, which must not allocate for the queue, against container/list.
func BenchmarkPushPopChurn(b *testing.B) {
	const live = 16

	b.Run("queue", func(b *testing.B) {
		q := filled(live)

		b.ReportAllocs()

		i := live
		for b.Loop() {
			intSink, boolSink = q.Pop()
			q.Push(i)
			i++
		}

		intSink = q.Len()
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
			i++
		}

		intSink = l.Len()
	})
}
