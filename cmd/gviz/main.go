package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/harmeetsingh/gviz/internal/attach"
	"github.com/harmeetsingh/gviz/internal/tui"
)

func main() {
	var (
		pprofURL = flag.String("url", "", "pprof base URL (e.g. http://localhost:6060)")
		pid      = flag.Int("pid", 0, "target process PID for Delve attach")
		interval = flag.Duration("interval", time.Second, "refresh interval")
	)
	flag.Parse()

	if *pprofURL == "" && *pid == 0 {
		fmt.Fprintln(os.Stderr, "gviz: provide --url or --pid")
		flag.Usage()
		os.Exit(1)
	}

	fetcher, err := attach.AutoDetect(attach.AutoDetectConfig{
		PprofURL: *pprofURL,
		PID:      *pid,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "gviz: attach failed: %v\n", err)
		os.Exit(1)
	}

	m := tui.New(fetcher, *interval)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "gviz: %v\n", err)
		os.Exit(1)
	}
}
