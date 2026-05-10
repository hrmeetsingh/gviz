package analyzer_test

import (
	"testing"
	"time"

	"github.com/hrmeetsingh/gviz/internal/analyzer"
	"github.com/hrmeetsingh/gviz/internal/model"
)

func makeGoroutine(id, parentID int64, state model.GoroutineState) *model.Goroutine {
	return &model.Goroutine{
		ID:        id,
		ParentID:  parentID,
		State:     state,
		CreatedAt: time.Now(),
	}
}

// --- BuildTree ---

func TestBuildTree_RootsHaveNoParent(t *testing.T) {
	goroutines := []*model.Goroutine{
		makeGoroutine(1, -1, model.StateRunning),
		makeGoroutine(2, 1, model.StateChanRecv),
		makeGoroutine(3, 1, model.StateChanSend),
	}
	roots := analyzer.BuildTree(goroutines)
	if len(roots) != 1 {
		t.Fatalf("want 1 root, got %d", len(roots))
	}
	if roots[0].ID != 1 {
		t.Errorf("want root ID=1, got %d", roots[0].ID)
	}
}

func TestBuildTree_ChildrenAttachedToParent(t *testing.T) {
	goroutines := []*model.Goroutine{
		makeGoroutine(1, -1, model.StateRunning),
		makeGoroutine(2, 1, model.StateChanRecv),
		makeGoroutine(3, 2, model.StateSelect),
	}
	roots := analyzer.BuildTree(goroutines)
	if len(roots[0].Children) != 1 {
		t.Fatalf("want 1 child of root, got %d", len(roots[0].Children))
	}
	child := roots[0].Children[0]
	if child.ID != 2 {
		t.Errorf("want child ID=2, got %d", child.ID)
	}
	if len(child.Children) != 1 || child.Children[0].ID != 3 {
		t.Errorf("want child 2 to have child 3")
	}
}

func TestBuildTree_OrphanBecomesRoot(t *testing.T) {
	// goroutine referencing a parent not in the list → treated as root
	goroutines := []*model.Goroutine{
		makeGoroutine(5, 99, model.StateRunning), // parent 99 doesn't exist
	}
	roots := analyzer.BuildTree(goroutines)
	if len(roots) != 1 {
		t.Fatalf("want 1 root (orphan promoted), got %d", len(roots))
	}
}

func TestBuildTree_MultipleRoots(t *testing.T) {
	goroutines := []*model.Goroutine{
		makeGoroutine(1, -1, model.StateRunning),
		makeGoroutine(2, -1, model.StateRunning),
	}
	roots := analyzer.BuildTree(goroutines)
	if len(roots) != 2 {
		t.Fatalf("want 2 roots, got %d", len(roots))
	}
}

// --- Diff (snapshot tracker) ---

func TestDiff_DetectsNewGoroutines(t *testing.T) {
	prev := []*model.Goroutine{
		makeGoroutine(1, -1, model.StateRunning),
	}
	curr := []*model.Goroutine{
		makeGoroutine(1, -1, model.StateRunning),
		makeGoroutine(2, 1, model.StateChanRecv),
	}
	newIDs, endedIDs := analyzer.Diff(prev, curr)
	if len(newIDs) != 1 || newIDs[0] != 2 {
		t.Errorf("want newIDs=[2], got %v", newIDs)
	}
	if len(endedIDs) != 0 {
		t.Errorf("want endedIDs=[], got %v", endedIDs)
	}
}

func TestDiff_DetectsEndedGoroutines(t *testing.T) {
	prev := []*model.Goroutine{
		makeGoroutine(1, -1, model.StateRunning),
		makeGoroutine(2, 1, model.StateChanRecv),
	}
	curr := []*model.Goroutine{
		makeGoroutine(1, -1, model.StateRunning),
	}
	newIDs, endedIDs := analyzer.Diff(prev, curr)
	if len(newIDs) != 0 {
		t.Errorf("want newIDs=[], got %v", newIDs)
	}
	if len(endedIDs) != 1 || endedIDs[0] != 2 {
		t.Errorf("want endedIDs=[2], got %v", endedIDs)
	}
}

