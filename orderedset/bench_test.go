package orderedset

import (
	"fmt"
	"testing"

	"github.com/Nergous/structures/set"
)

// benchSize is the number of values in containers prepared for benchmarks.
const benchSize = 1 << 12

// Package-level sinks prevent the compiler from optimizing away benchmarked
// results (and the work that produces them).
var (
	intSink         int
	boolSink        bool
	sliceSink       []int
	setSink         *Set[int]
	setBaselineSink *set.Set[int]
)

// filled returns a set holding the values 0..n-1 in insertion order.
func filled(n int) *Set[int] {
	s := NewWithCap[int](n)
	for i := range n {
		s.Add(i)
	}

	return s
}

// filledSet returns a set.Set holding the values 0..n-1.
func filledSet(n int) *set.Set[int] {
	s := set.NewWithCap[int](n)
	for i := range n {
		s.Add(i)
	}

	return s
}

// overlapping returns two sets of n values each that share half of their
// values: 0..n-1 and n/2..n/2+n-1.
func overlapping(n int) (*Set[int], *Set[int]) {
	a := filled(n)
	c := NewWithCap[int](n)
	for i := range n {
		c.Add(i + n/2)
	}

	return a, c
}

// overlappingSet returns the operands of overlapping as set.Set values for the
// set baseline.
func overlappingSet(n int) (*set.Set[int], *set.Set[int]) {
	a := filledSet(n)
	c := set.NewWithCap[int](n)
	for i := range n {
		c.Add(i + n/2)
	}

	return a, c
}

// BenchmarkFrom measures building a set from benchSize values in which every
// value appears twice, against set. From is Of over the same slice, so this
// covers both constructors.
func BenchmarkFrom(b *testing.B) {
	vs := make([]int, benchSize)
	for i := range vs {
		vs[i] = i / 2
	}

	b.Run("orderedset", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			setSink = From(vs)
		}
	})

	b.Run("set", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			setBaselineSink = set.From(vs)
		}
	})
}

// BenchmarkContains measures lookups of present ("hit") and absent ("miss")
// values, against set.
func BenchmarkContains(b *testing.B) {
	b.Run("orderedset", func(b *testing.B) {
		s := filled(benchSize)

		b.Run("hit", func(b *testing.B) {
			b.ReportAllocs()

			i := 0
			for b.Loop() {
				v := i % benchSize
				boolSink = s.Contains(v)
				i++
			}
		})

		b.Run("miss", func(b *testing.B) {
			b.ReportAllocs()

			i := 0
			for b.Loop() {
				v := benchSize + i%benchSize
				boolSink = s.Contains(v)
				i++
			}
		})
	})

	b.Run("set", func(b *testing.B) {
		s := filledSet(benchSize)

		b.Run("hit", func(b *testing.B) {
			b.ReportAllocs()

			i := 0
			for b.Loop() {
				v := i % benchSize
				boolSink = s.Contains(v)
				i++
			}
		})

		b.Run("miss", func(b *testing.B) {
			b.ReportAllocs()

			i := 0
			for b.Loop() {
				v := benchSize + i%benchSize
				boolSink = s.Contains(v)
				i++
			}
		})
	})
}

// BenchmarkFirst measures reading the oldest value, which must stay O(1) and
// allocation-free.
func BenchmarkFirst(b *testing.B) {
	s := filled(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		intSink, boolSink = s.First()
	}
}

// BenchmarkLast measures reading the newest value, which must stay O(1) and
// allocation-free.
func BenchmarkLast(b *testing.B) {
	s := filled(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		intSink, boolSink = s.Last()
	}
}

// BenchmarkAll measures a full iteration over benchSize values via the
// range-over-func iterator, confirming iteration is allocation-free, against
// set.
func BenchmarkAll(b *testing.B) {
	b.Run("orderedset", func(b *testing.B) {
		s := filled(benchSize)

		b.ReportAllocs()

		for b.Loop() {
			sum := 0
			for v := range s.All() {
				sum += v
			}

			intSink = sum
		}
	})

	b.Run("set", func(b *testing.B) {
		s := filledSet(benchSize)

		b.ReportAllocs()

		for b.Loop() {
			sum := 0
			for v := range s.All() {
				sum += v
			}

			intSink = sum
		}
	})
}

// BenchmarkBackward measures a full reverse iteration over benchSize values
// via the range-over-func iterator, confirming iteration is allocation-free.
func BenchmarkBackward(b *testing.B) {
	s := filled(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		sum := 0
		for v := range s.Backward() {
			sum += v
		}

		intSink = sum
	}
}

