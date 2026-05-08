package profiles_test

import (
	"testing"

	"github.com/harmeetsingh/gviz/internal/profiles"
)

const allocsProfileText = `heap profile: 3: 4096 [10: 32768] @ heap/1048576
1: 1024 [3: 3072] @ 0x1 0x2 0x3
#	0x1	main.allocBig+0x55		/proj/main.go:15
#	0x2	main.worker+0x30		/proj/main.go:25
#	0x3	main.main+0x20			/proj/main.go:10

2: 3072 [7: 29696] @ 0x4 0x5
#	0x4	main.allocSmall+0x20		/proj/main.go:40
#	0x5	main.handler+0x10		/proj/main.go:35
`

func TestParseAllocsProfile_ParsesRecords(t *testing.T) {
	records, err := profiles.ParseAllocsProfile(allocsProfileText)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) == 0 {
		t.Fatal("expected at least one alloc record")
	}
}

func TestParseAllocsProfile_RecordHasInUseBytes(t *testing.T) {
	records, err := profiles.ParseAllocsProfile(allocsProfileText)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if records[0].InUseBytes <= 0 {
		t.Errorf("want InUseBytes > 0, got %d", records[0].InUseBytes)
	}
}

func TestParseAllocsProfile_RecordHasTopFunction(t *testing.T) {
	records, err := profiles.ParseAllocsProfile(allocsProfileText)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if records[0].TopFunction == "" {
		t.Error("want non-empty TopFunction")
	}
}

func TestParseAllocsProfile_Empty(t *testing.T) {
	records, err := profiles.ParseAllocsProfile("")
	if err != nil {
		t.Fatalf("unexpected error on empty: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("want 0 records, got %d", len(records))
	}
}

func TestParseAllocsProfile_AllocCountPositive(t *testing.T) {
	records, err := profiles.ParseAllocsProfile(allocsProfileText)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range records {
		if r.AllocCount < 0 {
			t.Errorf("alloc count should not be negative, got %d", r.AllocCount)
		}
	}
}
