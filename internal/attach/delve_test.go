package attach_test

import (
	"errors"
	"testing"

	"github.com/hrmeetsingh/gviz/internal/attach"
	"github.com/hrmeetsingh/gviz/internal/model"
)

var errMock = errors.New("mock error")

// mockDelveClient implements attach.DelveClient for testing.
type mockDelveClient struct {
	goroutines []*attach.DelveGoroutine
	stacks     map[int64][]attach.DelveFrame // goroutine ID → stack frames
	err        error
}

func (m *mockDelveClient) ListGoroutines() ([]*attach.DelveGoroutine, error) {
	return m.goroutines, m.err
}

func (m *mockDelveClient) Stacktrace(goroutineID int64, depth int) ([]attach.DelveFrame, error) {
	if m.stacks != nil {
		if s, ok := m.stacks[goroutineID]; ok {
			return s, nil
		}
	}
	return nil, nil
}

func (m *mockDelveClient) Close() error { return nil }

func TestDelveFetcher_FetchReturnsGoroutines(t *testing.T) {
	mock := &mockDelveClient{
		goroutines: []*attach.DelveGoroutine{
			{ID: 1, Status: 0, UserCurrentLoc: attach.DelveLoc{Function: "main.main", File: "/proj/main.go", Line: 10}},
			{ID: 2, Status: 1, UserCurrentLoc: attach.DelveLoc{Function: "main.worker", File: "/proj/main.go", Line: 20}},
		},
	}
	f := attach.NewDelveFetcherFromClient(mock)
	goroutines, err := f.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(goroutines) != 2 {
		t.Fatalf("want 2 goroutines, got %d", len(goroutines))
	}
}

func TestDelveFetcher_MapsIDCorrectly(t *testing.T) {
	mock := &mockDelveClient{
		goroutines: []*attach.DelveGoroutine{
			{ID: 42, Status: 0, UserCurrentLoc: attach.DelveLoc{Function: "main.main", File: "/proj/main.go", Line: 5}},
		},
	}
	f := attach.NewDelveFetcherFromClient(mock)
	goroutines, err := f.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if goroutines[0].ID != 42 {
		t.Errorf("want ID=42, got %d", goroutines[0].ID)
	}
}

func TestDelveFetcher_MapsLabelFromFunction(t *testing.T) {
	mock := &mockDelveClient{
		goroutines: []*attach.DelveGoroutine{
			{ID: 1, UserCurrentLoc: attach.DelveLoc{Function: "main.producer", File: "/proj/main.go", Line: 10}},
		},
	}
	f := attach.NewDelveFetcherFromClient(mock)
	goroutines, _ := f.Fetch()
	if goroutines[0].Label != "main.producer" {
		t.Errorf("want label=main.producer, got %q", goroutines[0].Label)
	}
}

func TestDelveFetcher_PropagatesClientError(t *testing.T) {
	mock := &mockDelveClient{err: errMock}
	f := attach.NewDelveFetcherFromClient(mock)
	_, err := f.Fetch()
	if err == nil {
		t.Error("expected error from client, got nil")
	}
}

func TestDelveFetcher_StatusToState(t *testing.T) {
	cases := []struct {
		status int
		want   model.GoroutineState
	}{
		{2, model.StateRunning},  // Grunning
		{3, model.StateSyscall},  // Gsyscall
		{4, model.StateWaiting},  // Gwaiting (no wait reason → generic waiting)
		{0, model.StateWaiting},  // Gidle
	}
	for _, c := range cases {
		mock := &mockDelveClient{
			goroutines: []*attach.DelveGoroutine{
				{ID: 1, Status: c.status, UserCurrentLoc: attach.DelveLoc{Function: "f"}},
			},
		}
		f := attach.NewDelveFetcherFromClient(mock)
		gs, _ := f.Fetch()
		if gs[0].State != c.want {
			t.Errorf("status %d: want state=%q, got %q", c.status, c.want, gs[0].State)
		}
	}
}

