package binheap

import (
	"cmp"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
)

// assertHeap fails the test when h.data violates the heap property under the
// heap's own comparator.
func assertHeap[T any](t *testing.T, h *Heap[T]) {
	t.Helper()

	for i := 1; i < len(h.data); i++ {
		parent := (i - 1) / 2
		if h.compare(h.data[i], h.data[parent]) < 0 {
			t.Fatalf("heap property violated: data[%d]=%v < parent data[%d]=%v in %v",
				i, h.data[i], parent, h.data[parent], h.data)
		}
	}
}

// assertTailZeroed fails the test when any slot between len and cap still
// holds a non-zero pointer, i.e. the heap retains a reference it removed.
func assertTailZeroed(t *testing.T, h *Heap[*int]) {
	t.Helper()

	for i, p := range h.data[len(h.data):cap(h.data)] {
		if p != nil {
			t.Fatalf("slot %d beyond len still holds %p, want nil", len(h.data)+i, p)
		}
	}
}

func assertPanics(t *testing.T, name, want string, fn func()) {
	t.Helper()

	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Fatalf("%s did not panic, want panic", name)
		}
		if msg, _ := r.(string); want != "" && !strings.Contains(msg, want) {
			t.Fatalf("%s panicked with %v, want message containing %q", name, r, want)
		}
	}()
	fn()
}

func drain[T any](h *Heap[T]) []T {
	var out []T
	for {
		v, ok := h.Pop()
		if !ok {
			return out
		}
		out = append(out, v)
	}
}

func byPtr(a, b *int) int { return cmp.Compare(*a, *b) }

func ptrs(vs ...int) []*int {
	out := make([]*int, len(vs))
	for i := range vs {
		out[i] = &vs[i]
	}
	return out
}

func assertNewEmpty[T cmp.Ordered](t *testing.T) {
	t.Helper()

	h := New[T]()
	if got := h.Len(); got != 0 {
		t.Errorf("Len() = %d, want 0", got)
	}
	if got := h.Cap(); got != 0 {
		t.Errorf("Cap() = %d, want 0", got)
	}
}

func assertNewMaxEmpty[T cmp.Ordered](t *testing.T) {
	t.Helper()

	h := NewMax[T]()
	if got := h.Len(); got != 0 {
		t.Errorf("Len() = %d, want 0", got)
	}
	if got := h.Cap(); got != 0 {
		t.Errorf("Cap() = %d, want 0", got)
	}
}

func TestHeap_New(t *testing.T) {
	t.Run("int", assertNewEmpty[int])
	t.Run("string", assertNewEmpty[string])
	t.Run("float32", assertNewEmpty[float32])
}

func TestHeap_NewMax(t *testing.T) {
	t.Run("int", assertNewMaxEmpty[int])
	t.Run("string", assertNewMaxEmpty[string])
	t.Run("float32", assertNewMaxEmpty[float32])
}

func TestHeap_WithCap(t *testing.T) {
	constructors := []struct {
		name string
		make func(n int) *Heap[int]
	}{
		{name: "NewWithCap", make: NewWithCap[int]},
		{name: "NewMaxWithCap", make: NewMaxWithCap[int]},
		{name: "NewFuncWithCap", make: func(n int) *Heap[int] { return NewFuncWithCap(n, cmp.Compare[int]) }},
	}
	tests := []struct {
		name string
		n    int
		want int
	}{
		{name: "negative clamps to zero", n: -5, want: 0},
		{name: "zero", n: 0, want: 0},
		{name: "one", n: 1, want: 1},
		{name: "ten", n: 10, want: 10},
	}

	for _, c := range constructors {
		for _, tt := range tests {
			t.Run(c.name+"/"+tt.name, func(t *testing.T) {
				h := c.make(tt.n)
				if got := h.Len(); got != 0 {
					t.Errorf("Len() = %d, want 0", got)
				}
				if got := h.Cap(); got != tt.want {
					t.Errorf("Cap() = %d, want %d", got, tt.want)
				}
			})
		}
	}
}

