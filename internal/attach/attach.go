package attach

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hrmeetsingh/gviz/internal/model"
	"github.com/hrmeetsingh/gviz/internal/parser"
)

// --- pprof adapter ---

// PprofFetcher fetches goroutine data from a pprof HTTP endpoint.
type PprofFetcher struct {
	baseURL string
	client  *http.Client
}

// NewPprofFetcher creates a PprofFetcher targeting the given base URL.
// The goroutine endpoint will be baseURL + "/debug/pprof/goroutine?debug=2".
func NewPprofFetcher(baseURL string) *PprofFetcher {
	return &PprofFetcher{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{},
	}
}

// Fetch retrieves and parses the goroutine dump from the pprof endpoint.
func (f *PprofFetcher) Fetch() ([]*model.Goroutine, error) {
	url := f.baseURL + "/debug/pprof/goroutine?debug=2"
	resp, err := f.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("pprof fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pprof fetch: HTTP %d from %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("pprof read body: %w", err)
	}

	goroutines, err := parser.Parse(string(body))
	if err != nil {
		return nil, fmt.Errorf("pprof parse: %w", err)
	}
	return goroutines, nil
}

// --- auto-detect ---

// AutoDetectConfig holds parameters for auto-detection.
type AutoDetectConfig struct {
	PprofURL   string // pprof base URL (e.g. "http://localhost:6060")
	PID        int    // target process PID for Delve attach (0 = not provided)
	BinaryPath string // binary path for Delve launch-and-attach
	DelveAddr  string // address of existing Delve headless server
}

// AutoDetect returns the best available Fetcher for the given config.
// Priority: pprof URL → Delve addr → Delve PID → Delve binary.
// Returns an error if no adapter succeeds.
func AutoDetect(cfg AutoDetectConfig) (Fetcher, error) {
	if cfg.PprofURL != "" {
		f := NewPprofFetcher(cfg.PprofURL)
		if _, err := f.Fetch(); err == nil {
			return f, nil
		}
	}

	delveCfg := DelveServerConfig{
		PID:        cfg.PID,
		BinaryPath: cfg.BinaryPath,
		Addr:       cfg.DelveAddr,
	}

	if cfg.DelveAddr != "" || cfg.PID != 0 || cfg.BinaryPath != "" {
		client, err := StartDelveServer(delveCfg)
		if err != nil {
			return nil, fmt.Errorf("delve attach: %w", err)
		}
		return NewDelveFetcherFromClient(client), nil
	}

	return nil, fmt.Errorf("auto-detect: no reachable attach point (tried pprof=%q, pid=%d)", cfg.PprofURL, cfg.PID)
}