// BenchmarkClone measures cloning a set of benchSize values, which allocates
// the new set, its backing array, and its index map per call, against set.
func BenchmarkClone(b *testing.B) {
	b.Run("orderedset", func(b *testing.B) {
		s := filled(benchSize)

		b.ReportAllocs()

		for b.Loop() {
			setSink = s.Clone()
		}
	})

	b.Run("set", func(b *testing.B) {
		s := filledSet(benchSize)

		b.ReportAllocs()

		for b.Loop() {
			setBaselineSink = s.Clone()
		}
	})
}

// BenchmarkSlice measures copying benchSize values out to a plain slice, which
// allocates exactly one backing array per call.
func BenchmarkSlice(b *testing.B) {
	s := filled(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		sliceSink = s.Slice()
	}
}

// BenchmarkAdd measures Add into a set that grows from empty to benchSize, so
// the amortized cost of reallocations is included, against set. When the set
// reaches benchSize it is cleared with the timer stopped.
func BenchmarkAdd(b *testing.B) {
	b.Run("orderedset", func(b *testing.B) {
		s := New[int]()

		b.ReportAllocs()

		i := 0
		for b.Loop() {
			if s.Len() == benchSize {
				b.StopTimer()
				s.Clear()
				b.StartTimer()
			}

			boolSink = s.Add(i)
			i++
		}

		intSink = s.Len()
	})

	b.Run("set", func(b *testing.B) {
		s := set.New[int]()

		b.ReportAllocs()

		i := 0
		for b.Loop() {
			if s.Len() == benchSize {
				b.StopTimer()
				s.Clear()
				b.StartTimer()
			}

			boolSink = s.Add(i)
			i++
		}

		intSink = s.Len()
	})
}

// BenchmarkAddExisting measures Add of a value that is already present, which
// leaves the set unchanged and does not allocate.
func BenchmarkAddExisting(b *testing.B) {
	s := filled(benchSize)

	b.ReportAllocs()

	i := 0
	for b.Loop() {
		v := i % benchSize
		boolSink = s.Add(v)
		i++
	}

	intSink = s.Len()
}

// BenchmarkAddN measures one AddN of a 256-value batch of new values into a
// set with room for benchSize values, so no reallocation happens. When the set
// is full it is reset with the timer stopped.
func BenchmarkAddN(b *testing.B) {
	const batch = 256

	vs := filled(benchSize).Slice()
	s := NewWithCap[int](benchSize)

	b.ReportAllocs()

	for b.Loop() {
		if s.Len() == benchSize {
			b.StopTimer()
			// retain capacity so the next AddN does not reallocate
			s.Reset()
			b.StartTimer()
		}

		off := s.Len()
		s.AddN(vs[off : off+batch]...)
	}

	intSink = s.Len()
}

// BenchmarkRemove measures one Remove of a present value. When the set drains
// it is refilled to benchSize values with the timer stopped.
func BenchmarkRemove(b *testing.B) {
	vs := filled(benchSize).Slice()
	s := NewWithCap[int](benchSize)

	b.ReportAllocs()

	i := 0
	for b.Loop() {
		if s.IsEmpty() {
			b.StopTimer()
			s.AddN(vs...)
			b.StartTimer()
		}

		v := i % benchSize
		boolSink = s.Remove(v)
		i++
	}

	intSink = s.Len()
}

// BenchmarkRemoveFunc measures one RemoveFunc pass that removes every other
// value of benchSize. The removed values are put back with the timer stopped
// before each pass; the pass itself is long enough to amortize the restore.
func BenchmarkRemoveFunc(b *testing.B) {
	s := filled(benchSize)
	del := func(v int) bool {
		return v%2 == 0
	}

	b.ReportAllocs()

	for b.Loop() {
		if s.Len() < benchSize {
			b.StopTimer()
			for i := range benchSize {
				s.Add(i)
			}

			b.StartTimer()
		}

		intSink = s.RemoveFunc(del)
	}

	intSink = s.Len()
}

// BenchmarkPopFirst measures one PopFirst, which must not allocate. When the
// set drains it is refilled to benchSize values with the timer stopped.
func BenchmarkPopFirst(b *testing.B) {
	vs := filled(benchSize).Slice()
	s := NewWithCap[int](benchSize)

	b.ReportAllocs()

	for b.Loop() {
		if s.IsEmpty() {
			b.StopTimer()
			s.AddN(vs...)
			b.StartTimer()
		}

		intSink, boolSink = s.PopFirst()
	}

	intSink = s.Len()
}

