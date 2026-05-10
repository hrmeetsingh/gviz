package alerts_test

import (
	"testing"

	"github.com/hrmeetsingh/gviz/internal/alerts"
)

func TestLeakDetector_NoAlertBelowThreshold(t *testing.T) {
	d := alerts.NewLeakDetector(alerts.Config{Threshold: 100, GrowthWindow: 5})
	a := d.Observe(50)
	if a != nil {
		t.Errorf("want no alert below threshold, got %+v", a)
	}
}

func TestLeakDetector_AlertWhenThresholdExceeded(t *testing.T) {
	d := alerts.NewLeakDetector(alerts.Config{Threshold: 10, GrowthWindow: 5})
	a := d.Observe(11)
	if a == nil {
		t.Fatal("want alert when count > threshold, got nil")
	}
	if a.Level != alerts.LevelWarn {
		t.Errorf("want level=warn, got %q", a.Level)
	}
}

func TestLeakDetector_AlertOnMonotonicGrowth(t *testing.T) {
	d := alerts.NewLeakDetector(alerts.Config{Threshold: 1000, GrowthWindow: 3})
	d.Observe(10)
	d.Observe(11)
	a := d.Observe(12) // 3 consecutive increases
	if a == nil {
		t.Fatal("want alert on monotonic growth over window, got nil")
	}
	if a.Level != alerts.LevelWarn {
		t.Errorf("want level=warn, got %q", a.Level)
	}
}

func TestLeakDetector_NoAlertOnStableCount(t *testing.T) {
	d := alerts.NewLeakDetector(alerts.Config{Threshold: 1000, GrowthWindow: 3})
	d.Observe(10)
	d.Observe(10)
	a := d.Observe(10)
	if a != nil {
		t.Errorf("want no alert on stable count, got %+v", a)
	}
}

func TestLeakDetector_ClearsAfterDrop(t *testing.T) {
	d := alerts.NewLeakDetector(alerts.Config{Threshold: 1000, GrowthWindow: 3})
	d.Observe(10)
	d.Observe(11)
	d.Observe(12) // growth alert fires
	a := d.Observe(5) // count drops — no alert
	if a != nil {
		t.Errorf("want no alert after count drops, got %+v", a)
	}
}

func TestLeakDetector_AlertMessageContainsCount(t *testing.T) {
	d := alerts.NewLeakDetector(alerts.Config{Threshold: 5, GrowthWindow: 3})
	a := d.Observe(20)
	if a == nil {
		t.Fatal("want alert")
	}
	if a.Count != 20 {
		t.Errorf("want Count=20 in alert, got %d", a.Count)
	}
}
