package alerts

import "fmt"

// Level represents alert severity.
type Level string

const (
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

// Alert is a single leak detection event.
type Alert struct {
	Level   Level
	Message string
	Count   int
}

// Config controls the leak detector's behaviour.
type Config struct {
	// Threshold: fires LevelWarn when goroutine count exceeds this value.
	Threshold int
	// GrowthWindow: fires LevelWarn when count increases for this many
	// consecutive observations.
	GrowthWindow int
}

// LeakDetector tracks goroutine counts over time and emits alerts.
type LeakDetector struct {
	cfg     Config
	history []int
}

// NewLeakDetector creates a LeakDetector with the given config.
func NewLeakDetector(cfg Config) *LeakDetector {
	return &LeakDetector{cfg: cfg}
}

// Observe records a new goroutine count observation and returns an Alert if a
// leak condition is met, or nil otherwise.
func (d *LeakDetector) Observe(count int) *Alert {
	d.history = append(d.history, count)

	if d.cfg.Threshold > 0 && count > d.cfg.Threshold {
		return &Alert{
			Level:   LevelWarn,
			Message: fmt.Sprintf("goroutine count %d exceeds threshold %d", count, d.cfg.Threshold),
			Count:   count,
		}
	}

	if d.cfg.GrowthWindow > 1 && len(d.history) >= d.cfg.GrowthWindow {
		window := d.history[len(d.history)-d.cfg.GrowthWindow:]
		if isMonotonicallyIncreasing(window) {
			return &Alert{
				Level:   LevelWarn,
				Message: fmt.Sprintf("goroutine count grew for %d consecutive ticks (now %d)", d.cfg.GrowthWindow, count),
				Count:   count,
			}
		}
	}

	return nil
}

func isMonotonicallyIncreasing(xs []int) bool {
	for i := 1; i < len(xs); i++ {
		if xs[i] <= xs[i-1] {
			return false
		}
	}
	return true
}
