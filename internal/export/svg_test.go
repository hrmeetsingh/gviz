package export_test

import (
	"strings"
	"testing"
	"time"

	"github.com/hrmeetsingh/gviz/internal/export"
	"github.com/hrmeetsingh/gviz/internal/model"
)

func makeSnapshot() *model.Snapshot {
	child := &model.Goroutine{
		ID: 2, ParentID: 1, State: model.StateChanRecv, Label: "main.worker",
		Stack: []model.Frame{{Function: "main.worker", File: "/proj/main.go", Line: 20}},
	}
	root := &model.Goroutine{
		ID: 1, ParentID: -1, State: model.StateRunning, Label: "main.main",
		Stack:    []model.Frame{{Function: "main.main", File: "/proj/main.go", Line: 10}},
		Children: []*model.Goroutine{child},
	}
	return &model.Snapshot{
		At:    time.Now(),
		Roots: []*model.Goroutine{root},
		ByID:  map[int64]*model.Goroutine{1: root, 2: child},
	}
}

func TestSVGRenderer_ProducesOutput(t *testing.T) {
	snap := makeSnapshot()
	r := export.NewSVGRenderer()
	out := r.Render(snap)
	if out == "" {
		t.Error("expected non-empty SVG output")
	}
}

func TestSVGRenderer_OutputIsSVG(t *testing.T) {
	snap := makeSnapshot()
	r := export.NewSVGRenderer()
	out := r.Render(snap)
	if !strings.HasPrefix(strings.TrimSpace(out), "<svg") {
		t.Errorf("expected output to start with <svg, got: %q", out[:min(50, len(out))])
	}
}

func TestSVGRenderer_ContainsGoroutineID(t *testing.T) {
	snap := makeSnapshot()
	r := export.NewSVGRenderer()
	out := r.Render(snap)
	if !strings.Contains(out, "goroutine 1") && !strings.Contains(out, "main.main") {
		t.Error("expected SVG to reference goroutine 1 or main.main")
	}
}

func TestSVGRenderer_ContainsChildGoroutine(t *testing.T) {
	snap := makeSnapshot()
	r := export.NewSVGRenderer()
	out := r.Render(snap)
	if !strings.Contains(out, "goroutine 2") && !strings.Contains(out, "main.worker") {
		t.Error("expected SVG to reference goroutine 2 or main.worker")
	}
}

func TestSVGRenderer_EmptySnapshot(t *testing.T) {
	snap := &model.Snapshot{At: time.Now(), Roots: nil, ByID: map[int64]*model.Goroutine{}}
	r := export.NewSVGRenderer()
	out := r.Render(snap)
	// Should not panic, return valid SVG
	if !strings.Contains(out, "<svg") {
		t.Error("expected valid SVG even for empty snapshot")
	}
}

func TestSVGRenderer_ClosedSVGTag(t *testing.T) {
	snap := makeSnapshot()
	r := export.NewSVGRenderer()
	out := r.Render(snap)
	if !strings.Contains(out, "</svg>") {
		t.Error("expected closing </svg> tag")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
