package orderedmap

import (
	"maps"
	"testing"
)

// benchSize is the number of keys in containers prepared for benchmarks.
const benchSize = 1 << 12

// Package-level sinks prevent the compiler from optimizing away benchmarked
// results (and the work that produces them).
var (
	intSink        int
	keySink        int
	boolSink       bool
	mapSink        *Map[int, int]
	builtinMapSink map[int]int
)

// filled returns a map holding the keys 0..n-1 in insertion order, each
// mapped to itself.
func filled(n int) *Map[int, int] {
	m := NewWithCap[int, int](n)
	for i := range n {
		m.Set(i, i)
	}

	return m
}

// filledBuiltin returns a built-in map holding the keys 0..n-1, each mapped to
// itself.
func filledBuiltin(n int) map[int]int {
	m := make(map[int]int, n)
	for i := range n {
		m[i] = i
	}

	return m
}

// BenchmarkGet measures Get of a present key, against a built-in map.
func BenchmarkGet(b *testing.B) {
	b.Run("orderedmap", func(b *testing.B) {
		m := filled(benchSize)

		b.ReportAllocs()

		i := 0
		for b.Loop() {
			k := i % benchSize
			intSink, boolSink = m.Get(k)
			i++
		}
	})

	b.Run("builtin-map", func(b *testing.B) {
		m := filledBuiltin(benchSize)

		b.ReportAllocs()

		i := 0
		for b.Loop() {
			k := i % benchSize
			intSink, boolSink = m[k]
			i++
		}
	})
}

// BenchmarkContains measures lookups of present ("hit") and absent ("miss")
// keys, against a built-in map.
func BenchmarkContains(b *testing.B) {
	b.Run("orderedmap", func(b *testing.B) {
		m := filled(benchSize)

		b.Run("hit", func(b *testing.B) {
			b.ReportAllocs()

			i := 0
			for b.Loop() {
				k := i % benchSize
				boolSink = m.Contains(k)
				i++
			}
		})

		b.Run("miss", func(b *testing.B) {
			b.ReportAllocs()

			i := 0
			for b.Loop() {
				k := benchSize + i%benchSize
				boolSink = m.Contains(k)
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
				k := i % benchSize
				_, boolSink = m[k]
				i++
			}
		})

		b.Run("miss", func(b *testing.B) {
			b.ReportAllocs()

			i := 0
			for b.Loop() {
				k := benchSize + i%benchSize
				_, boolSink = m[k]
				i++
			}
		})
	})
}

// BenchmarkFirst measures reading the oldest pair, which must stay O(1) and
// allocation-free.
func BenchmarkFirst(b *testing.B) {
	m := filled(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		keySink, intSink, boolSink = m.First()
	}
}

// BenchmarkLast measures reading the newest pair, which must stay O(1) and
// allocation-free.
func BenchmarkLast(b *testing.B) {
	m := filled(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		keySink, intSink, boolSink = m.Last()
	}
}