func TestHeap_From(t *testing.T) {
	tests := []struct {
		name string
		s    []int
	}{
		{name: "nil", s: nil},
		{name: "empty", s: []int{}},
		{name: "one", s: []int{1}},
		{name: "sorted", s: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
		{name: "reversed", s: []int{10, 9, 8, 7, 6, 5, 4, 3, 2, 1}},
		{name: "shuffled", s: []int{1, 6, 2, 7, 3, 8, 4, 9, 5, 10}},
		{name: "duplicates", s: []int{3, 1, 3, 1, 2, 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := From(tt.s)
			assertHeap(t, h)
			if got := h.Len(); got != len(tt.s) {
				t.Errorf("Len() = %d, want %d", got, len(tt.s))
			}
			if got := h.Cap(); got != len(tt.s) {
				t.Errorf("Cap() = %d, want %d", got, len(tt.s))
			}

			want := slices.Sorted(slices.Values(tt.s))
			if got := drain(h); !slices.Equal(got, want) {
				t.Errorf("pop order = %v, want %v", got, want)
			}
		})
	}
}

func TestHeap_FromCopiesInput(t *testing.T) {
	s := []int{5, 3, 8, 1}
	h := From(s)
	s[0], s[3] = -100, -200

	if got, _ := h.Peek(); got != 1 {
		t.Errorf("Peek() after changing input = %d, want 1 (heap must not alias input)", got)
	}
	h.Push(0)
	if !slices.Equal(s, []int{-100, 3, 8, -200}) {
		t.Errorf("input = %v, want it unchanged by heap mutation", s)
	}
}

func TestHeap_FromMax(t *testing.T) {
	s := []int{1, 6, 2, 7, 3, 8, 4, 9, 5, 10}
	h := FromMax(s)
	assertHeap(t, h)

	want := []int{10, 9, 8, 7, 6, 5, 4, 3, 2, 1}
	if got := drain(h); !slices.Equal(got, want) {
		t.Errorf("pop order = %v, want %v", got, want)
	}
}

func TestHeap_FromFunc(t *testing.T) {
	byLen := func(a, b string) int { return cmp.Compare(len(a), len(b)) }
	h := FromFunc([]string{"ccc", "a", "dddd", "bb"}, byLen)
	assertHeap(t, h)

	want := []string{"a", "bb", "ccc", "dddd"}
	if got := drain(h); !slices.Equal(got, want) {
		t.Errorf("pop order = %v, want %v", got, want)
	}
}

func TestHeap_NilComparatorPanics(t *testing.T) {
	assertPanics(t, "NewFunc(nil)", "nil comparator", func() { NewFunc[int](nil) })
	assertPanics(t, "NewFuncWithCap(nil)", "nil comparator", func() { NewFuncWithCap[int](4, nil) })
	assertPanics(t, "FromFunc(nil)", "nil comparator", func() { FromFunc([]int{1}, nil) })
}

func TestHeap_PushPop(t *testing.T) {
	tests := []struct {
		name string
		make func() *Heap[int]
		push []int
		want []int
	}{
		{name: "min empty", make: New[int], push: nil, want: nil},
		{name: "min single", make: New[int], push: []int{42}, want: []int{42}},
		{name: "min order", make: New[int], push: []int{5, 1, 4, 2, 3}, want: []int{1, 2, 3, 4, 5}},
		{name: "min duplicates", make: New[int], push: []int{2, 1, 2, 1}, want: []int{1, 1, 2, 2}},
		{name: "max order", make: NewMax[int], push: []int{5, 1, 4, 2, 3}, want: []int{5, 4, 3, 2, 1}},
		{
			name: "func order",
			make: func() *Heap[int] { return NewFunc(func(a, b int) int { return cmp.Compare(a%10, b%10) }) },
			push: []int{19, 21, 35, 3},
			want: []int{21, 3, 35, 19},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := tt.make()
			for _, v := range tt.push {
				h.Push(v)
				assertHeap(t, h)
			}

			for i, want := range tt.want {
				got, ok := h.Pop()
				if !ok {
					t.Fatalf("Pop() #%d: ok = false, want true", i)
				}
				if got != want {
					t.Errorf("Pop() #%d = %d, want %d", i, got, want)
				}
				assertHeap(t, h)
			}

			if got, ok := h.Pop(); ok {
				t.Errorf("Pop() on empty = %d, true; want ok = false", got)
			}
		})
	}
}

