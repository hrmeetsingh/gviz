package export

import (
	"fmt"
	"strings"

	"github.com/hrmeetsingh/gviz/internal/model"
)

const (
	nodeWidth  = 220
	nodeHeight = 40
	hGap       = 60  // horizontal gap between sibling subtrees
	vGap       = 80  // vertical distance between parent and child row
	nodePad    = 10  // text padding inside node box
	svgPadding = 20
)

// SVGRenderer converts a Snapshot into an SVG document.
type SVGRenderer struct{}

// NewSVGRenderer creates a new SVGRenderer.
func NewSVGRenderer() *SVGRenderer { return &SVGRenderer{} }

// layout holds computed positions for a goroutine node.
type layout struct {
	g    *model.Goroutine
	x, y int
	w    int // subtree width
}

// Render converts the snapshot tree into an SVG string.
func (r *SVGRenderer) Render(snap *model.Snapshot) string {
	if snap == nil {
		return `<svg xmlns="http://www.w3.org/2000/svg"></svg>`
	}

	nodes := layoutTree(snap.Roots, svgPadding, svgPadding)

	// Compute canvas size.
	maxX, maxY := 0, 0
	for _, n := range nodes {
		if n.x+nodeWidth > maxX {
			maxX = n.x + nodeWidth
		}
		if n.y+nodeHeight > maxY {
			maxY = n.y + nodeHeight
		}
	}
	width := maxX + svgPadding
	height := maxY + svgPadding

	var sb strings.Builder
	fmt.Fprintf(&sb, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d">`+"\n", width, height)
	sb.WriteString(`  <style>rect{fill:#282c34;stroke:#61afef;stroke-width:1.5} text{font-family:monospace;font-size:11px;fill:#abb2bf} line{stroke:#61afef;stroke-width:1}</style>`+"\n")

	// Draw edges first (under nodes).
	nodeByID := make(map[int64]*layout, len(nodes))
	for i := range nodes {
		nodeByID[nodes[i].g.ID] = &nodes[i]
	}
	for _, n := range nodes {
		if n.g.ParentID != -1 {
			if parent, ok := nodeByID[n.g.ParentID]; ok {
				px := parent.x + nodeWidth/2
				py := parent.y + nodeHeight
				cx := n.x + nodeWidth/2
				cy := n.y
				fmt.Fprintf(&sb, `  <line x1="%d" y1="%d" x2="%d" y2="%d"/>`+"\n", px, py, cx, cy)
			}
		}
	}

	// Draw nodes.
	for _, n := range nodes {
		label := n.g.Label
		if label == "" {
			label = fmt.Sprintf("goroutine %d", n.g.ID)
		}
		stateColor := stateColorHex(n.g.State)
		fmt.Fprintf(&sb, `  <rect x="%d" y="%d" width="%d" height="%d" rx="4"/>`+"\n",
			n.x, n.y, nodeWidth, nodeHeight)
		fmt.Fprintf(&sb, `  <text x="%d" y="%d">goroutine %d</text>`+"\n",
			n.x+nodePad, n.y+16, n.g.ID)
		fmt.Fprintf(&sb, `  <text x="%d" y="%d" style="fill:%s">%s  %s</text>`+"\n",
			n.x+nodePad, n.y+30, stateColor, escapeXML(label), string(n.g.State))
	}

	sb.WriteString("</svg>\n")
	return sb.String()
}

// layoutTree assigns x/y coordinates to all nodes via DFS.
func layoutTree(roots []*model.Goroutine, startX, startY int) []layout {
	var result []layout
	x := startX
	for _, root := range roots {
		lays := layoutNode(root, x, startY)
		result = append(result, lays...)
		if len(lays) > 0 {
			// Advance x by the subtree width of this root.
			x += lays[0].w + hGap
		}
	}
	return result
}

func layoutNode(g *model.Goroutine, x, y int) []layout {
	var result []layout
	childX := x
	childW := 0
	for _, child := range g.Children {
		childLays := layoutNode(child, childX, y+vGap)
		result = append(result, childLays...)
		if len(childLays) > 0 {
			childX += childLays[0].w + hGap
			childW += childLays[0].w + hGap
		}
	}
	if childW > 0 {
		childW -= hGap
	}
	selfW := childW
	if selfW < nodeWidth {
		selfW = nodeWidth
	}
	// Center the node over its children.
	selfX := x + (selfW-nodeWidth)/2
	result = append([]layout{{g: g, x: selfX, y: y, w: selfW}}, result...)
	return result
}

func stateColorHex(state model.GoroutineState) string {
	switch state {
	case model.StateRunning:
		return "#98c379"
	case model.StateChanRecv, model.StateChanSend:
		return "#61afef"
	case model.StateWaiting, model.StateSleep:
		return "#e5c07b"
	case model.StateSyscall:
		return "#c678dd"
	default:
		return "#abb2bf"
	}
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