// BenchmarkAll measures a full iteration over benchSize pairs via the
// range-over-func iterator, confirming iteration is allocation-free, against
// a built-in map.
func BenchmarkAll(b *testing.B) {
	b.Run("orderedmap", func(b *testing.B) {
		m := filled(benchSize)

		b.ReportAllocs()

		for b.Loop() {
			sum := 0
			for _, v := range m.All() {
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
			for _, v := range m {
				sum += v
			}

			intSink = sum
		}
	})
}

// BenchmarkBackward measures a full reverse iteration over benchSize pairs via
// the range-over-func iterator, confirming iteration is allocation-free.
func BenchmarkBackward(b *testing.B) {
	m := filled(benchSize)

	b.ReportAllocs()

	for b.Loop() {
		sum := 0
		for _, v := range m.Backward() {
			sum += v
		}

		intSink = sum
	}
}

// BenchmarkClone measures cloning a map of benchSize pairs, which allocates
// the new map, its backing array, and its index map per call, against a
// built-in map.
func BenchmarkClone(b *testing.B) {
	b.Run("orderedmap", func(b *testing.B) {
		m := filled(benchSize)

		b.ReportAllocs()

		for b.Loop() {
			mapSink = m.Clone()
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

// BenchmarkSet measures Set into a map that grows from empty to benchSize, so
// the amortized cost of reallocations is included, against a built-in map.
// When the map reaches benchSize it is cleared with the timer stopped.
func BenchmarkSet(b *testing.B) {
	b.Run("orderedmap", func(b *testing.B) {
		m := New[int, int]()

		b.ReportAllocs()

		i := 0
		for b.Loop() {
			if m.Len() == benchSize {
				b.StopTimer()
				m.Clear()
				b.StartTimer()
			}

			boolSink = m.Set(i, i)
			i++
		}

		intSink = m.Len()
	})

	b.Run("builtin-map", func(b *testing.B) {
		m := make(map[int]int)

		b.ReportAllocs()

		i := 0
		for b.Loop() {
			if len(m) == benchSize {
				b.StopTimer()
				m = make(map[int]int)
				b.StartTimer()
			}

			m[i] = i
			i++
		}

		intSink = len(m)
	})
}

// BenchmarkSetExisting measures Set of a key that is already present, which
// replaces its value and keeps its place.
func BenchmarkSetExisting(b *testing.B) {
	m := filled(benchSize)

	b.ReportAllocs()

	i := 0
	for b.Loop() {
		k := i % benchSize
		boolSink = m.Set(k, i)
		i++
	}

	intSink = m.Len()
}

// BenchmarkRemove measures one Remove of a present key. When the map drains it
// is refilled to benchSize keys with the timer stopped.
func BenchmarkRemove(b *testing.B) {
	m := NewWithCap[int, int](benchSize)

	b.ReportAllocs()

	i := 0
	for b.Loop() {
		if m.IsEmpty() {
			b.StopTimer()
			for k := range benchSize {
				m.Set(k, k)
			}

			b.StartTimer()
		}

		k := i % benchSize
		intSink, boolSink = m.Remove(k)
		i++
	}

	intSink = m.Len()
}

// BenchmarkRemoveFunc measures one RemoveFunc pass that removes every other
// pair of benchSize. The removed pairs are put back with the timer stopped
// before each pass; the pass itself is long enough to amortize the restore.
func BenchmarkRemoveFunc(b *testing.B) {
	m := filled(benchSize)
	del := func(k, _ int) bool {
		return k%2 == 0
	}

	b.ReportAllocs()

	for b.Loop() {
		if m.Len() < benchSize {
			b.StopTimer()
			for i := range benchSize {
				m.Set(i, i)
			}

			b.StartTimer()
		}

		intSink = m.RemoveFunc(del)
	}

	intSink = m.Len()
}

// BenchmarkPopFirst measures one PopFirst, which must not allocate. When the
// map drains it is refilled to benchSize keys with the timer stopped.
func BenchmarkPopFirst(b *testing.B) {
	m := NewWithCap[int, int](benchSize)

	b.ReportAllocs()

	for b.Loop() {
		if m.IsEmpty() {
			b.StopTimer()
			for k := range benchSize {
				m.Set(k, k)
			}

			b.StartTimer()
		}

		keySink, intSink, boolSink = m.PopFirst()
	}

	intSink = m.Len()
}

// BenchmarkPopLast measures one PopLast, which must not allocate. When the map
// drains it is refilled to benchSize keys with the timer stopped.
func BenchmarkPopLast(b *testing.B) {
	m := NewWithCap[int, int](benchSize)

	b.ReportAllocs()

	for b.Loop() {
		if m.IsEmpty() {
			b.StopTimer()
			for k := range benchSize {
				m.Set(k, k)
			}

			b.StartTimer()
		}

		keySink, intSink, boolSink = m.PopLast()
	}

	intSink = m.Len()
}

// BenchmarkMoveToBack measures MoveToBack on a map of benchSize keys, the hit
// path of an LRU cache.
func BenchmarkMoveToBack(b *testing.B) {
	m := filled(benchSize)

	b.ReportAllocs()

	i := 0
	for b.Loop() {
		k := (i * 7) % benchSize
		boolSink = m.MoveToBack(k)
		i++
	}

	intSink = m.Len()
}

// BenchmarkSetRemoveChurn measures one Remove plus one Set of the same key at
// a steady size of benchSize, which leaves holes behind and exercises
// compaction, against a built-in map.
func BenchmarkSetRemoveChurn(b *testing.B) {
	b.Run("orderedmap", func(b *testing.B) {
		m := filled(benchSize)

		b.ReportAllocs()

		i := 0
		for b.Loop() {
			k := i % benchSize
			intSink, boolSink = m.Remove(k)
			boolSink = m.Set(k, k)
			i++
		}

		intSink = m.Len()
	})

	b.Run("builtin-map", func(b *testing.B) {
		m := filledBuiltin(benchSize)

		b.ReportAllocs()

		i := 0
		for b.Loop() {
			k := i % benchSize
			intSink = m[k]
			delete(m, k)
			m[k] = k
			i++
		}

		intSink = len(m)
	})
}

// BenchmarkPopFirstSetChurn measures one PopFirst plus one Set of a new key at
// a steady size of benchSize: the eviction path of a FIFO cache.
func BenchmarkPopFirstSetChurn(b *testing.B) {
	m := filled(benchSize)

	b.ReportAllocs()

	i := 0
	for b.Loop() {
		k := benchSize + i
		keySink, intSink, boolSink = m.PopFirst()
		boolSink = m.Set(k, k)
		i++
	}

	intSink = m.Len()
}
