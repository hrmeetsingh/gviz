// Package app coordinates the fetch→parse→snapshot pipeline, keeping the TUI
// layer purely presentational.
package app

import (
	"github.com/harmeetsingh/gviz/internal/analyzer"
	"github.com/harmeetsingh/gviz/internal/attach"
	"github.com/harmeetsingh/gviz/internal/model"
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
