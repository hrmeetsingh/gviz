package history

// RingBuffer is a fixed-capacity circular buffer. When full, new elements
// overwrite the oldest. The zero value is not usable; use NewRingBuffer.
type RingBuffer[T any] struct {
	buf  []T
	cap  int
	head int // index of the oldest element
	len  int
}

// NewRingBuffer creates a RingBuffer with the given capacity. Panics if cap < 1.
func NewRingBuffer[T any](cap int) *RingBuffer[T] {
	if cap < 1 {
		panic("history.NewRingBuffer: capacity must be >= 1")
	}
	return &RingBuffer[T]{buf: make([]T, cap), cap: cap}
}

// Push adds v to the buffer, evicting the oldest element if full.
func (rb *RingBuffer[T]) Push(v T) {
	if rb.len < rb.cap {
		rb.buf[(rb.head+rb.len)%rb.cap] = v
		rb.len++
	} else {
		// Overwrite oldest slot and advance head.
		rb.buf[rb.head] = v
		rb.head = (rb.head + 1) % rb.cap
	}
}

// Len returns the number of elements currently stored.
func (rb *RingBuffer[T]) Len() int { return rb.len }

// All returns all elements in insertion order (oldest first).
func (rb *RingBuffer[T]) All() []T {
	if rb.len == 0 {
		return nil
	}
	out := make([]T, rb.len)
	for i := 0; i < rb.len; i++ {
		out[i] = rb.buf[(rb.head+i)%rb.cap]
	}
	return out
}

// Latest returns the most recently pushed element and true, or the zero value
// and false if the buffer is empty.
func (rb *RingBuffer[T]) Latest() (T, bool) {
	if rb.len == 0 {
		var zero T
		return zero, false
	}
	return rb.buf[(rb.head+rb.len-1)%rb.cap], true
}
