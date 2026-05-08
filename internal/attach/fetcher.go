package attach

import "github.com/harmeetsingh/gviz/internal/model"

// Fetcher is the common interface for all attach adapters.
// Implementations: PprofFetcher, (future) DelveFetcher.
type Fetcher interface {
	Fetch() ([]*model.Goroutine, error)
}
