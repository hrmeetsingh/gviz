package profiles_test

import (
	"testing"

	"github.com/hrmeetsingh/gviz/internal/profiles"
)

// BuildCallTree is tested with a hand-crafted sample set since parsing the
// binary pprof CPU format requires a real profile. We test the tree structure
// and weight aggregation logic directly.

func TestCallTree_SingleSample(t *testing.T) {
	samples := []profiles.CPUSample{
		{Stack: []string{"main.main", "runtime.goexit"}, Count: 3},
	}
	tree := profiles.BuildCallTree(samples)
	if tree == nil {
		t.Fatal("expected non-nil call tree")
	}
}

func TestCallTree_AggregatesWeightAtRoot(t *testing.T) {
	samples := []profiles.CPUSample{
		{Stack: []string{"main.worker", "main.main", "runtime.goexit"}, Count: 5},
		{Stack: []string{"main.worker", "main.main", "runtime.goexit"}, Count: 3},
	}
	tree := profiles.BuildCallTree(samples)
	// root should be "runtime.goexit" (bottom of stack) or "main.worker" (top)
	// We expect total weight to be 8
	total := profiles.TotalWeight(tree)
	if total != 8 {
		t.Errorf("want total weight=8, got %d", total)
	}
}

func TestCallTree_WeightPerFunction(t *testing.T) {
	samples := []profiles.CPUSample{
		{Stack: []string{"main.hotPath"}, Count: 10},
		{Stack: []string{"main.coldPath"}, Count: 2},
	}
	tree := profiles.BuildCallTree(samples)
	weights := profiles.FlatWeights(tree)
	if weights["main.hotPath"] != 10 {
		t.Errorf("want main.hotPath weight=10, got %d", weights["main.hotPath"])
	}
	if weights["main.coldPath"] != 2 {
		t.Errorf("want main.coldPath weight=2, got %d", weights["main.coldPath"])
	}
}

func TestCallTree_EmptySamples(t *testing.T) {
	tree := profiles.BuildCallTree(nil)
	if profiles.TotalWeight(tree) != 0 {
		t.Error("expected total weight=0 for nil samples")
	}
}

func TestCallTree_ChildrenAreSubcalls(t *testing.T) {
	samples := []profiles.CPUSample{
		{Stack: []string{"main.leaf", "main.parent", "main.root"}, Count: 1},
	}
	tree := profiles.BuildCallTree(samples)
	// tree should have a chain: root → parent → leaf
	if tree == nil || len(tree.Children) == 0 {
		t.Error("expected children in call tree")
	}
}
