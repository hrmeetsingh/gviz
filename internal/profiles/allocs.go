package profiles

import (
	"strconv"
	"strings"
)

// AllocRecord represents one entry from a pprof allocs/heap profile (debug=1).
type AllocRecord struct {
	// InUseBytes is the number of bytes currently in use for this stack.
	InUseBytes int64
	// InUseObjects is the number of live objects.
	InUseObjects int64
	// AllocBytes is cumulative bytes allocated.
	AllocBytes int64
	// AllocCount is cumulative number of allocations.
	AllocCount int64
	// TopFunction is the innermost function in the allocation stack.
	TopFunction string
}

// ParseAllocsProfile parses the text output of /debug/pprof/allocs?debug=1
// (same format as /debug/pprof/heap?debug=1).
//
// Each block looks like:
//
//	<inuse_objs>: <inuse_bytes> [<alloc_objs>: <alloc_bytes>] @ <heap/N>
//	#  <addr>  <function>+<offset>  <file>:<line>
func ParseAllocsProfile(text string) ([]AllocRecord, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}

	var records []AllocRecord
	lines := strings.Split(text, "\n")
	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])

		if line == "" || strings.HasPrefix(line, "heap profile:") {
			i++
			continue
		}

		// Data line format: "N: BYTES [M: BYTES2] @ heap/..."
		// e.g. "1: 1024 [3: 3072] @ 0x1 0x2"
		if strings.HasPrefix(line, "#") {
			i++
			continue
		}

		rec, ok := parseAllocsLine(line)
		if !ok {
			i++
			continue
		}

		// Scan frame lines for top function.
		i++
		for i < len(lines) {
			fl := strings.TrimSpace(lines[i])
			if !strings.HasPrefix(fl, "#") {
				break
			}
			if rec.TopFunction == "" {
				parts := strings.Fields(fl)
				if len(parts) >= 3 {
					fn := parts[2]
					if idx := strings.LastIndex(fn, "+"); idx != -1 {
						fn = fn[:idx]
					}
					rec.TopFunction = fn
				}
			}
			i++
		}
		records = append(records, rec)
	}
	return records, nil
}

// parseAllocsLine parses "inuse_objs: inuse_bytes [alloc_objs: alloc_bytes] @ ..."
func parseAllocsLine(line string) (AllocRecord, bool) {
	// Strip everything from "@" onward.
	if idx := strings.Index(line, "@"); idx != -1 {
		line = strings.TrimSpace(line[:idx])
	}
	// Strip bracket section "[M: B]"
	bracket := ""
	if s := strings.Index(line, "["); s != -1 {
		if e := strings.Index(line, "]"); e != -1 {
			bracket = line[s+1 : e]
			line = strings.TrimSpace(line[:s])
		}
	}

	// "N: BYTES"
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return AllocRecord{}, false
	}
	inUseObjs, err1 := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	inUseBytes, err2 := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	if err1 != nil || err2 != nil {
		return AllocRecord{}, false
	}
	rec := AllocRecord{InUseObjects: inUseObjs, InUseBytes: inUseBytes}

	// "[M: B]"
	if bracket != "" {
		bp := strings.SplitN(bracket, ":", 2)
		if len(bp) == 2 {
			if v, err := strconv.ParseInt(strings.TrimSpace(bp[0]), 10, 64); err == nil {
				rec.AllocCount = v
			}
			if v, err := strconv.ParseInt(strings.TrimSpace(bp[1]), 10, 64); err == nil {
				rec.AllocBytes = v
			}
		}
	}
	return rec, true
}
