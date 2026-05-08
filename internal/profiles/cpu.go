package profiles

// CPUSample is a single sample from a CPU profile: a call stack and a hit count.
type CPUSample struct {
	Stack []string // ordered top-of-stack first (leaf → root)
	Count int64
}

// CallNode is a node in the call tree.
type CallNode struct {
	Function string
	Weight   int64
	Children []*CallNode
}

// BuildCallTree aggregates CPUSamples into a call tree.
// Samples are processed bottom-up (root first), so the tree root represents
// the entry point (bottom of stack).
func BuildCallTree(samples []CPUSample) *CallNode {
	if len(samples) == 0 {
		return &CallNode{}
	}

	// Use a synthetic root to hold multiple entry points.
	root := &CallNode{Function: "(root)"}

	for _, s := range samples {
		if len(s.Stack) == 0 {
			continue
		}
		// Walk stack bottom-up: last element is the outermost frame.
		node := root
		for j := len(s.Stack) - 1; j >= 0; j-- {
			fn := s.Stack[j]
			child := findOrCreateChild(node, fn)
			child.Weight += s.Count
			node = child
		}
	}
	return root
}

func findOrCreateChild(parent *CallNode, fn string) *CallNode {
	for _, c := range parent.Children {
		if c.Function == fn {
			return c
		}
	}
	c := &CallNode{Function: fn}
	parent.Children = append(parent.Children, c)
	return c
}

// TotalWeight returns the sum of weights of direct children of the root
// (representing total CPU samples).
func TotalWeight(root *CallNode) int64 {
	if root == nil {
		return 0
	}
	var total int64
	for _, c := range root.Children {
		total += c.Weight
	}
	return total
}

// FlatWeights returns a map of function name → weight by walking the entire tree.
func FlatWeights(root *CallNode) map[string]int64 {
	out := make(map[string]int64)
	if root == nil {
		return out
	}
	var walk func(*CallNode)
	walk = func(n *CallNode) {
		if n.Function != "(root)" && n.Weight > 0 {
			out[n.Function] += n.Weight
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
	return out
}
