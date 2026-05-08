package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/harmeetsingh/gviz/internal/model"
)

// Parse parses a goroutine dump (as produced by runtime/pprof or
// /debug/pprof/goroutine?debug=2) into a slice of Goroutine structs.
func Parse(dump string) ([]*model.Goroutine, error) {
	dump = strings.TrimSpace(dump)
	if dump == "" {
		return nil, nil
	}

	// Split on blank lines that start a new "goroutine N [...]:" header.
	blocks := splitGoroutineBlocks(dump)
	var result []*model.Goroutine
	for _, block := range blocks {
		g, err := parseBlock(block)
		if err != nil {
			continue // skip malformed blocks
		}
		result = append(result, g)
	}
	return result, nil
}

// splitGoroutineBlocks splits the full dump text into per-goroutine blocks.
func splitGoroutineBlocks(dump string) []string {
	var blocks []string
	var current strings.Builder

	for _, line := range strings.Split(dump, "\n") {
		if strings.HasPrefix(line, "goroutine ") && current.Len() > 0 {
			blocks = append(blocks, strings.TrimSpace(current.String()))
			current.Reset()
		}
		current.WriteString(line)
		current.WriteByte('\n')
	}
	if current.Len() > 0 {
		if b := strings.TrimSpace(current.String()); b != "" {
			blocks = append(blocks, b)
		}
	}
	return blocks
}

// parseBlock parses a single goroutine block.
func parseBlock(block string) (*model.Goroutine, error) {
	lines := strings.Split(block, "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty block")
	}

	g := &model.Goroutine{ParentID: -1}

	// Parse header: "goroutine N [state]:" or "goroutine N [state, wait reason]:"
	header := lines[0]
	if err := parseHeader(header, g); err != nil {
		return nil, err
	}

	// Parse stack frames and "created by" line.
	i := 1
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			i++
			continue
		}

		if strings.HasPrefix(line, "created by ") {
			parseCreatedBy(line, g)
			break
		}

		// Function name line (no leading tab in trimmed form, doesn't start with a path char).
		if !strings.HasPrefix(line, "/") && !strings.HasPrefix(line, "\t") {
			frame := model.Frame{Function: stripOffset(line)}
			// Next line should be the file:line info.
			if i+1 < len(lines) {
				fileLine := strings.TrimSpace(lines[i+1])
				if strings.HasPrefix(fileLine, "/") || strings.HasPrefix(fileLine, "\t/") {
					parseFileLine(fileLine, &frame)
					i++ // consumed file line
				}
			}
			g.Stack = append(g.Stack, frame)
		}
		i++
	}

	// Populate Label from top of stack so callers always have a human name.
	if len(g.Stack) > 0 {
		g.Label = g.Stack[0].Function
	}

	return g, nil
}

// parseHeader fills ID, State, WaitReason from the goroutine header line.
func parseHeader(line string, g *model.Goroutine) error {
	// "goroutine 18 [chan receive]:"
	// "goroutine 5 [sleep, 3 minutes]:"
	line = strings.TrimSuffix(line, ":")
	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 3 {
		return fmt.Errorf("malformed header: %q", line)
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return fmt.Errorf("bad goroutine ID in %q: %w", line, err)
	}
	g.ID = id

	stateStr := strings.TrimPrefix(parts[2], "[")
	stateStr = strings.TrimSuffix(stateStr, "]")

	if idx := strings.Index(stateStr, ","); idx != -1 {
		g.State = model.GoroutineState(strings.TrimSpace(stateStr[:idx]))
		g.WaitReason = strings.TrimSpace(stateStr[idx+1:])
	} else {
		g.State = model.GoroutineState(stateStr)
	}
	return nil
}

// parseCreatedBy extracts the parent goroutine ID from a "created by" line.
// Handles Go 1.21+ format: "created by pkg.Func in goroutine N"
// and older format:         "created by pkg.Func"  (parentID stays -1)
func parseCreatedBy(line string, g *model.Goroutine) {
	if i := strings.LastIndex(line, " in goroutine "); i != -1 {
		idStr := strings.TrimSpace(line[i+len(" in goroutine "):])
		if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
			g.ParentID = id
		}
	}
}

// parseFileLine extracts file path and line number from a stack file line.
// Format: "\t/path/to/file.go:42 +0x68"
func parseFileLine(line string, frame *model.Frame) {
	line = strings.TrimSpace(line)
	// strip offset suffix " +0x..."
	if i := strings.LastIndex(line, " +0x"); i != -1 {
		line = line[:i]
	}
	// "file.go:N"
	if i := strings.LastIndex(line, ":"); i != -1 {
		frame.File = line[:i]
		if n, err := strconv.Atoi(line[i+1:]); err == nil {
			frame.Line = n
		}
	}
}

// stripOffset removes the trailing " +0x..." offset and argument list from a
// function name line, e.g. "main.main()" → "main.main".
func stripOffset(s string) string {
	if i := strings.LastIndex(s, " +0x"); i != -1 {
		s = s[:i]
	}
	s = strings.TrimSpace(s)
	// Strip function arguments: "pkg.Func(...)" → "pkg.Func"
	if i := strings.Index(s, "("); i != -1 {
		s = s[:i]
	}
	return s
}
