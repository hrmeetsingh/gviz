package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/hrmeetsingh/gviz/internal/model"
)

// RenderOptions controls rendering behaviour.
type RenderOptions struct {
	Width        int
	ShowChannels bool
}

// palette cycles through distinct colors for goroutine tree branches.
var palette = []lipgloss.Color{
	"#61AFEF", // blue
	"#98C379", // green
	"#E5C07B", // yellow
	"#E06C75", // red
	"#C678DD", // purple
	"#56B6C2", // cyan
}

func colorFor(id int64) lipgloss.Color {
	return palette[int(id)%len(palette)]
}

// RenderTree renders the goroutine tree in a git-branch style.
func RenderTree(roots []*model.Goroutine, opts RenderOptions) string {
	if len(roots) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, root := range roots {
		renderNode(&sb, root, "", true, opts)
	}
	return sb.String()
}

func renderNode(sb *strings.Builder, g *model.Goroutine, prefix string, isLast bool, opts RenderOptions) {
	color := colorFor(g.ID)
	lineStyle := lipgloss.NewStyle().Foreground(color)
	textStyle := lipgloss.NewStyle().Foreground(color)

	// Branch connector
	connector := "├── "
	childPrefix := prefix + "│   "
	if isLast {
		connector = "└── "
		childPrefix = prefix + "    "
	}

	label := goroutineLabel(g)
	line := lineStyle.Render(prefix+connector) + textStyle.Render(label)
	sb.WriteString(line)
	sb.WriteByte('\n')

	if opts.ShowChannels && len(g.Channels) > 0 {
		for _, ch := range g.Channels {
			sb.WriteString(lineStyle.Render(renderChannelLine(childPrefix, ch)))
			sb.WriteByte('\n')
		}
	}

	for i, child := range g.Children {
		renderNode(sb, child, childPrefix, i == len(g.Children)-1, opts)
	}
}

// renderChannelLine formats a single channel annotation with a dotted line.
func renderChannelLine(prefix string, ch model.Channel) string {
	peer := fmt.Sprintf("goroutine %d", ch.PeerID)
	if ch.PeerID == -1 {
		peer = "unknown"
	}
	return fmt.Sprintf("%s    %s ···> %s", prefix, ch.Direction, peer)
}

func goroutineLabel(g *model.Goroutine) string {
	label := g.Label
	if label == "" && len(g.Stack) > 0 {
		label = g.Stack[0].Function
	}
	if label == "" {
		label = fmt.Sprintf("goroutine %d", g.ID)
	}

	state := string(g.State)
	if g.WaitReason != "" {
		state = fmt.Sprintf("%s (%s)", state, g.WaitReason)
	}
	return fmt.Sprintf("[%d] %s  •  %s", g.ID, label, state)
}

// FilterGoroutines returns goroutines matching query against label or state.
// An empty query returns all goroutines.
func FilterGoroutines(goroutines []*model.Goroutine, query string) []*model.Goroutine {
	if query == "" {
		return goroutines
	}
	q := strings.ToLower(query)
	var result []*model.Goroutine
	for _, g := range goroutines {
		if strings.Contains(strings.ToLower(g.Label), q) ||
			strings.Contains(strings.ToLower(string(g.State)), q) {
			result = append(result, g)
		}
	}
	return result
}
