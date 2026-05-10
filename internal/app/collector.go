// Package app coordinates the fetch→parse→snapshot pipeline, keeping the TUI
// layer purely presentational.
package app

import (
	"github.com/hrmeetsingh/gviz/internal/analyzer"
	"github.com/hrmeetsingh/gviz/internal/attach"
	"github.com/hrmeetsingh/gviz/internal/model"
)

// CollectSnapshot fetches goroutines via the given Fetcher, builds a snapshot
// diffed against prev (nil for first call), and returns it.
func CollectSnapshot(f attach.Fetcher, prev *model.Snapshot) (*model.Snapshot, error) {
	goroutines, err := f.Fetch()
	if err != nil {
		return nil, err
	}
	return analyzer.BuildSnapshot(goroutines, prev), nil
}
