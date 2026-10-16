package stack

import "testing"

// benchSize is the number of elements in containers prepared for benchmarks.
const benchSize = 1 << 12

// Package-level sinks prevent the compiler from optimizing away benchmarked
// results (and the work that produces them).
var (
	intSink   int
	boolSink  bool
	sliceSink []int
	stackSink *Stack[int]
)

// filled returns a stack holding the elements 0..n-1 in insertion order.
func filled(n int) *Stack[int] {
	s := NewWithCap[int](n)
	for i := range n {
		s.Push(i)
	}

	return s
}

// BenchmarkPeek measures reading the top element, which must stay O(1) and
// allocation-free.
func BenchmarkPeek(b *testing.B) {
	s := filled(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		intSink, boolSink = s.Peek()
	}
}

// BenchmarkAll measures a full iteration over benchSize elements via the
// range-over-func iterator, confirming iteration is allocation-free.
func BenchmarkAll(b *testing.B) {
	s := filled(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		sum := 0
		for v := range s.All() {
			sum += v
		}

		intSink = sum
	}
}

// BenchmarkClone measures cloning a stack of benchSize elements, which
// allocates the new stack and its backing array per call.
func BenchmarkClone(b *testing.B) {
	s := filled(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		stackSink = s.Clone()
	}
}

// BenchmarkSlice measures copying benchSize elements out to a plain slice,
// which allocates exactly one backing array per call.
func BenchmarkSlice(b *testing.B) {
	s := filled(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		sliceSink = s.Slice()
	}
}

// BenchmarkPush measures Push into a stack that grows from empty to benchSize,
// so the amortized cost of reallocations is included, against a plain []int.
// When the stack reaches benchSize it is cleared with the timer stopped.
func BenchmarkPush(b *testing.B) {
	b.Run("stack", func(b *testing.B) {
		s := New[int]()

		b.ReportAllocs()

		i := 0
		for b.Loop() {
			if s.Len() == benchSize {
				b.StopTimer()
				s.Clear()
				b.StartTimer()
			}

			s.Push(i)
			i++
		}

		intSink = s.Len()
	})

	b.Run("slice", func(b *testing.B) {
		var data []int

		b.ReportAllocs()

		i := 0
		for b.Loop() {
			if len(data) == benchSize {
				b.StopTimer()
				data = nil
				b.StartTimer()
			}

			data = append(data, i)
			i++
		}

		intSink = len(data)
	})
}

// BenchmarkPushN measures one PushN of a 256-element batch into a stack with
// room for 1024 batches, so no reallocation happens. When the stack is full it
// is reset with the timer stopped.
func BenchmarkPushN(b *testing.B) {
	const (
		batch  = 256
		rounds = 1024
	)

	vs := filled(batch).Slice()
	s := NewWithCap[int](rounds * batch)

	b.ReportAllocs()

	for b.Loop() {
		if s.Len() == rounds*batch {
			b.StopTimer()
			// retain capacity so the next PushN does not reallocate
			s.Reset()
			b.StartTimer()
		}

		s.PushN(vs...)
	}

	intSink = s.Len()
}

// BenchmarkPop measures one Pop, which must not allocate. When the stack
// drains it is refilled to benchSize elements with the timer stopped, against
// a plain []int.
func BenchmarkPop(b *testing.B) {
	vs := filled(benchSize).Slice()

	b.Run("stack", func(b *testing.B) {
		s := NewWithCap[int](benchSize)

		b.ReportAllocs()

		for b.Loop() {
			if s.IsEmpty() {
				b.StopTimer()
				s.PushN(vs...)
				b.StartTimer()
			}

			intSink, boolSink = s.Pop()
		}

		intSink = s.Len()
	})

	b.Run("slice", func(b *testing.B) {
		data := make([]int, 0, benchSize)

		b.ReportAllocs()

		for b.Loop() {
			if len(data) == 0 {
				b.StopTimer()
				data = append(data, vs...)
				b.StartTimer()
			}

			top := len(data) - 1
			intSink = data[top]
			data = data[:top]
		}

		intSink = len(data)
	})
}

// BenchmarkPopN measures one PopN of a 128-element batch, which allocates the
// result slice once. When fewer than a batch remain the stack is refilled with
// the timer stopped.
func BenchmarkPopN(b *testing.B) {
	const batch = 128

	vs := filled(benchSize).Slice()
	s := NewWithCap[int](benchSize)

	b.ReportAllocs()

	for b.Loop() {
		if s.Len() < batch {
			b.StopTimer()
			// retain capacity so the next PushN does not reallocate
			s.Reset()
			s.PushN(vs...)
			b.StartTimer()
		}

		sliceSink = s.PopN(batch)
	}

	intSink = s.Len()
}

// BenchmarkPushPopChurn measures one Pop plus one Push at a steady size of 16
// elements, which must not allocate for the stack, against a plain []int.
func BenchmarkPushPopChurn(b *testing.B) {
	const live = 16

	b.Run("stack", func(b *testing.B) {
		s := filled(live)

		b.ReportAllocs()

		i := live
		for b.Loop() {
			intSink, boolSink = s.Pop()
			s.Push(i)
			i++
		}

		intSink = s.Len()
	})

	b.Run("slice", func(b *testing.B) {
		data := filled(live).Slice()

		b.ReportAllocs()

		i := live
		for b.Loop() {
			top := len(data) - 1
			intSink = data[top]
			data = data[:top]
			data = append(data, i)
			i++
		}

		intSink = len(data)
	})
}