func TestDiff_NoDifference(t *testing.T) {
	gs := []*model.Goroutine{
		makeGoroutine(1, -1, model.StateRunning),
		makeGoroutine(2, 1, model.StateChanRecv),
	}
	newIDs, endedIDs := analyzer.Diff(gs, gs)
	if len(newIDs) != 0 || len(endedIDs) != 0 {
		t.Errorf("want no diff, got new=%v ended=%v", newIDs, endedIDs)
	}
}

// --- InferChannelPairs ---

func TestInferChannelPairs_PairsSendAndRecv(t *testing.T) {
	goroutines := []*model.Goroutine{
		{ID: 1, ParentID: -1, State: model.StateRunning},
		{ID: 2, ParentID: 1, State: model.StateChanSend,
			Stack: []model.Frame{{Function: "main.producer"}}},
		{ID: 3, ParentID: 1, State: model.StateChanRecv,
			Stack: []model.Frame{{Function: "main.consumer"}}},
	}
	pairs := analyzer.InferChannelPairs(goroutines)
	// At minimum, both chan-active goroutines should appear in a pair
	if len(pairs) == 0 {
		t.Error("expected at least one channel pair inferred")
	}
}

// --- BuildSnapshot ---

func TestBuildSnapshot_PopulatesFields(t *testing.T) {
	goroutines := []*model.Goroutine{
		makeGoroutine(1, -1, model.StateRunning),
		makeGoroutine(2, 1, model.StateChanRecv),
	}
	snap := analyzer.BuildSnapshot(goroutines, nil)
	if len(snap.ByID) != 2 {
		t.Errorf("want 2 entries in ByID, got %d", len(snap.ByID))
	}
	if len(snap.Roots) != 1 {
		t.Errorf("want 1 root, got %d", len(snap.Roots))
	}
	if snap.At.IsZero() {
		t.Error("snapshot At time should not be zero")
	}
}

func TestBuildSnapshot_WiresChannelPairs(t *testing.T) {
	goroutines := []*model.Goroutine{
		makeGoroutine(1, -1, model.StateRunning),
		makeGoroutine(2, 1, model.StateChanSend),
		makeGoroutine(3, 1, model.StateChanRecv),
	}
	snap := analyzer.BuildSnapshot(goroutines, nil)

	sender := snap.ByID[2]
	if len(sender.Channels) == 0 {
		t.Fatal("sender goroutine should have at least one Channel entry")
	}
	if sender.Channels[0].Direction != "send" {
		t.Errorf("sender direction: want \"send\", got %q", sender.Channels[0].Direction)
	}
	if sender.Channels[0].PeerID != 3 {
		t.Errorf("sender peer: want 3, got %d", sender.Channels[0].PeerID)
	}

	receiver := snap.ByID[3]
	if len(receiver.Channels) == 0 {
		t.Fatal("receiver goroutine should have at least one Channel entry")
	}
	if receiver.Channels[0].Direction != "recv" {
		t.Errorf("receiver direction: want \"recv\", got %q", receiver.Channels[0].Direction)
	}
	if receiver.Channels[0].PeerID != 2 {
		t.Errorf("receiver peer: want 2, got %d", receiver.Channels[0].PeerID)
	}
}

func TestBuildSnapshot_NoChannelsForNonChanState(t *testing.T) {
	goroutines := []*model.Goroutine{
		makeGoroutine(1, -1, model.StateRunning),
		makeGoroutine(2, 1, model.StateSleep),
	}
	snap := analyzer.BuildSnapshot(goroutines, nil)
	sleeper := snap.ByID[2]
	if len(sleeper.Channels) != 0 {
		t.Errorf("sleep goroutine should have no channels, got %d", len(sleeper.Channels))
	}
}

func TestBuildSnapshot_DiffFromPrev(t *testing.T) {
	prev := &model.Snapshot{
		ByID: map[int64]*model.Goroutine{
			1: makeGoroutine(1, -1, model.StateRunning),
		},
	}
	curr := []*model.Goroutine{
		makeGoroutine(1, -1, model.StateRunning),
		makeGoroutine(2, 1, model.StateChanRecv),
	}
	snap := analyzer.BuildSnapshot(curr, prev)
	if len(snap.NewIDs) != 1 || snap.NewIDs[0] != 2 {
		t.Errorf("want NewIDs=[2], got %v", snap.NewIDs)
	}
	if len(snap.EndedIDs) != 0 {
		t.Errorf("want EndedIDs=[], got %v", snap.EndedIDs)
	}
}
