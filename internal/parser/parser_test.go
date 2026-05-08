package parser_test

import (
	"testing"

	"github.com/harmeetsingh/gviz/internal/model"
	"github.com/harmeetsingh/gviz/internal/parser"
)

const singleGoroutine = `goroutine 1 [running]:
main.main()
	/home/user/proj/main.go:42 +0x68
`

const multiGoroutine = `goroutine 1 [running]:
main.main()
	/home/user/proj/main.go:42 +0x68

goroutine 18 [chan receive]:
main.producer(0xc0000b4000)
	/home/user/proj/main.go:20 +0x45
created by main.main in goroutine 1
	/home/user/proj/main.go:15 +0x25

goroutine 19 [chan send]:
main.consumer(0xc0000b4000)
	/home/user/proj/main.go:30 +0x55
created by main.main in goroutine 1
	/home/user/proj/main.go:16 +0x30
`

const goroutineWithWaitReason = `goroutine 5 [sleep, 3 minutes]:
time.Sleep(0x2540be400)
	/usr/local/go/src/runtime/time.go:193 +0xd2
main.longSleeper()
	/home/user/proj/main.go:55 +0x27
created by main.main in goroutine 1
	/home/user/proj/main.go:50 +0x40
`

const goroutineWithSubroutines = `goroutine 1 [running]:
main.main()
	/home/user/proj/main.go:10 +0x68

goroutine 2 [chan receive]:
main.worker(0xc000014080)
	/home/user/proj/main.go:20 +0x45
created by main.main in goroutine 1
	/home/user/proj/main.go:15 +0x25

goroutine 3 [select]:
main.subWorker()
	/home/user/proj/main.go:35 +0x60
created by main.worker in goroutine 2
	/home/user/proj/main.go:22 +0x40
`

func TestParse_SingleGoroutine(t *testing.T) {
	goroutines, err := parser.Parse(singleGoroutine)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(goroutines) != 1 {
		t.Fatalf("want 1 goroutine, got %d", len(goroutines))
	}
	g := goroutines[0]
	if g.ID != 1 {
		t.Errorf("want ID=1, got %d", g.ID)
	}
	if g.State != model.StateRunning {
		t.Errorf("want state=running, got %q", g.State)
	}
	if g.ParentID != -1 {
		t.Errorf("want parentID=-1, got %d", g.ParentID)
	}
}

func TestParse_MultipleGoroutines(t *testing.T) {
	goroutines, err := parser.Parse(multiGoroutine)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(goroutines) != 3 {
		t.Fatalf("want 3 goroutines, got %d", len(goroutines))
	}
}

func TestParse_ParentIDFromCreatedBy(t *testing.T) {
	goroutines, err := parser.Parse(multiGoroutine)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	byID := make(map[int64]*model.Goroutine)
	for _, g := range goroutines {
		byID[g.ID] = g
	}

	if byID[18].ParentID != 1 {
		t.Errorf("goroutine 18: want parentID=1, got %d", byID[18].ParentID)
	}
	if byID[19].ParentID != 1 {
		t.Errorf("goroutine 19: want parentID=1, got %d", byID[19].ParentID)
	}
}

func TestParse_ChannelStateInferred(t *testing.T) {
	goroutines, err := parser.Parse(multiGoroutine)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	byID := make(map[int64]*model.Goroutine)
	for _, g := range goroutines {
		byID[g.ID] = g
	}

	if byID[18].State != model.StateChanRecv {
		t.Errorf("goroutine 18: want state=chan receive, got %q", byID[18].State)
	}
	if byID[19].State != model.StateChanSend {
		t.Errorf("goroutine 19: want state=chan send, got %q", byID[19].State)
	}
}

func TestParse_WaitReasonParsed(t *testing.T) {
	goroutines, err := parser.Parse(goroutineWithWaitReason)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(goroutines) != 1 {
		t.Fatalf("want 1 goroutine, got %d", len(goroutines))
	}
	g := goroutines[0]
	if g.ID != 5 {
		t.Errorf("want ID=5, got %d", g.ID)
	}
	if g.WaitReason != "3 minutes" {
		t.Errorf("want wait reason '3 minutes', got %q", g.WaitReason)
	}
}

func TestParse_StackFramesParsed(t *testing.T) {
	goroutines, err := parser.Parse(singleGoroutine)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	g := goroutines[0]
	if len(g.Stack) == 0 {
		t.Fatal("expected stack frames, got none")
	}
	if g.Stack[0].Function != "main.main" {
		t.Errorf("want function=main.main, got %q", g.Stack[0].Function)
	}
	if g.Stack[0].File != "/home/user/proj/main.go" {
		t.Errorf("want file=/home/user/proj/main.go, got %q", g.Stack[0].File)
	}
	if g.Stack[0].Line != 42 {
		t.Errorf("want line=42, got %d", g.Stack[0].Line)
	}
}

func TestParse_EmptyInput(t *testing.T) {
	goroutines, err := parser.Parse("")
	if err != nil {
		t.Fatalf("unexpected error on empty input: %v", err)
	}
	if len(goroutines) != 0 {
		t.Errorf("want 0 goroutines, got %d", len(goroutines))
	}
}

func TestParse_DeepNestedParent(t *testing.T) {
	goroutines, err := parser.Parse(goroutineWithSubroutines)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	byID := make(map[int64]*model.Goroutine)
	for _, g := range goroutines {
		byID[g.ID] = g
	}
	// goroutine 3 created by goroutine 2
	if byID[3].ParentID != 2 {
		t.Errorf("goroutine 3: want parentID=2, got %d", byID[3].ParentID)
	}
}
