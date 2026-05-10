package attach

import (
	"fmt"

	"github.com/hrmeetsingh/gviz/internal/model"
)

// DelveLoc mirrors the location fields we need from delve's api.Location.
type DelveLoc struct {
	Function string
	File     string
	Line     int
}

// DelveFrame mirrors a single stack frame from delve's api.Stackframe.
type DelveFrame struct {
	Function string
	File     string
	Line     int
}

// DelveGoroutine is a minimal representation of a delve api.Goroutine.
// It holds only the fields gviz needs, allowing tests to use a mock client
// without importing the full delve API.
type DelveGoroutine struct {
	ID             int64
	Status         int    // matches proc.G* constants: 0=idle, 1=runnable, 2=running, 3=syscall, 4=waiting, 6=dead
	WaitReason     string // human-readable wait reason from delve
	ThreadID       int
	UserCurrentLoc DelveLoc
	GoStatementLoc DelveLoc
	AncestorGID    int64 // goroutine ID of creator, 0 if unknown
}

// DelveClient is the interface that wraps the delve RPC goroutine operations.
// The real implementation uses rpc2.RPCClient; tests use a mock.
type DelveClient interface {
	ListGoroutines() ([]*DelveGoroutine, error)
	Stacktrace(goroutineID int64, depth int) ([]DelveFrame, error)
	Close() error
}

// DelveFetcher implements Fetcher using a DelveClient.
type DelveFetcher struct {
	client DelveClient
}

// NewDelveFetcherFromClient wraps an existing DelveClient (used in tests).
func NewDelveFetcherFromClient(client DelveClient) *DelveFetcher {
	return &DelveFetcher{client: client}
}

const defaultStackDepth = 50

// Fetch retrieves goroutines via the Delve client, including full stack traces,
// and maps them to model.Goroutine.
func (f *DelveFetcher) Fetch() ([]*model.Goroutine, error) {
	dgs, err := f.client.ListGoroutines()
	if err != nil {
		return nil, fmt.Errorf("delve ListGoroutines: %w", err)
	}
	result := make([]*model.Goroutine, 0, len(dgs))
	for _, dg := range dgs {
		g := delveToModel(dg)
		if frames, err := f.client.Stacktrace(dg.ID, defaultStackDepth); err == nil && len(frames) > 0 {
			g.Stack = make([]model.Frame, len(frames))
			for i, fr := range frames {
				g.Stack[i] = model.Frame{Function: fr.Function, File: fr.File, Line: fr.Line}
			}
			g.Label = model.EntryLabel(g.Stack)
		}
		result = append(result, g)
	}
	return result, nil
}

func delveToModel(dg *DelveGoroutine) *model.Goroutine {
	g := &model.Goroutine{
		ID:       dg.ID,
		ParentID: -1,
		ThreadID: dg.ThreadID,
		State:    delveState(dg),
		Label:    dg.UserCurrentLoc.Function,
	}
	if dg.WaitReason != "" {
		g.WaitReason = dg.WaitReason
	}
	if dg.AncestorGID > 0 {
		g.ParentID = dg.AncestorGID
	}
	if dg.UserCurrentLoc.Function != "" {
		g.Stack = []model.Frame{{
			Function: dg.UserCurrentLoc.Function,
			File:     dg.UserCurrentLoc.File,
			Line:     dg.UserCurrentLoc.Line,
		}}
	}
	return g
}

// delveState maps Delve's goroutine status + wait reason to our GoroutineState.
// When the goroutine is waiting (status 4), the wait reason string gives the
// specific state (chan send, chan receive, select, sleep, etc.).
func delveState(dg *DelveGoroutine) model.GoroutineState {
	if dg.Status == 4 && dg.WaitReason != "" {
		return model.GoroutineState(dg.WaitReason)
	}
	switch dg.Status {
	case 2:
		return model.StateRunning
	case 3:
		return model.StateSyscall
	default:
		return model.StateWaiting
	}
}


// AutoDetectWithDelve is like AutoDetect but accepts a pre-built DelveClient
// for testing (avoids needing a live process).
func AutoDetectWithDelve(cfg AutoDetectConfig, delveClient DelveClient) (Fetcher, error) {
	if cfg.PprofURL != "" {
		f := NewPprofFetcher(cfg.PprofURL)
		if _, err := f.Fetch(); err == nil {
			return f, nil
		}
	}
	if delveClient != nil {
		return NewDelveFetcherFromClient(delveClient), nil
	}
	return nil, fmt.Errorf("auto-detect: no reachable attach point")
}
