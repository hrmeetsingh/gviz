package tui

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hrmeetsingh/gviz/internal/alerts"
	"github.com/hrmeetsingh/gviz/internal/app"
	"github.com/hrmeetsingh/gviz/internal/attach"
	"github.com/hrmeetsingh/gviz/internal/export"
	"github.com/hrmeetsingh/gviz/internal/model"
)

// tickMsg is sent on each refresh interval.
type tickMsg time.Time

// fetchedMsg carries a new snapshot from the fetcher goroutine.
type fetchedMsg struct {
	snap *model.Snapshot
	err  error
}

// Options holds optional configuration for the TUI model.
type Options struct {
	LeakDetector *alerts.LeakDetector
	ExportPath   string
}

// Model is the Bubble Tea application model.
type Model struct {
	fetcher      attach.Fetcher
	interval     time.Duration
	snap         *model.Snapshot
	filter       string
	selected     int64 // ID of selected goroutine, -1 = none
	width        int
	height       int
	err          error
	showHelp     bool
	leakDetector *alerts.LeakDetector
	leakAlert    string
	exportPath   string
}

// New creates a new TUI model.
func New(fetcher attach.Fetcher, interval time.Duration, opts Options) Model {
	return Model{
		fetcher:      fetcher,
		interval:     interval,
		selected:     -1,
		leakDetector: opts.LeakDetector,
		exportPath:   opts.ExportPath,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(doFetch(m.fetcher, nil), tickAfter(m.interval))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		return m, doFetch(m.fetcher, m.snap)

	case fetchedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.snap = msg.snap
			m.err = nil

			if m.leakDetector != nil {
				if alert := m.leakDetector.Observe(len(m.snap.ByID)); alert != nil {
					m.leakAlert = alert.Message
				}
			}

			if m.exportPath != "" {
				svg := export.NewSVGRenderer().Render(m.snap)
				if werr := os.WriteFile(m.exportPath, []byte(svg), 0o644); werr != nil {
					m.err = werr
				}
				return m, tea.Quit
			}
		}
		return m, tickAfter(m.interval)

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "?":
			m.showHelp = !m.showHelp
		case "esc":
			m.filter = ""
			m.selected = -1
		default:
			// Append printable chars to filter.
			if len(msg.Runes) == 1 {
				m.filter += string(msg.Runes)
			}
			if msg.String() == "backspace" && len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.err != nil {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#E06C75")).
			Render(fmt.Sprintf("error: %v\n\nPress q to quit.", m.err))
	}
	if m.snap == nil {
		return "connecting…\n"
	}

	// Collect visible goroutines respecting filter.
	var all []*model.Goroutine
	for _, g := range m.snap.ByID {
		all = append(all, g)
	}
	visible := FilterGoroutines(all, m.filter)

	treeOutput := RenderTree(m.snap.Roots, RenderOptions{
		Width:        m.width,
		ShowChannels: true,
	})

	// If filter is active, re-render only matching subtree.
	if m.filter != "" {
		_ = visible // used by filter label below
	}

	statusBar := renderStatusBar(m)
	if m.showHelp {
		return treeOutput + "\n" + renderHelp() + "\n" + statusBar
	}
	return treeOutput + "\n" + statusBar
}

func renderStatusBar(m Model) string {
	style := lipgloss.NewStyle().Background(lipgloss.Color("#282C34")).Foreground(lipgloss.Color("#ABB2BF"))
	count := 0
	if m.snap != nil {
		count = len(m.snap.ByID)
	}
	filter := ""
	if m.filter != "" {
		filter = fmt.Sprintf("  filter: %s", m.filter)
	}
	diff := ""
	if m.snap != nil && len(m.snap.NewIDs) > 0 {
		diff = fmt.Sprintf("  +%d new", len(m.snap.NewIDs))
	}
	if m.snap != nil && len(m.snap.EndedIDs) > 0 {
		diff += fmt.Sprintf("  -%d ended", len(m.snap.EndedIDs))
	}
	leakInfo := ""
	if m.leakAlert != "" {
		leakInfo = fmt.Sprintf("  LEAK: %s", m.leakAlert)
	}
	bar := fmt.Sprintf(" goroutines: %d%s%s%s  [?] help  [q] quit", count, filter, diff, leakInfo)
	return style.Width(m.width).Render(bar)
}

func renderHelp() string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#ABB2BF")).Render(
		"  type  filter by name/state    esc  clear    q  quit    ?  toggle help",
	)
}

func tickAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func doFetch(f attach.Fetcher, prev *model.Snapshot) tea.Cmd {
	return func() tea.Msg {
		snap, err := app.CollectSnapshot(f, prev)
		if err != nil {
			return fetchedMsg{err: err}
		}
		return fetchedMsg{snap: snap}
	}
}