func TestHeap_PopZeroesSlot(t *testing.T) {
	h := FromFunc(ptrs(3, 1, 2), byPtr)
	h.Pop()
	assertTailZeroed(t, h)
	h.Pop()
	h.Pop()
	assertTailZeroed(t, h)
}

func TestHeap_Peek(t *testing.T) {
	t.Run("empty returns zero value and false", func(t *testing.T) {
		got, ok := New[int]().Peek()
		if ok || got != 0 {
			t.Errorf("Peek() = %d, %v; want 0, false", got, ok)
		}
	})

	t.Run("returns least without removing it", func(t *testing.T) {
		h := From([]int{4, 2, 9})
		got, ok := h.Peek()
		if !ok || got != 2 {
			t.Errorf("Peek() = %d, %v; want 2, true", got, ok)
		}
		if h.Len() != 3 {
			t.Errorf("Len() after Peek() = %d, want 3", h.Len())
		}
	})
}

func TestHeap_IsEmpty(t *testing.T) {
	h := New[int]()
	if !h.IsEmpty() {
		t.Error("new heap: IsEmpty() = false, want true")
	}
	h.Push(1)
	if h.IsEmpty() {
		t.Error("after Push: IsEmpty() = true, want false")
	}
	h.Pop()
	if !h.IsEmpty() {
		t.Error("after Pop: IsEmpty() = false, want true")
	}
}

func TestHeap_PushN(t *testing.T) {
	tests := []struct {
		name    string
		initial []int
		push    []int
	}{
		{name: "no arguments", initial: []int{3, 1, 2}, push: nil},
		{name: "into empty", initial: nil, push: []int{5, 3, 9, 1, 7, 2, 8}},
		{name: "small batch", initial: []int{10, 20, 30, 40, 50, 60, 70, 80}, push: []int{5, 45}},
		{name: "large batch", initial: []int{10, 20}, push: []int{9, 8, 7, 6, 5, 4, 3, 2, 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := From(tt.initial)
			h.PushN(tt.push...)
			assertHeap(t, h)

			want := slices.Sorted(slices.Values(slices.Concat(tt.initial, tt.push)))
			if got := drain(h); !slices.Equal(got, want) {
				t.Errorf("pop order = %v, want %v", got, want)
			}
		})
	}
}

func TestHeap_PushNBothStrategies(t *testing.T) {
	// A 6-element batch into a large heap must sift up; the same batch into a
	// tiny heap must rebuild. Both have to keep the heap valid.
	if rebuildCheaper(1<<16, 6) {
		t.Fatal("rebuildCheaper(65536, 6) = true, want sift-up for a small batch")
	}
	if !rebuildCheaper(4, 6) {
		t.Fatal("rebuildCheaper(4, 6) = false, want rebuild for a batch larger than the heap")
	}

	r := rand.New(rand.NewPCG(1, 2))
	for _, n := range []int{0, 4, 1 << 12} {
		s := make([]int, n)
		for i := range s {
			s[i] = r.IntN(1000)
		}
		h := From(s)
		batch := []int{r.IntN(1000), r.IntN(1000), -1, r.IntN(1000), r.IntN(1000), r.IntN(1000)}
		h.PushN(batch...)
		assertHeap(t, h)
		if got, _ := h.Peek(); got != -1 {
			t.Errorf("n=%d: Peek() = %d, want -1", n, got)
		}
	}
}

func TestHeap_PopN(t *testing.T) {
	tests := []struct {
		name     string
		initial  []int
		n        int
		want     []int
		wantLeft int
	}{
		{name: "zero", initial: []int{3, 1, 2}, n: 0, want: nil, wantLeft: 3},
		{name: "negative", initial: []int{3, 1, 2}, n: -1, want: nil, wantLeft: 3},
		{name: "empty heap", initial: nil, n: 2, want: nil, wantLeft: 0},
		{name: "partial", initial: []int{5, 3, 4, 1, 2}, n: 2, want: []int{1, 2}, wantLeft: 3},
		{name: "exact", initial: []int{3, 1, 2}, n: 3, want: []int{1, 2, 3}, wantLeft: 0},
		{name: "more than len", initial: []int{3, 1, 2}, n: 10, want: []int{1, 2, 3}, wantLeft: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := From(tt.initial)
			got := h.PopN(tt.n)
			if !slices.Equal(got, tt.want) || (tt.want == nil) != (got == nil) {
				t.Errorf("PopN(%d) = %#v, want %#v", tt.n, got, tt.want)
			}
			if h.Len() != tt.wantLeft {
				t.Errorf("Len() = %d, want %d", h.Len(), tt.wantLeft)
			}
			assertHeap(t, h)
		})
	}

	t.Run("zeroes vacated slots", func(t *testing.T) {
		h := FromFunc(ptrs(4, 1, 3, 2), byPtr)
		h.PopN(3)
		assertTailZeroed(t, h)
	})
}