// BenchmarkPopLast measures one PopLast, which must not allocate. When the set
// drains it is refilled to benchSize values with the timer stopped.
func BenchmarkPopLast(b *testing.B) {
	vs := filled(benchSize).Slice()
	s := NewWithCap[int](benchSize)

	b.ReportAllocs()

	for b.Loop() {
		if s.IsEmpty() {
			b.StopTimer()
			s.AddN(vs...)
			b.StartTimer()
		}

		intSink, boolSink = s.PopLast()
	}

	intSink = s.Len()
}

// BenchmarkMoveToBack measures MoveToBack on a set of benchSize values, the
// hit path of an LRU cache.
func BenchmarkMoveToBack(b *testing.B) {
	s := filled(benchSize)

	b.ReportAllocs()

	i := 0
	for b.Loop() {
		v := (i * 7) % benchSize
		boolSink = s.MoveToBack(v)
		i++
	}

	intSink = s.Len()
}

// BenchmarkUnion measures non-mutating union of two sets with 50% overlap
// across several operand sizes, against set; it allocates the result set.
func BenchmarkUnion(b *testing.B) {
	sizes := []int{16, 1024, 65536}

	b.Run("orderedset", func(b *testing.B) {
		for _, size := range sizes {
			b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
				a, c := overlapping(size)

				b.ReportAllocs()

				for b.Loop() {
					setSink = a.Union(c)
				}
			})
		}
	})

	b.Run("set", func(b *testing.B) {
		for _, size := range sizes {
			b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
				a, c := overlappingSet(size)

				b.ReportAllocs()

				for b.Loop() {
					setBaselineSink = a.Union(c)
				}
			})
		}
	})
}

// BenchmarkIntersection measures non-mutating intersection of two sets with
// 50% overlap across several operand sizes, against set; it allocates the
// result set.
func BenchmarkIntersection(b *testing.B) {
	sizes := []int{16, 1024, 65536}

	b.Run("orderedset", func(b *testing.B) {
		for _, size := range sizes {
			b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
				a, c := overlapping(size)

				b.ReportAllocs()

				for b.Loop() {
					setSink = a.Intersection(c)
				}
			})
		}
	})

	b.Run("set", func(b *testing.B) {
		for _, size := range sizes {
			b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
				a, c := overlappingSet(size)

				b.ReportAllocs()

				for b.Loop() {
					setBaselineSink = a.Intersection(c)
				}
			})
		}
	})
}

// BenchmarkDifference measures non-mutating difference of two sets with 50%
// overlap across several operand sizes, against set; it allocates the result
// set.
func BenchmarkDifference(b *testing.B) {
	sizes := []int{16, 1024, 65536}

	b.Run("orderedset", func(b *testing.B) {
		for _, size := range sizes {
			b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
				a, c := overlapping(size)

				b.ReportAllocs()

				for b.Loop() {
					setSink = a.Difference(c)
				}
			})
		}
	})

	b.Run("set", func(b *testing.B) {
		for _, size := range sizes {
			b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
				a, c := overlappingSet(size)

				b.ReportAllocs()

				for b.Loop() {
					setBaselineSink = a.Difference(c)
				}
			})
		}
	})
}

// BenchmarkAddRemoveChurn measures one Remove plus one Add of the same value
// at a steady size of benchSize, which leaves holes behind and exercises
// compaction, against set.
func BenchmarkAddRemoveChurn(b *testing.B) {
	b.Run("orderedset", func(b *testing.B) {
		s := filled(benchSize)

		b.ReportAllocs()

		i := 0
		for b.Loop() {
			v := i % benchSize
			boolSink = s.Remove(v)
			boolSink = s.Add(v)
			i++
		}

		intSink = s.Len()
	})

	b.Run("set", func(b *testing.B) {
		s := filledSet(benchSize)

		b.ReportAllocs()

		i := 0
		for b.Loop() {
			v := i % benchSize
			boolSink = s.Remove(v)
			boolSink = s.Add(v)
			i++
		}

		intSink = s.Len()
	})
}

// BenchmarkPopFirstAddChurn measures one PopFirst plus one Add of a new value
// at a steady size of benchSize: the eviction path of a FIFO cache.
func BenchmarkPopFirstAddChurn(b *testing.B) {
	s := filled(benchSize)

	b.ReportAllocs()

	i := 0
	for b.Loop() {
		v := benchSize + i
		intSink, boolSink = s.PopFirst()
		boolSink = s.Add(v)
		i++
	}

	intSink = s.Len()
}
