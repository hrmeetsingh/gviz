package tui_test

import (
	"strings"
	"testing"

	"github.com/hrmeetsingh/gviz/internal/model"
	"github.com/hrmeetsingh/gviz/internal/tui"
)

func sampleTree() []*model.Goroutine {
	child := &model.Goroutine{
		ID:       2,
		ParentID: 1,
		State:    model.StateChanRecv,
		Label:    "producer",
		Stack:    []model.Frame{{Function: "main.producer", File: "/proj/main.go", Line: 20}},
	}
	root := &model.Goroutine{
		ID:       1,
		ParentID: -1,
		State:    model.StateRunning,
		Label:    "main",
		Stack:    []model.Frame{{Function: "main.main", File: "/proj/main.go", Line: 10}},
		Children: []*model.Goroutine{child},
	}
	return []*model.Goroutine{root}
}

func TestRenderTree_ContainsGoroutineIDs(t *testing.T) {
	roots := sampleTree()
	out := tui.RenderTree(roots, tui.RenderOptions{Width: 80})
	if !strings.Contains(out, "1") {
		t.Error("rendered output should contain goroutine ID 1")
	}
	if !strings.Contains(out, "2") {
		t.Error("rendered output should contain goroutine ID 2")
	}
}

func TestRenderTree_ContainsStateLabels(t *testing.T) {
	roots := sampleTree()
	out := tui.RenderTree(roots, tui.RenderOptions{Width: 80})
	if !strings.Contains(out, "running") {
		t.Error("rendered output should contain state 'running'")
	}
	if !strings.Contains(out, "chan receive") {
		t.Error("rendered output should contain state 'chan receive'")
	}
}

func TestRenderTree_ChildIndentedBelowParent(t *testing.T) {
	roots := sampleTree()
	out := tui.RenderTree(roots, tui.RenderOptions{Width: 80})
	lines := strings.Split(out, "\n")
	var parentLine, childLine int
	for i, l := range lines {
		if strings.Contains(l, "main.main") || strings.Contains(l, "goroutine 1") {
			parentLine = i
		}
		if strings.Contains(l, "producer") || strings.Contains(l, "goroutine 2") {
			childLine = i
		}
	}
	if childLine <= parentLine {
		t.Errorf("child (line %d) should appear after parent (line %d)", childLine, parentLine)
	}
}

func TestRenderTree_EmptyRoots(t *testing.T) {
	out := tui.RenderTree(nil, tui.RenderOptions{Width: 80})
	// Should not panic, returns empty or placeholder
	_ = out
}

func TestRenderTree_ChannelAnnotation(t *testing.T) {
	roots := sampleTree()
	sender := &model.Goroutine{
		ID:       3,
		ParentID: 1,
		State:    model.StateChanSend,
		Label:    "consumer",
		Channels: []model.Channel{{Direction: "send", PeerID: 2}},
	}
	roots[0].Children = append(roots[0].Children, sender)
	out := tui.RenderTree(roots, tui.RenderOptions{Width: 80, ShowChannels: true})
	// Channel annotation marker (dotted) should appear
	if !strings.Contains(out, "···") && !strings.Contains(out, "...") && !strings.Contains(out, "⋯") {
		t.Error("expected dotted channel line in output")
	}
}

func TestFilterGoroutines_ByState(t *testing.T) {
	gs := []*model.Goroutine{
		{ID: 1, State: model.StateRunning},
		{ID: 2, State: model.StateChanRecv},
		{ID: 3, State: model.StateChanSend},
	}
	result := tui.FilterGoroutines(gs, "chan")
	if len(result) != 2 {
		t.Errorf("want 2 filtered, got %d", len(result))
	}
}

func TestFilterGoroutines_ByLabel(t *testing.T) {
	gs := []*model.Goroutine{
		{ID: 1, Label: "main.main", State: model.StateRunning},
		{ID: 2, Label: "main.worker", State: model.StateChanRecv},
		{ID: 3, Label: "http.server", State: model.StateIOWait},
	}
	result := tui.FilterGoroutines(gs, "main")
	if len(result) != 2 {
		t.Errorf("want 2 filtered, got %d", len(result))
	}
}

func TestFilterGoroutines_EmptyQuery(t *testing.T) {
	gs := []*model.Goroutine{
		{ID: 1, State: model.StateRunning},
		{ID: 2, State: model.StateChanRecv},
	}
	result := tui.FilterGoroutines(gs, "")
	if len(result) != 2 {
		t.Errorf("empty query should return all, got %d", len(result))
	}
}
