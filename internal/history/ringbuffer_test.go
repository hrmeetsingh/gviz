package history_test

import (
	"testing"

	"github.com/harmeetsingh/gviz/internal/history"
)

func TestRingBuffer_PushAndGet(t *testing.T) {
	rb := history.NewRingBuffer[int](5)
	rb.Push(10)
	rb.Push(20)
	if rb.Len() != 2 {
		t.Fatalf("want Len=2, got %d", rb.Len())
	}
	all := rb.All()
	if all[0] != 10 || all[1] != 20 {
		t.Errorf("want [10 20], got %v", all)
	}
}

func TestRingBuffer_WrapAround(t *testing.T) {
	rb := history.NewRingBuffer[int](3)
	rb.Push(1)
	rb.Push(2)
	rb.Push(3)
	rb.Push(4) // overwrites 1
	all := rb.All()
	if len(all) != 3 {
		t.Fatalf("want len=3 after wrap, got %d", len(all))
	}
	if all[0] != 2 || all[1] != 3 || all[2] != 4 {
		t.Errorf("want [2 3 4], got %v", all)
	}
}

func TestRingBuffer_LenCappedAtCapacity(t *testing.T) {
	rb := history.NewRingBuffer[int](3)
	for i := 0; i < 10; i++ {
		rb.Push(i)
	}
	if rb.Len() != 3 {
		t.Errorf("want Len=3 (capacity), got %d", rb.Len())
	}
}

func TestRingBuffer_Latest(t *testing.T) {
	rb := history.NewRingBuffer[int](5)
	rb.Push(100)
	rb.Push(200)
	v, ok := rb.Latest()
	if !ok {
		t.Fatal("want ok=true, got false")
	}
	if v != 200 {
		t.Errorf("want Latest=200, got %d", v)
	}
}

func TestRingBuffer_LatestOnEmpty(t *testing.T) {
	rb := history.NewRingBuffer[int](5)
	_, ok := rb.Latest()
	if ok {
		t.Error("want ok=false on empty ring buffer")
	}
}

func TestRingBuffer_AllEmpty(t *testing.T) {
	rb := history.NewRingBuffer[int](5)
	all := rb.All()
	if len(all) != 0 {
		t.Errorf("want empty slice, got %v", all)
	}
}

func TestRingBuffer_CapacityOne(t *testing.T) {
	rb := history.NewRingBuffer[string](1)
	rb.Push("a")
	rb.Push("b")
	all := rb.All()
	if len(all) != 1 || all[0] != "b" {
		t.Errorf("want [b], got %v", all)
	}
}