func TestHeap_PushPopMethod(t *testing.T) {
	tests := []struct {
		name     string
		make     func() *Heap[int]
		v        int
		want     int
		wantHeap []int
	}{
		{name: "empty returns v untouched", make: New[int], v: 7, want: 7, wantHeap: nil},
		{name: "v below root returns v", make: func() *Heap[int] { return From([]int{5, 10, 20}) }, v: 1, want: 1, wantHeap: []int{5, 10, 20}},
		{name: "v equal to root returns v", make: func() *Heap[int] { return From([]int{5, 10, 20}) }, v: 5, want: 5, wantHeap: []int{5, 10, 20}},
		{name: "v above root replaces root", make: func() *Heap[int] { return From([]int{5, 10, 20}) }, v: 15, want: 5, wantHeap: []int{10, 15, 20}},
		{name: "max heap", make: func() *Heap[int] { return FromMax([]int{5, 10, 20}) }, v: 12, want: 20, wantHeap: []int{12, 10, 5}},
		{name: "max heap v above root returns v", make: func() *Heap[int] { return FromMax([]int{5, 10, 20}) }, v: 30, want: 30, wantHeap: []int{20, 10, 5}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := tt.make()
			capBefore := h.Cap()
			if got := h.PushPop(tt.v); got != tt.want {
				t.Errorf("PushPop(%d) = %d, want %d", tt.v, got, tt.want)
			}
			assertHeap(t, h)
			if got := h.Sorted(); !slices.Equal(got, tt.wantHeap) {
				t.Errorf("heap after PushPop = %v, want %v", got, tt.wantHeap)
			}
			if h.Cap() != capBefore {
				t.Errorf("Cap() = %d, want %d (PushPop must not grow)", h.Cap(), capBefore)
			}
		})
	}
}

func TestHeap_Replace(t *testing.T) {
	t.Run("empty stores v and reports false", func(t *testing.T) {
		h := New[int]()
		got, ok := h.Replace(7)
		if ok || got != 0 {
			t.Errorf("Replace(7) = %d, %v; want 0, false", got, ok)
		}
		if top, _ := h.Peek(); h.Len() != 1 || top != 7 {
			t.Errorf("heap = %v, want [7]", h.Slice())
		}
	})

	t.Run("pops old root even when v is smaller", func(t *testing.T) {
		h := From([]int{5, 10, 20})
		got, ok := h.Replace(1)
		if !ok || got != 5 {
			t.Errorf("Replace(1) = %d, %v; want 5, true", got, ok)
		}
		assertHeap(t, h)
		if want := []int{1, 10, 20}; !slices.Equal(h.Sorted(), want) {
			t.Errorf("heap = %v, want %v", h.Sorted(), want)
		}
	})

	t.Run("sifts larger v down", func(t *testing.T) {
		h := From([]int{5, 10, 20})
		got, ok := h.Replace(30)
		if !ok || got != 5 {
			t.Errorf("Replace(30) = %d, %v; want 5, true", got, ok)
		}
		assertHeap(t, h)
		if want := []int{10, 20, 30}; !slices.Equal(h.Sorted(), want) {
			t.Errorf("heap = %v, want %v", h.Sorted(), want)
		}
	})
}

func TestHeap_Contains(t *testing.T) {
	h := From([]int{4, 8, 15, 16, 23, 42})
	for _, v := range []int{4, 15, 42} {
		if !h.Contains(v) {
			t.Errorf("Contains(%d) = false, want true", v)
		}
	}
	if h.Contains(7) {
		t.Error("Contains(7) = true, want false")
	}
	if New[int]().Contains(1) {
		t.Error("empty heap: Contains(1) = true, want false")
	}

	t.Run("matches by comparator equality", func(t *testing.T) {
		type task struct {
			name     string
			priority int
		}
		h := NewFunc(func(a, b task) int { return cmp.Compare(a.priority, b.priority) })
		h.Push(task{name: "a", priority: 1})
		if !h.Contains(task{name: "other", priority: 1}) {
			t.Error("Contains(same priority) = false, want true")
		}
	})
}

