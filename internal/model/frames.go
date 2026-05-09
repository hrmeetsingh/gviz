package model

import "strings"

// RuntimePrefixes lists function name prefixes that belong to the Go runtime
// or standard library internals rather than user code.
var RuntimePrefixes = []string{
	"runtime.", "runtime/", "internal/", "sync.", "sync/",
	"net.", "net/", "time.", "syscall.", "os.", "io.",
}

// IsRuntimeFrame returns true if the function name belongs to the Go runtime
// or standard library internals rather than user code.
func IsRuntimeFrame(fn string) bool {
	for _, prefix := range RuntimePrefixes {
		if strings.HasPrefix(fn, prefix) {
			return true
		}
	}
	return false
}

// EntryLabel returns the goroutine's entry function — the deepest stack frame
// that isn't a runtime/stdlib internal. Falls back to the top frame if every
// frame is internal, or empty string if the stack is empty.
func EntryLabel(stack []Frame) string {
	if len(stack) == 0 {
		return ""
	}
	for i := len(stack) - 1; i >= 0; i-- {
		if !IsRuntimeFrame(stack[i].Function) {
			return stack[i].Function
		}
	}
	return stack[0].Function
}
