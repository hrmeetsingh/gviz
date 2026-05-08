package profiles_test

import (
	"testing"
	"time"

	"github.com/harmeetsingh/gviz/internal/profiles"
)

const mutexProfileText = `--- mutex:
cycles/second=1000000000
sampling period=1
1 1 @ 0x1 0x2 0x3
#	0x1	sync.(*Mutex).Lock+0x71	/usr/local/go/src/sync/mutex.go:81
#	0x2	main.criticalSection+0x45	/proj/main.go:30
#	0x3	main.worker+0x23		/proj/main.go:20

2 2 @ 0x4 0x5
#	0x4	sync.(*RWMutex).RLock+0x4a	/usr/local/go/src/sync/rwmutex.go:58
#	0x5	main.reader+0x15		/proj/main.go:50
`

func TestParseMutexProfile_ParsesRecords(t *testing.T) {
	records, err := profiles.ParseMutexProfile(mutexProfileText)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) == 0 {
		t.Fatal("expected at least one mutex record")
	}
}

func TestParseMutexProfile_RecordHasContentionCount(t *testing.T) {
	records, err := profiles.ParseMutexProfile(mutexProfileText)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if records[0].Count < 1 {
		t.Errorf("want count >= 1, got %d", records[0].Count)
	}
}

func TestParseMutexProfile_RecordHasFunction(t *testing.T) {
	records, err := profiles.ParseMutexProfile(mutexProfileText)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if records[0].TopFunction == "" {
		t.Error("want non-empty TopFunction")
	}
}

func TestParseMutexProfile_Empty(t *testing.T) {
	records, err := profiles.ParseMutexProfile("")
	if err != nil {
		t.Fatalf("unexpected error on empty: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("want 0 records for empty input, got %d", len(records))
	}
}

func TestParseMutexProfile_WaitDurationPositive(t *testing.T) {
	records, err := profiles.ParseMutexProfile(mutexProfileText)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range records {
		if r.WaitDuration < 0 {
			t.Errorf("wait duration should not be negative, got %v", r.WaitDuration)
		}
		_ = r.WaitDuration.String() // ensure it's a valid Duration
		_ = time.Duration(0)
	}
}
