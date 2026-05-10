package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hrmeetsingh/gviz/internal/alerts"
	"github.com/hrmeetsingh/gviz/internal/attach"
	"github.com/hrmeetsingh/gviz/internal/tui"
)

func main() {
	var (
		pprofURL       = flag.String("url", "", "pprof base URL (e.g. http://localhost:6060)")
		pid            = flag.Int("pid", 0, "target process PID for Delve attach")
		binaryPath     = flag.String("binary", "", "binary path for Delve launch-and-attach")
		dlvAddr        = flag.String("dlv-addr", "", "address of existing Delve headless server (e.g. 127.0.0.1:4321)")
		interval       = flag.Duration("interval", time.Second, "refresh interval")
		leakThreshold  = flag.Int("leak-threshold", 0, "alert when goroutine count exceeds N (0 = off)")
		leakWindow     = flag.Int("leak-window", 5, "alert on N consecutive count increases")
		exportPath     = flag.String("export", "", "write SVG snapshot to file on next refresh, then quit")
	)
	flag.Parse()

	if *pprofURL == "" && *pid == 0 && *binaryPath == "" && *dlvAddr == "" {
		fmt.Fprintln(os.Stderr, "gviz: provide --url, --pid, --binary, or --dlv-addr")
		flag.Usage()
		os.Exit(1)
	}

	fetcher, err := attach.AutoDetect(attach.AutoDetectConfig{
		PprofURL:   *pprofURL,
		PID:        *pid,
		BinaryPath: *binaryPath,
		DelveAddr:  *dlvAddr,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "gviz: attach failed: %v\n", err)
		os.Exit(1)
	}

	var detector *alerts.LeakDetector
	if *leakThreshold > 0 || *leakWindow > 0 {
		detector = alerts.NewLeakDetector(alerts.Config{
			Threshold:    *leakThreshold,
			GrowthWindow: *leakWindow,
		})
	}

	m := tui.New(fetcher, *interval, tui.Options{
		LeakDetector: detector,
		ExportPath:   *exportPath,
	})
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "gviz: %v\n", err)
		os.Exit(1)
	}
}
