package attach

import (
	"fmt"

	"github.com/harmeetsingh/gviz/internal/model"
)

// DelveLoc mirrors the location fields we need from delve's api.Location.
type DelveLoc struct {
	Function string
	File     string
	Line     int
}

// DelveGoroutine is a minimal representation of a delve api.Goroutine.
// It holds only the fields gviz needs, allowing tests to use a mock client
// without importing the full delve API.
type DelveGoroutine struct {
	ID             int64
	Status         int // 0=running, 1=waiting, 2=syscall, 3=dead
	UserCurrentLoc DelveLoc
	GoStatementLoc DelveLoc
}

// DelveClient is the interface that wraps the delve RPC goroutine listing.
// The real implementation uses rpc2.RPCClient; tests use a mock.
type DelveClient interface {
	ListGoroutines() ([]*DelveGoroutine, error)
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

// Fetch retrieves goroutines via the Delve client and maps them to model.Goroutine.
func (f *DelveFetcher) Fetch() ([]*model.Goroutine, error) {
	dgs, err := f.client.ListGoroutines()
	if err != nil {
		return nil, fmt.Errorf("delve ListGoroutines: %w", err)
	}
	result := make([]*model.Goroutine, 0, len(dgs))
	for _, dg := range dgs {
		result = append(result, delveToModel(dg))
	}
	return result, nil
}

func delveToModel(dg *DelveGoroutine) *model.Goroutine {
	g := &model.Goroutine{
		ID:       dg.ID,
		ParentID: -1,
		Label:    dg.UserCurrentLoc.Function,
		State:    delveStatusToState(dg.Status),
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

func delveStatusToState(status int) model.GoroutineState {
	switch status {
	case 0:
		return model.StateRunning
	case 1:
		return model.StateWaiting
	case 2:
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
