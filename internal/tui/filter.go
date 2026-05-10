package tui

import (
	"regexp"
	"strings"

	"github.com/hrmeetsingh/gviz/internal/model"
)

// FilterGoroutinesRegex filters goroutines by compiling query as a regex
// matched against Label and State. If the regex is invalid, it falls back to
// case-insensitive literal substring matching. An empty query returns all.
func FilterGoroutinesRegex(goroutines []*model.Goroutine, query string) []*model.Goroutine {
	if query == "" {
		return goroutines
	}

	re, err := regexp.Compile("(?i)" + query)
	if err != nil {
		// Fallback: literal substring match.
		return FilterGoroutines(goroutines, query)
	}

	var result []*model.Goroutine
	for _, g := range goroutines {
		if re.MatchString(g.Label) || re.MatchString(string(g.State)) {
			result = append(result, g)
		}
	}
	return result
}

// FilterByState returns goroutines matching the given state exactly.
// An empty state returns all goroutines.
func FilterByState(goroutines []*model.Goroutine, state model.GoroutineState) []*model.Goroutine {
	if state == "" {
		return goroutines
	}
	var result []*model.Goroutine
	for _, g := range goroutines {
		if strings.EqualFold(string(g.State), string(state)) {
			result = append(result, g)
		}
	}
	return result
}
