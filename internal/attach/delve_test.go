package attach_test

import (
	"errors"
	"testing"

	"github.com/harmeetsingh/gviz/internal/attach"
	"github.com/harmeetsingh/gviz/internal/model"
)

var errMock = errors.New("mock error")

// mockDelveClient implements attach.DelveClient for testing.
type mockDelveClient struct {
	goroutines []*attach.DelveGoroutine
	err        error
}

func (m *mockDelveClient) ListGoroutines() ([]*attach.DelveGoroutine, error) {
	return m.goroutines, m.err
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
		{0, model.StateRunning},
		{1, model.StateWaiting},
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
