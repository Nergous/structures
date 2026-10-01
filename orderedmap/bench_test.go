package orderedmap

import "testing"

// Package-level sinks prevent the compiler from optimizing away benchmarked
// results (and the work that produces them).
var (
	intSink  int
	boolSink bool
)

// benchSize is the steady-state map size for benchmarks that keep the map
// length constant.
const benchSize = 1 << 12

func filled(n int) *Map[int, int] {
	m := NewWithCap[int, int](n)
	for i := range n {
		m.Set(i, i)
	}
	return m
}

func filledBuiltin(n int) map[int]int {
	m := make(map[int]int, n)
	for i := range n {
		m[i] = i
	}
	return m
}

// BenchmarkSet measures inserting new keys, against a built-in map that gets
// the same keys, which shows the cost of keeping order.
func BenchmarkSet(b *testing.B) {
	b.Run("orderedmap", func(b *testing.B) {
		m := NewWithCap[int, int](benchSize)
		b.ReportAllocs()
		k := 0
		for b.Loop() {
			if m.Len() == benchSize {
				b.StopTimer()
				m.Reset()
				b.StartTimer()
			}
			boolSink = m.Set(k, k)
			k++
		}
	})

	b.Run("builtin-map", func(b *testing.B) {
		m := make(map[int]int, benchSize)
		b.ReportAllocs()
		k := 0
		for b.Loop() {
			if len(m) == benchSize {
				b.StopTimer()
				clear(m)
				b.StartTimer()
			}
			m[k] = k
			k++
		}
	})
}

// BenchmarkSetExisting measures replacing the value of a key already present.
func BenchmarkSetExisting(b *testing.B) {
	m := filled(benchSize)
	b.ReportAllocs()

	i := 0
	for b.Loop() {
		boolSink = m.Set(i%benchSize, i)
		i++
	}
}

// BenchmarkGet measures lookups of present keys.
func BenchmarkGet(b *testing.B) {
	b.Run("orderedmap", func(b *testing.B) {
		m := filled(benchSize)
		b.ReportAllocs()
		i := 0
		for b.Loop() {
			v, _ := m.Get(i % benchSize)
			intSink = v
			i++
		}
	})

	b.Run("builtin-map", func(b *testing.B) {
		m := filledBuiltin(benchSize)
		b.ReportAllocs()
		i := 0
		for b.Loop() {
			intSink = m[i%benchSize]
			i++
		}
	})
}

// BenchmarkRemoveSet measures deleting a key and inserting it again, which
// leaves holes behind and exercises compaction.
func BenchmarkRemoveSet(b *testing.B) {
	b.Run("orderedmap", func(b *testing.B) {
		m := filled(benchSize)
		b.ReportAllocs()
		i := 0
		for b.Loop() {
			k := i % benchSize
			v, _ := m.Remove(k)
			m.Set(k, v)
			i++
		}
	})

	b.Run("builtin-map", func(b *testing.B) {
		m := filledBuiltin(benchSize)
		b.ReportAllocs()
		i := 0
		for b.Loop() {
			k := i % benchSize
			v := m[k]
			delete(m, k)
			m[k] = v
			i++
		}
	})
}

// BenchmarkMoveToBack measures marking a key as the newest, the hit path of an
// LRU cache.
func BenchmarkMoveToBack(b *testing.B) {
	m := filled(benchSize)
	b.ReportAllocs()

	i := 0
	for b.Loop() {
		boolSink = m.MoveToBack((i * 7) % benchSize)
		i++
	}
}

// BenchmarkQueue keeps the map at benchSize by popping the oldest key for
// every inserted one: the eviction path of a FIFO cache.
func BenchmarkQueue(b *testing.B) {
	m := filled(benchSize)
	b.ReportAllocs()

	k := benchSize
	for b.Loop() {
		m.PopFirst()
		m.Set(k, k)
		k++
	}
	intSink = m.Len()
}

// BenchmarkAll measures a full ordered traversal, against ranging over a
// built-in map.
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

// BenchmarkRemoveFunc measures removing every other pair in one pass.
func BenchmarkRemoveFunc(b *testing.B) {
	b.ReportAllocs()

	var m *Map[int, int]
	for b.Loop() {
		b.StopTimer()
		m = filled(benchSize)
		b.StartTimer()

		intSink = m.RemoveFunc(func(k, _ int) bool { return k%2 == 0 })
	}
}

// BenchmarkClone measures copying a map with no holes.
func BenchmarkClone(b *testing.B) {
	m := filled(benchSize)
	b.ReportAllocs()

	var c *Map[int, int]
	for b.Loop() {
		c = m.Clone()
	}
	intSink = c.Len()
}