func TestHeap_Remove(t *testing.T) {
	t.Run("moved element sifts up", func(t *testing.T) {
		// Removing 11 moves 4 under 10; without a sift-up the heap breaks.
		h := &Heap[int]{data: []int{1, 10, 2, 11, 12, 3, 4}, compare: cmp.Compare[int]}
		assertHeap(t, h)

		if !h.Remove(11) {
			t.Fatal("Remove(11) = false, want true")
		}
		assertHeap(t, h)
		if want := []int{1, 2, 3, 4, 10, 12}; !slices.Equal(drain(h), want) {
			t.Errorf("pop order mismatch, want %v", want)
		}
	})

	tests := []struct {
		name    string
		initial []int
		v       int
		found   bool
		want    []int
	}{
		{name: "root", initial: []int{1, 2, 3, 4}, v: 1, found: true, want: []int{2, 3, 4}},
		{name: "last element", initial: []int{1, 2, 3, 4}, v: 4, found: true, want: []int{1, 2, 3}},
		{name: "only element", initial: []int{9}, v: 9, found: true, want: nil},
		{name: "one of duplicates", initial: []int{2, 2, 1}, v: 2, found: true, want: []int{1, 2}},
		{name: "missing", initial: []int{1, 2, 3}, v: 7, found: false, want: []int{1, 2, 3}},
		{name: "empty", initial: nil, v: 1, found: false, want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := From(tt.initial)
			if got := h.Remove(tt.v); got != tt.found {
				t.Errorf("Remove(%d) = %v, want %v", tt.v, got, tt.found)
			}
			assertHeap(t, h)
			if got := drain(h); !slices.Equal(got, tt.want) {
				t.Errorf("remaining = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("zeroes vacated slot", func(t *testing.T) {
		vs := ptrs(1, 2, 3, 4)
		h := FromFunc(vs, byPtr)
		v := 3
		if !h.Remove(&v) {
			t.Fatal("Remove(3) = false, want true")
		}
		assertTailZeroed(t, h)
	})
}

func TestHeap_Randomized(t *testing.T) {
	r := rand.New(rand.NewPCG(42, 7))
	h := New[int]()
	var ref []int // sorted reference model

	for step := range 5000 {
		switch op := r.IntN(7); op {
		case 0, 1:
			v := r.IntN(100)
			h.Push(v)
			ref = append(ref, v)
		case 2:
			vs := make([]int, r.IntN(20))
			for i := range vs {
				vs[i] = r.IntN(100)
			}
			h.PushN(vs...)
			ref = append(ref, vs...)
		case 3:
			got, ok := h.Pop()
			if ok != (len(ref) > 0) || (ok && got != ref[0]) {
				t.Fatalf("step %d: Pop() = %d, %v; reference %v", step, got, ok, ref)
			}
			if ok {
				ref = ref[1:]
			}
		case 4:
			v := r.IntN(100)
			got := h.PushPop(v)
			ref = append(ref, v)
			slices.Sort(ref)
			if got != ref[0] {
				t.Fatalf("step %d: PushPop(%d) = %d, want %d", step, v, got, ref[0])
			}
			ref = ref[1:]
		case 5:
			v := r.IntN(100)
			got, ok := h.Replace(v)
			if ok != (len(ref) > 0) || (ok && got != ref[0]) {
				t.Fatalf("step %d: Replace(%d) = %d, %v; reference %v", step, v, got, ok, ref)
			}
			if ok {
				ref = ref[1:]
			}
			ref = append(ref, v)
		case 6:
			v := r.IntN(100)
			i := slices.Index(ref, v)
			if got := h.Remove(v); got != (i >= 0) {
				t.Fatalf("step %d: Remove(%d) = %v, want %v", step, v, got, i >= 0)
			}
			if i >= 0 {
				ref = slices.Delete(ref, i, i+1)
			}
		}
		slices.Sort(ref)

		assertHeap(t, h)
		if h.Len() != len(ref) {
			t.Fatalf("step %d: Len() = %d, want %d", step, h.Len(), len(ref))
		}
	}

	if got := h.Sorted(); !slices.Equal(got, ref) {
		t.Fatalf("final Sorted() = %v, want %v", got, ref)
	}
}

func TestHeap_All(t *testing.T) {
	h := From([]int{5, 3, 8, 1, 9})

	var got []int
	for v := range h.All() {
		got = append(got, v)
	}
	if !slices.Equal(got, h.data) {
		t.Errorf("All() = %v, want layout order %v", got, h.data)
	}
	if got[0] != 1 {
		t.Errorf("first element = %d, want least element 1", got[0])
	}

	count := 0
	for range h.All() {
		count++
		if count == 2 {
			break
		}
	}
	if count != 2 {
		t.Errorf("early break yielded %d elements, want 2", count)
	}
	if h.Len() != 5 {
		t.Errorf("Len() after All() = %d, want 5 (All must not consume)", h.Len())
	}
}

func TestHeap_Slice(t *testing.T) {
	h := From([]int{5, 3, 8, 1})
	out := h.Slice()
	if !slices.Equal(out, h.data) {
		t.Errorf("Slice() = %v, want layout order %v", out, h.data)
	}

	out[0] = 99
	if top, _ := h.Peek(); top != 1 {
		t.Errorf("Peek() after modifying Slice() = %d, want 1 (slice must be detached)", top)
	}
	if got := New[int]().Slice(); got != nil {
		t.Errorf("empty Slice() = %#v, want nil", got)
	}
}

func TestHeap_Sorted(t *testing.T) {
	h := From([]int{5, 3, 8, 1, 3})
	if want := []int{1, 3, 3, 5, 8}; !slices.Equal(h.Sorted(), want) {
		t.Errorf("Sorted() = %v, want %v", h.Sorted(), want)
	}
	if h.Len() != 5 {
		t.Errorf("Len() after Sorted() = %d, want 5", h.Len())
	}
	assertHeap(t, h)

	if want := []int{8, 5, 3, 1}; !slices.Equal(FromMax([]int{1, 8, 3, 5}).Sorted(), want) {
		t.Errorf("max Sorted() = %v, want %v", FromMax([]int{1, 8, 3, 5}).Sorted(), want)
	}
	if got := New[int]().Sorted(); got != nil {
		t.Errorf("empty Sorted() = %#v, want nil", got)
	}
}

func TestHeap_Clone(t *testing.T) {
	a := FromMax([]int{1, 2, 3})
	b := a.Clone()

	b.Pop()
	b.Push(0)
	if want := []int{3, 2, 1}; !slices.Equal(a.Sorted(), want) {
		t.Errorf("original = %v, want %v (clone must be independent)", a.Sorted(), want)
	}
	if want := []int{2, 1, 0}; !slices.Equal(b.Sorted(), want) {
		t.Errorf("clone = %v, want %v (clone must keep the max comparator)", b.Sorted(), want)
	}
}

func TestHeap_ClearReset(t *testing.T) {
	t.Run("Clear releases backing array and stays usable", func(t *testing.T) {
		h := NewMaxWithCap[int](8)
		h.PushN(1, 2)
		h.Clear()
		if h.Len() != 0 || h.Cap() != 0 {
			t.Errorf("after Clear: Len=%d Cap=%d, want 0 0", h.Len(), h.Cap())
		}
		h.PushN(1, 3, 2)
		if top, _ := h.Peek(); top != 3 {
			t.Errorf("Peek() after Clear+Push = %d, want 3 (comparator must survive)", top)
		}
	})

	t.Run("Reset keeps capacity and zeroes elements", func(t *testing.T) {
		h := NewFuncWithCap(8, byPtr)
		h.PushN(ptrs(1, 2, 3)...)
		h.Reset()
		if h.Len() != 0 || h.Cap() != 8 {
			t.Errorf("after Reset: Len=%d Cap=%d, want 0 8", h.Len(), h.Cap())
		}
		assertTailZeroed(t, h)
	})
}

func TestHeap_Capacity(t *testing.T) {
	h := NewWithCap[int](1024)
	h.PushN(3, 1, 2)

	h.Clip()
	if h.Len() != 3 || h.Cap() != 3 {
		t.Errorf("after Clip: Len=%d Cap=%d, want 3 3", h.Len(), h.Cap())
	}

	h.Grow(0)
	h.Grow(-1)
	if h.Cap() != 3 {
		t.Errorf("Grow(<=0) changed Cap to %d, want 3", h.Cap())
	}
	h.Grow(100)
	if h.Cap() < 103 {
		t.Errorf("after Grow(100): Cap=%d, want >= 103", h.Cap())
	}

	h.Shrink()
	if h.Len() != 3 || h.Cap() != 3 {
		t.Errorf("after Shrink: Len=%d Cap=%d, want 3 3", h.Len(), h.Cap())
	}
	h.Shrink() // no-op when tight

	if want := []int{1, 2, 3}; !slices.Equal(drain(h), want) {
		t.Errorf("heap order broken by capacity methods, want %v", want)
	}
}

func TestHeap_NilReceiverReads(t *testing.T) {
	var h *Heap[int]

	if h.Len() != 0 || h.Cap() != 0 || !h.IsEmpty() {
		t.Errorf("nil: Len=%d Cap=%d IsEmpty=%v, want 0 0 true", h.Len(), h.Cap(), h.IsEmpty())
	}
	if v, ok := h.Peek(); ok || v != 0 {
		t.Errorf("nil Peek() = %d, %v; want 0, false", v, ok)
	}
	if v, ok := h.Pop(); ok || v != 0 {
		t.Errorf("nil Pop() = %d, %v; want 0, false", v, ok)
	}
	if got := h.PopN(3); got != nil {
		t.Errorf("nil PopN(3) = %v, want nil", got)
	}
	if got := h.PushPop(5); got != 5 {
		t.Errorf("nil PushPop(5) = %d, want 5", got)
	}
	if h.Contains(1) || h.Remove(1) {
		t.Error("nil Contains/Remove reported true, want false")
	}
	if h.Slice() != nil || h.Sorted() != nil || h.Clone() != nil {
		t.Error("nil Slice/Sorted/Clone returned non-nil, want nil")
	}
	for range h.All() {
		t.Error("nil All() yielded an element")
	}
}

func TestHeap_NilReceiverMutatorsPanic(t *testing.T) {
	tests := []struct {
		name string
		op   string
		call func(h *Heap[int])
	}{
		{name: "Push", op: "Push", call: func(h *Heap[int]) { h.Push(1) }},
		{name: "PushN", op: "PushN", call: func(h *Heap[int]) { h.PushN(1, 2) }},
		{name: "PushN with no arguments", op: "PushN", call: func(h *Heap[int]) { h.PushN() }},
		{name: "Replace", op: "Replace", call: func(h *Heap[int]) { h.Replace(1) }},
		{name: "Grow", op: "Grow", call: func(h *Heap[int]) { h.Grow(8) }},
		{name: "Grow non-positive", op: "Grow", call: func(h *Heap[int]) { h.Grow(0) }},
		{name: "Clear", op: "Clear", call: func(h *Heap[int]) { h.Clear() }},
		{name: "Reset", op: "Reset", call: func(h *Heap[int]) { h.Reset() }},
		{name: "Shrink", op: "Shrink", call: func(h *Heap[int]) { h.Shrink() }},
		{name: "Clip", op: "Clip", call: func(h *Heap[int]) { h.Clip() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertPanics(t, tt.name, "binheap: "+tt.op+" on nil receiver", func() { tt.call(nil) })
		})
	}
}

func TestHeap_ZeroValue(t *testing.T) {
	var h Heap[int]

	if h.Len() != 0 || !h.IsEmpty() || h.Contains(1) || h.Remove(1) {
		t.Error("zero-value heap must behave as empty for reads and removals")
	}
	if _, ok := h.Pop(); ok {
		t.Error("zero-value Pop() ok = true, want false")
	}
	if got := h.PushPop(3); got != 3 {
		t.Errorf("zero-value PushPop(3) = %d, want 3", got)
	}

	const want = "on zero-value Heap"
	assertPanics(t, "zero-value Push", want, func() { h.Push(1) })
	assertPanics(t, "zero-value PushN", want, func() { h.PushN(1) })
	assertPanics(t, "zero-value Replace", want, func() { h.Replace(1) })
}