func TestAutoDetect_UsesDelveFetcherWhenPprofFails(t *testing.T) {
	mock := &mockDelveClient{
		goroutines: []*attach.DelveGoroutine{
			{ID: 1, UserCurrentLoc: attach.DelveLoc{Function: "main.main"}},
		},
	}
	fetcher, err := attach.AutoDetectWithDelve(attach.AutoDetectConfig{
		PprofURL: "http://127.0.0.1:1", // unreachable
	}, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gs, err := fetcher.Fetch()
	if err != nil || len(gs) == 0 {
		t.Errorf("expected goroutines from delve fallback, got err=%v len=%d", err, len(gs))
	}
}

// --- New tests for enhanced Delve mapping ---

func TestDelveFetcher_FullStackFromStacktrace(t *testing.T) {
	mock := &mockDelveClient{
		goroutines: []*attach.DelveGoroutine{
			{ID: 5, Status: 4, WaitReason: "chan receive",
				UserCurrentLoc: attach.DelveLoc{Function: "runtime.chanrecv1", File: "/go/src/runtime/chan.go", Line: 442},
				GoStatementLoc: attach.DelveLoc{Function: "main.main", File: "/proj/main.go", Line: 15},
			},
		},
		stacks: map[int64][]attach.DelveFrame{
			5: {
				{Function: "runtime.chanrecv1", File: "/go/src/runtime/chan.go", Line: 442},
				{Function: "main.worker", File: "/proj/main.go", Line: 20},
			},
		},
	}
	f := attach.NewDelveFetcherFromClient(mock)
	gs, err := f.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	g := gs[0]
	if len(g.Stack) != 2 {
		t.Fatalf("want 2 stack frames, got %d", len(g.Stack))
	}
	if g.Stack[0].Function != "runtime.chanrecv1" {
		t.Errorf("frame 0: want runtime.chanrecv1, got %q", g.Stack[0].Function)
	}
	if g.Stack[1].Function != "main.worker" {
		t.Errorf("frame 1: want main.worker, got %q", g.Stack[1].Function)
	}
}

func TestDelveFetcher_WaitReasonMapped(t *testing.T) {
	mock := &mockDelveClient{
		goroutines: []*attach.DelveGoroutine{
			{ID: 5, Status: 4, WaitReason: "chan receive",
				UserCurrentLoc: attach.DelveLoc{Function: "main.worker"},
			},
		},
	}
	f := attach.NewDelveFetcherFromClient(mock)
	gs, _ := f.Fetch()
	if gs[0].State != model.StateChanRecv {
		t.Errorf("want state=%q, got %q", model.StateChanRecv, gs[0].State)
	}
}

func TestDelveFetcher_WaitReasonChanSend(t *testing.T) {
	mock := &mockDelveClient{
		goroutines: []*attach.DelveGoroutine{
			{ID: 6, Status: 4, WaitReason: "chan send",
				UserCurrentLoc: attach.DelveLoc{Function: "main.dispatcher"},
			},
		},
	}
	f := attach.NewDelveFetcherFromClient(mock)
	gs, _ := f.Fetch()
	if gs[0].State != model.StateChanSend {
		t.Errorf("want state=%q, got %q", model.StateChanSend, gs[0].State)
	}
}

func TestDelveFetcher_WaitReasonSelect(t *testing.T) {
	mock := &mockDelveClient{
		goroutines: []*attach.DelveGoroutine{
			{ID: 7, Status: 4, WaitReason: "select",
				UserCurrentLoc: attach.DelveLoc{Function: "main.contextWorker"},
			},
		},
	}
	f := attach.NewDelveFetcherFromClient(mock)
	gs, _ := f.Fetch()
	if gs[0].State != model.StateSelect {
		t.Errorf("want state=%q, got %q", model.StateSelect, gs[0].State)
	}
}

func TestDelveFetcher_WaitReasonSleep(t *testing.T) {
	mock := &mockDelveClient{
		goroutines: []*attach.DelveGoroutine{
			{ID: 8, Status: 4, WaitReason: "sleep",
				UserCurrentLoc: attach.DelveLoc{Function: "main.sleeper"},
			},
		},
	}
	f := attach.NewDelveFetcherFromClient(mock)
	gs, _ := f.Fetch()
	if gs[0].State != model.StateSleep {
		t.Errorf("want state=%q, got %q", model.StateSleep, gs[0].State)
	}
}

func TestDelveFetcher_LabelFromEntryFunction(t *testing.T) {
	mock := &mockDelveClient{
		goroutines: []*attach.DelveGoroutine{
			{ID: 5, Status: 4, WaitReason: "chan receive",
				UserCurrentLoc: attach.DelveLoc{Function: "runtime.chanrecv1"},
				GoStatementLoc: attach.DelveLoc{Function: "main.main", File: "/proj/main.go", Line: 15},
			},
		},
		stacks: map[int64][]attach.DelveFrame{
			5: {
				{Function: "runtime.chanrecv1", File: "/go/src/runtime/chan.go", Line: 442},
				{Function: "main.worker", File: "/proj/main.go", Line: 20},
			},
		},
	}
	f := attach.NewDelveFetcherFromClient(mock)
	gs, _ := f.Fetch()
	// Label should come from the entry function (deepest non-runtime frame), not top-of-stack
	if gs[0].Label != "main.worker" {
		t.Errorf("want label=main.worker, got %q", gs[0].Label)
	}
}

func TestDelveFetcher_ParentIDFromGoStatement(t *testing.T) {
	mock := &mockDelveClient{
		goroutines: []*attach.DelveGoroutine{
			{ID: 1, Status: 0,
				UserCurrentLoc: attach.DelveLoc{Function: "main.main"},
				GoStatementLoc: attach.DelveLoc{},
			},
			{ID: 5, Status: 4, WaitReason: "chan receive",
				UserCurrentLoc: attach.DelveLoc{Function: "main.worker"},
				GoStatementLoc: attach.DelveLoc{Function: "main.main", File: "/proj/main.go", Line: 15},
				AncestorGID:    1,
			},
		},
	}
	f := attach.NewDelveFetcherFromClient(mock)
	gs, _ := f.Fetch()
	byID := make(map[int64]*model.Goroutine)
	for _, g := range gs {
		byID[g.ID] = g
	}
	if byID[5].ParentID != 1 {
		t.Errorf("goroutine 5: want parentID=1, got %d", byID[5].ParentID)
	}
}

func TestDelveFetcher_ThreadIDPopulated(t *testing.T) {
	mock := &mockDelveClient{
		goroutines: []*attach.DelveGoroutine{
			{ID: 1, Status: 0, ThreadID: 12345,
				UserCurrentLoc: attach.DelveLoc{Function: "main.main"},
			},
		},
	}
	f := attach.NewDelveFetcherFromClient(mock)
	gs, _ := f.Fetch()
	if gs[0].ThreadID != 12345 {
		t.Errorf("want ThreadID=12345, got %d", gs[0].ThreadID)
	}
}
