package bitset

import (
	"fmt"
	"maps"
	"testing"
)

// benchSize is the number of values in containers prepared for benchmarks.
const benchSize = 1 << 12

// Package-level sinks prevent the compiler from optimizing away benchmarked
// results (and the work that produces them).
var (
	intSink        int
	boolSink       bool
	sliceSink      []int
	setSink        *Set
	builtinMapSink map[int]struct{}
)

// filled returns a set holding the values 0..n-1.
func filled(n int) *Set {
	s := NewWithCap(n)
	for i := range n {
		s.Add(i)
	}

	return s
}

// filledBuiltin returns a built-in map holding the keys 0..n-1.
func filledBuiltin(n int) map[int]struct{} {
	m := make(map[int]struct{}, n)
	for i := range n {
		m[i] = struct{}{}
	}

	return m
}

// overlapping returns two sets of n values each that share half of their
// values: 0..n-1 and n/2..n/2+n-1.
func overlapping(n int) (*Set, *Set) {
	a := filled(n)
	c := NewWithCap(n + n/2)
	for i := range n {
		c.Add(i + n/2)
	}

	return a, c
}

// BenchmarkFrom measures building a set from benchSize values in which every
// value appears twice. From is Of over the same slice, so this covers both
// constructors.
func BenchmarkFrom(b *testing.B) {
	vs := make([]int, benchSize)
	for i := range vs {
		vs[i] = i / 2
	}

	b.ReportAllocs()

	for b.Loop() {
		setSink = From(vs)
	}
}

// BenchmarkContains measures lookups of present ("hit") and absent ("miss")
// values, against a built-in map.
func BenchmarkContains(b *testing.B) {
	b.Run("bitset", func(b *testing.B) {
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

	b.Run("builtin-map", func(b *testing.B) {
		m := filledBuiltin(benchSize)

		b.Run("hit", func(b *testing.B) {
			b.ReportAllocs()

			i := 0
			for b.Loop() {
				v := i % benchSize
				_, boolSink = m[v]
				i++
			}
		})

		b.Run("miss", func(b *testing.B) {
			b.ReportAllocs()

			i := 0
			for b.Loop() {
				v := benchSize + i%benchSize
				_, boolSink = m[v]
				i++
			}
		})
	})
}

// BenchmarkMin measures Min on a set whose smallest value sits in its last
// word, the worst case of the forward word scan.
func BenchmarkMin(b *testing.B) {
	s := filled(benchSize)
	s.RemoveFunc(func(v int) bool { return v < benchSize-1 })

	b.ReportAllocs()

	for b.Loop() {
		intSink, boolSink = s.Min()
	}
}

// BenchmarkAll measures a full iteration over benchSize values via the
// range-over-func iterator, confirming iteration is allocation-free, against
// a built-in map.
func BenchmarkAll(b *testing.B) {
	b.Run("bitset", func(b *testing.B) {
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

	b.Run("builtin-map", func(b *testing.B) {
		m := filledBuiltin(benchSize)

		b.ReportAllocs()

		for b.Loop() {
			sum := 0
			for v := range m {
				sum += v
			}

			intSink = sum
		}
	})
}

// BenchmarkClone measures cloning a set of benchSize values, which allocates
// the new set and its backing array per call, against a built-in map.
func BenchmarkClone(b *testing.B) {
	b.Run("bitset", func(b *testing.B) {
		s := filled(benchSize)

		b.ReportAllocs()

		for b.Loop() {
			setSink = s.Clone()
		}
	})

	b.Run("builtin-map", func(b *testing.B) {
		m := filledBuiltin(benchSize)

		b.ReportAllocs()

		for b.Loop() {
			builtinMapSink = maps.Clone(m)
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
// the amortized cost of reallocations is included, against a built-in map.
// When the set reaches benchSize it is cleared with the timer stopped.
func BenchmarkAdd(b *testing.B) {
	b.Run("bitset", func(b *testing.B) {
		s := New()

		b.ReportAllocs()

		i := 0
		for b.Loop() {
			if i == benchSize {
				b.StopTimer()
				s.Clear()
				i = 0
				b.StartTimer()
			}

			boolSink = s.Add(i)
			i++
		}

		intSink = s.Len()
	})

	b.Run("builtin-map", func(b *testing.B) {
		m := make(map[int]struct{})

		b.ReportAllocs()

		i := 0
		for b.Loop() {
			if i == benchSize {
				b.StopTimer()
				m = make(map[int]struct{})
				i = 0
				b.StartTimer()
			}

			m[i] = struct{}{}
			i++
		}

		intSink = len(m)
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
	s := NewWithCap(benchSize)

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

// BenchmarkAddSet measures merging a set of benchSize values with 50% overlap
// into a set that already covers it, so no reallocation happens.
func BenchmarkAddSet(b *testing.B) {
	a, c := overlapping(benchSize)
	a.Grow(c.Cap())

	b.ReportAllocs()

	for b.Loop() {
		a.AddSet(c)
	}

	intSink = a.Len()
}

// BenchmarkRemove measures one Remove of a present value. When the set drains
// it is refilled to benchSize values with the timer stopped.
func BenchmarkRemove(b *testing.B) {
	vs := filled(benchSize).Slice()
	s := NewWithCap(benchSize)

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
	full := s.Clone()
	del := func(v int) bool {
		return v%2 == 0
	}

	b.ReportAllocs()

	for b.Loop() {
		if s.Len() < benchSize {
			b.StopTimer()
			s.AddSet(full)
			b.StartTimer()
		}

		intSink = s.RemoveFunc(del)
	}

	intSink = s.Len()
}

// BenchmarkUnion measures non-mutating union of two sets with 50% overlap
// across several operand sizes; it allocates the result set.
func BenchmarkUnion(b *testing.B) {
	sizes := []int{16, 1024, 65536}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			a, c := overlapping(size)

			b.ReportAllocs()

			for b.Loop() {
				setSink = a.Union(c)
			}
		})
	}
}

// BenchmarkIntersection measures non-mutating intersection of two sets with
// 50% overlap across several operand sizes; it allocates the result set.
func BenchmarkIntersection(b *testing.B) {
	sizes := []int{16, 1024, 65536}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			a, c := overlapping(size)

			b.ReportAllocs()

			for b.Loop() {
				setSink = a.Intersection(c)
			}
		})
	}
}

// BenchmarkDifference measures non-mutating difference of two sets with 50%
// overlap across several operand sizes; it allocates the result set.
func BenchmarkDifference(b *testing.B) {
	sizes := []int{16, 1024, 65536}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			a, c := overlapping(size)

			b.ReportAllocs()

			for b.Loop() {
				setSink = a.Difference(c)
			}
		})
	}
}

// BenchmarkIsSubset measures a subset check that has to compare every word,
// since a is a subset of c.
func BenchmarkIsSubset(b *testing.B) {
	a := filled(benchSize)
	c := filled(2 * benchSize)

	b.ReportAllocs()

	for b.Loop() {
		boolSink = a.IsSubset(c)
	}
}

// BenchmarkAddRemoveChurn measures one Remove plus one Add of the same value
// at a steady size of benchSize, against a built-in map.
func BenchmarkAddRemoveChurn(b *testing.B) {
	b.Run("bitset", func(b *testing.B) {
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

	b.Run("builtin-map", func(b *testing.B) {
		m := filledBuiltin(benchSize)

		b.ReportAllocs()

		i := 0
		for b.Loop() {
			v := i % benchSize
			delete(m, v)
			m[v] = struct{}{}
			i++
		}

		intSink = len(m)
	})
}
