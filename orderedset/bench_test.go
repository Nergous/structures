package orderedset

import (
	"testing"

	"github.com/Nergous/structures/set"
)

// Package-level sinks prevent the compiler from optimizing away benchmarked
// results (and the work that produces them).
var (
	intSink     int
	boolSink    bool
	sliceSink   []int
	setSink     *Set[int]
	hashSetSink *set.Set[int]
)

// benchSize is the steady-state set size for benchmarks that keep the set
// length constant.
const benchSize = 1 << 12

// benchIterSize is the set size for traversal, set algebra, and construction
// benchmarks.
const benchIterSize = 1 << 10

// benchFill returns an ordered set holding lo, lo+1, ..., lo+n-1.
func benchFill(lo, n int) *Set[int] {
	s := NewWithCap[int](n)
	for i := range n {
		s.Add(lo + i)
	}

	return s
}

// benchFillSet returns an unordered set holding lo, lo+1, ..., lo+n-1.
func benchFillSet(lo, n int) *set.Set[int] {
	s := set.NewWithCap[int](n)
	for i := range n {
		s.Add(lo + i)
	}

	return s
}

// BenchmarkAdd measures inserting new values, against an unordered set that
// gets the same values, which shows the cost of keeping order.
func BenchmarkAdd(b *testing.B) {
	b.Run("orderedset", func(b *testing.B) {
		s := NewWithCap[int](benchSize)

		b.ReportAllocs()

		v := 0
		for b.Loop() {
			if s.Len() == benchSize {
				b.StopTimer()
				s.Reset()
				b.StartTimer()
			}

			boolSink = s.Add(v)
			v++
		}
	})

	b.Run("set", func(b *testing.B) {
		s := set.NewWithCap[int](benchSize)

		b.ReportAllocs()

		v := 0
		for b.Loop() {
			if s.Len() == benchSize {
				b.StopTimer()
				s.Reset()
				b.StartTimer()
			}

			boolSink = s.Add(v)
			v++
		}
	})
}

// BenchmarkAddExisting measures adding a value already present, which must
// leave both membership and order unchanged.
func BenchmarkAddExisting(b *testing.B) {
	s := benchFill(0, benchSize)

	b.ReportAllocs()

	i := 0
	for b.Loop() {
		boolSink = s.Add(i % benchSize)
		i++
	}
}

// BenchmarkContains measures lookups of present values, against an unordered
// set.
func BenchmarkContains(b *testing.B) {
	b.Run("orderedset", func(b *testing.B) {
		s := benchFill(0, benchSize)

		b.ReportAllocs()

		i := 0
		for b.Loop() {
			boolSink = s.Contains(i % benchSize)
			i++
		}
	})

	b.Run("set", func(b *testing.B) {
		s := benchFillSet(0, benchSize)

		b.ReportAllocs()

		i := 0
		for b.Loop() {
			boolSink = s.Contains(i % benchSize)
			i++
		}
	})
}

// BenchmarkRemove measures Remove in isolation, oldest value first. Refilling
// happens outside the timed section: whenever the set drains we add a fresh
// chunk with the timer stopped, so only removals are measured.
func BenchmarkRemove(b *testing.B) {
	b.ReportAllocs()

	s := NewWithCap[int](benchSize)
	next := 0
	for b.Loop() {
		if s.IsEmpty() {
			b.StopTimer()
			for j := range benchSize {
				s.Add(next + j)
			}

			next += benchSize
			b.StartTimer()
		}

		boolSink = s.Remove(next - s.Len())
	}
}

// BenchmarkAddRemoveChurn measures removing a value and adding it back at a
// constant set size, which leaves holes behind and exercises compaction,
// against an unordered set doing the same.
func BenchmarkAddRemoveChurn(b *testing.B) {
	b.Run("orderedset", func(b *testing.B) {
		s := benchFill(0, benchSize)

		b.ReportAllocs()

		i := 0
		for b.Loop() {
			v := i % benchSize
			s.Remove(v)
			boolSink = s.Add(v)
			i++
		}
	})

	b.Run("set", func(b *testing.B) {
		s := benchFillSet(0, benchSize)

		b.ReportAllocs()

		i := 0
		for b.Loop() {
			v := i % benchSize
			s.Remove(v)
			boolSink = s.Add(v)
			i++
		}
	})
}

// BenchmarkPopFirst measures removing the oldest value. The set is refilled
// with the timer stopped whenever it drains.
func BenchmarkPopFirst(b *testing.B) {
	b.ReportAllocs()

	s := NewWithCap[int](benchSize)
	for b.Loop() {
		if s.IsEmpty() {
			b.StopTimer()
			for j := range benchSize {
				s.Add(j)
			}

			b.StartTimer()
		}

		v, ok := s.PopFirst()
		intSink = v
		boolSink = ok
	}
}

