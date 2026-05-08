package profiles

import (
	"strconv"
	"strings"
	"time"
)

// MutexRecord represents one entry from a pprof mutex profile.
type MutexRecord struct {
	// Count is the number of times the mutex was contended in this stack.
	Count int64
	// Cycles is the total number of delay cycles (divide by cycles/second for duration).
	Cycles int64
	// WaitDuration is the estimated total wait time for this record.
	WaitDuration time.Duration
	// TopFunction is the innermost function in the contention stack.
	TopFunction string
}

// ParseMutexProfile parses the text output of /debug/pprof/mutex?debug=1.
// Format (per block):
//
//	<count> <cycles> @ <hex addrs...>
//	#  <addr>  <function>+<offset>  <file>:<line>
//	...
func ParseMutexProfile(text string) ([]MutexRecord, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}

	var records []MutexRecord
	var cyclesPerSec int64 = 1_000_000_000 // default; overridden by header

	lines := strings.Split(text, "\n")
	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])

		// Parse "cycles/second=N" header.
		if strings.HasPrefix(line, "cycles/second=") {
			if v, err := strconv.ParseInt(strings.TrimPrefix(line, "cycles/second="), 10, 64); err == nil {
				cyclesPerSec = v
			}
			i++
			continue
		}

		// Skip non-data lines.
		if line == "" || strings.HasPrefix(line, "---") || strings.HasPrefix(line, "sampling") {
			i++
			continue
		}

		// Data line: "<count> <cycles> @ ..."
		fields := strings.Fields(line)
		if len(fields) < 2 || strings.HasPrefix(line, "#") {
			i++
			continue
		}
		count, err1 := strconv.ParseInt(fields[0], 10, 64)
		cycles, err2 := strconv.ParseInt(fields[1], 10, 64)
		if err1 != nil || err2 != nil {
			i++
			continue
		}

		rec := MutexRecord{
			Count:  count,
			Cycles: cycles,
		}
		if cyclesPerSec > 0 {
			rec.WaitDuration = time.Duration(cycles * int64(time.Second) / cyclesPerSec)
		}

		// Scan frame lines for the top function.
		i++
		for i < len(lines) {
			fl := strings.TrimSpace(lines[i])
			if !strings.HasPrefix(fl, "#") {
				break
			}
			if rec.TopFunction == "" {
				parts := strings.Fields(fl)
				// "#  0xADDR  pkg.Func+offset  file:line"
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
