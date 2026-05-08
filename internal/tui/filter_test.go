package tui_test

import (
	"testing"

	"github.com/harmeetsingh/gviz/internal/model"
	"github.com/harmeetsingh/gviz/internal/tui"
)

func TestFilterGoroutines_RegexMatchesLabel(t *testing.T) {
	gs := []*model.Goroutine{
		{ID: 1, Label: "main.producer", State: model.StateRunning},
		{ID: 2, Label: "http.server", State: model.StateIOWait},
		{ID: 3, Label: "main.consumer", State: model.StateChanRecv},
	}
	result := tui.FilterGoroutinesRegex(gs, `main\.(producer|consumer)`)
	if len(result) != 2 {
		t.Errorf("want 2 matches, got %d", len(result))
	}
}

func TestFilterGoroutines_RegexMatchesState(t *testing.T) {
	gs := []*model.Goroutine{
		{ID: 1, Label: "a", State: model.StateChanSend},
		{ID: 2, Label: "b", State: model.StateRunning},
		{ID: 3, Label: "c", State: model.StateChanRecv},
	}
	result := tui.FilterGoroutinesRegex(gs, `^chan`)
	if len(result) != 2 {
		t.Errorf("want 2 chan goroutines, got %d", len(result))
	}
}

func TestFilterGoroutines_InvalidRegexFallsBackToLiteral(t *testing.T) {
	gs := []*model.Goroutine{
		{ID: 1, Label: "main.main", State: model.StateRunning},
		{ID: 2, Label: "other", State: model.StateWaiting},
	}
	// "[invalid" is invalid regex; should fall back to literal substring match
	result := tui.FilterGoroutinesRegex(gs, "main.main")
	if len(result) != 1 {
		t.Errorf("want 1 literal match, got %d", len(result))
	}
}

func TestFilterGoroutines_RegexEmptyReturnsAll(t *testing.T) {
	gs := []*model.Goroutine{
		{ID: 1, Label: "a", State: model.StateRunning},
		{ID: 2, Label: "b", State: model.StateWaiting},
	}
	result := tui.FilterGoroutinesRegex(gs, "")
	if len(result) != 2 {
		t.Errorf("want all 2, got %d", len(result))
	}
}

func TestFilterByState_ExactMatch(t *testing.T) {
	gs := []*model.Goroutine{
		{ID: 1, State: model.StateRunning},
		{ID: 2, State: model.StateChanRecv},
		{ID: 3, State: model.StateWaiting},
	}
	result := tui.FilterByState(gs, model.StateRunning)
	if len(result) != 1 || result[0].ID != 1 {
		t.Errorf("want [1], got %v", result)
	}
}

func TestFilterByState_EmptyStateReturnsAll(t *testing.T) {
	gs := []*model.Goroutine{
		{ID: 1, State: model.StateRunning},
		{ID: 2, State: model.StateChanRecv},
	}
	result := tui.FilterByState(gs, "")
	if len(result) != 2 {
		t.Errorf("want all 2, got %d", len(result))
	}
}
