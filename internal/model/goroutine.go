package model

import "time"

// GoroutineState represents the runtime state of a goroutine.
type GoroutineState string

const (
	StateRunning    GoroutineState = "running"
	StateWaiting    GoroutineState = "waiting"
	StateChanRecv   GoroutineState = "chan receive"
	StateChanSend   GoroutineState = "chan send"
	StateSyscall    GoroutineState = "syscall"
	StateSleep      GoroutineState = "sleep"
	StateSelect     GoroutineState = "select"
	StateIOWait     GoroutineState = "IO wait"
	StateSemAcquire GoroutineState = "semacquire"
	StateFinished   GoroutineState = "finished"
)

// Frame is a single stack frame.
type Frame struct {
	Function string
	File     string
	Line     int
}

// Channel represents an inferred channel relationship.
type Channel struct {
	Direction string // "send" or "recv"
	PeerID    int64  // goroutine ID on the other end, -1 if unknown
}

// Goroutine is the core data structure.
type Goroutine struct {
	ID         int64
	State      GoroutineState
	Label      string
	WaitReason string
	ParentID   int64 // -1 if no known parent
	ThreadID   int   // OS thread ID (from Delve), 0 if unknown
	Stack      []Frame
	Channels   []Channel
	CtxTimeout *time.Duration // nil if not detectable
	CreatedAt  time.Time
	EndedAt    *time.Time // nil if still alive
	Children   []*Goroutine
}

// Snapshot captures all goroutines at a point in time.
type Snapshot struct {
	At       time.Time
	Roots    []*Goroutine          // goroutines with no known parent
	ByID     map[int64]*Goroutine  // all goroutines indexed by ID
	NewIDs   []int64               // appeared since previous snapshot
	EndedIDs []int64               // gone since previous snapshot
}