// BenchmarkPopLast measures removing the newest value. The set is refilled
// with the timer stopped whenever it drains.
func BenchmarkPopLast(b *testing.B) {
	b.ReportAllocs()

	s := NewWithCap[int](benchSize)
	for b.Loop() {
		if s.IsEmpty() {
			b.StopTimer()
			for j := range benchSize {
				s.Add(j)
			}

			b.StartTimer()
		}

		v, ok := s.PopLast()
		intSink = v
		boolSink = ok
	}
}

// BenchmarkMoveToBack measures marking a value as the newest, the hit path of
// an LRU cache.
func BenchmarkMoveToBack(b *testing.B) {
	s := benchFill(0, benchSize)

	b.ReportAllocs()

	i := 0
	for b.Loop() {
		boolSink = s.MoveToBack((i * 7) % benchSize)
		i++
	}
}

// BenchmarkAll measures a full ordered traversal, against ranging over an
// unordered set.
func BenchmarkAll(b *testing.B) {
	b.Run("orderedset", func(b *testing.B) {
		s := benchFill(0, benchIterSize)

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
		s := benchFillSet(0, benchIterSize)

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

// BenchmarkBackward measures a full traversal from newest to oldest.
func BenchmarkBackward(b *testing.B) {
	s := benchFill(0, benchIterSize)

	b.ReportAllocs()

	for b.Loop() {
		sum := 0
		for v := range s.Backward() {
			sum += v
		}

		intSink = sum
	}
}

// BenchmarkSlice measures copying the set out to a plain slice in insertion
// order, which allocates exactly one backing array per call.
func BenchmarkSlice(b *testing.B) {
	s := benchFill(0, benchIterSize)

	b.ReportAllocs()

	for b.Loop() {
		sliceSink = s.Slice()
	}
}

// BenchmarkClone measures copying a set with no holes.
func BenchmarkClone(b *testing.B) {
	s := benchFill(0, benchSize)

	b.ReportAllocs()

	for b.Loop() {
		setSink = s.Clone()
	}
}

// BenchmarkRemoveFunc measures removing every other value in one pass.
func BenchmarkRemoveFunc(b *testing.B) {
	b.ReportAllocs()

	var s *Set[int]
	for b.Loop() {
		b.StopTimer()
		s = benchFill(0, benchSize)
		b.StartTimer()

		intSink = s.RemoveFunc(func(v int) bool { return v%2 == 0 })
	}
}

// BenchmarkUnion measures non-mutating union of two sets with 50% overlap,
// against an unordered set.
func BenchmarkUnion(b *testing.B) {
	b.Run("orderedset", func(b *testing.B) {
		a := benchFill(0, benchIterSize)
		c := benchFill(benchIterSize/2, benchIterSize)

		b.ReportAllocs()

		for b.Loop() {
			setSink = a.Union(c)
		}
	})

	b.Run("set", func(b *testing.B) {
		a := benchFillSet(0, benchIterSize)
		c := benchFillSet(benchIterSize/2, benchIterSize)

		b.ReportAllocs()

		for b.Loop() {
			hashSetSink = a.Union(c)
		}
	})
}

// BenchmarkIntersection measures non-mutating intersection of two sets with
// 50% overlap, against an unordered set.
func BenchmarkIntersection(b *testing.B) {
	b.Run("orderedset", func(b *testing.B) {
		a := benchFill(0, benchIterSize)
		c := benchFill(benchIterSize/2, benchIterSize)

		b.ReportAllocs()

		for b.Loop() {
			setSink = a.Intersection(c)
		}
	})

	b.Run("set", func(b *testing.B) {
		a := benchFillSet(0, benchIterSize)
		c := benchFillSet(benchIterSize/2, benchIterSize)

		b.ReportAllocs()

		for b.Loop() {
			hashSetSink = a.Intersection(c)
		}
	})
}

// BenchmarkDifference measures non-mutating difference of two sets with 50%
// overlap, against an unordered set.
func BenchmarkDifference(b *testing.B) {
	b.Run("orderedset", func(b *testing.B) {
		a := benchFill(0, benchIterSize)
		c := benchFill(benchIterSize/2, benchIterSize)

		b.ReportAllocs()

		for b.Loop() {
			setSink = a.Difference(c)
		}
	})

	b.Run("set", func(b *testing.B) {
		a := benchFillSet(0, benchIterSize)
		c := benchFillSet(benchIterSize/2, benchIterSize)

		b.ReportAllocs()

		for b.Loop() {
			hashSetSink = a.Difference(c)
		}
	})
}

// BenchmarkFrom measures building a set from a slice in which every value
// appears twice, against an unordered set. From is Of over the same slice,
// so this covers both constructors.
func BenchmarkFrom(b *testing.B) {
	vs := make([]int, benchIterSize)
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
			hashSetSink = set.From(vs)
		}
	})
}
