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

// Label should be the goroutine's entry function (last user frame before
// "created by"), not the top-of-stack runtime function.
func TestParse_LabelIsEntryFunction(t *testing.T) {
	goroutines, err := parser.Parse(goroutineWithWaitReason)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	g := goroutines[0]
	// goroutineWithWaitReason stack: time.Sleep (top) → main.longSleeper (entry)
	// Label should be "main.longSleeper", not "time.Sleep"
	if g.Label != "main.longSleeper" {
		t.Errorf("want label=main.longSleeper, got %q", g.Label)
	}
}

func TestParse_LabelEntryFunctionMultiFrame(t *testing.T) {
	goroutines, err := parser.Parse(multiGoroutine)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	byID := make(map[int64]*model.Goroutine)
	for _, g := range goroutines {
		byID[g.ID] = g
	}
	// goroutine 18 has single user frame: main.producer → label should be main.producer
	if byID[18].Label != "main.producer" {
		t.Errorf("goroutine 18: want label=main.producer, got %q", byID[18].Label)
	}
	// goroutine 1 has no "created by" — single frame main.main → label should be main.main
	if byID[1].Label != "main.main" {
		t.Errorf("goroutine 1: want label=main.main, got %q", byID[1].Label)
	}
}

const goroutineDeepRuntimeStack = `goroutine 42 [chan receive]:
runtime.gopark(0x0?, 0x0?, 0x0?, 0x0?, 0x0?)
	/usr/local/go/src/runtime/proc.go:402 +0xce
runtime.chanrecv(0xc0000b2000, 0xc0000a3f58, 0x1)
	/usr/local/go/src/runtime/chan.go:583 +0x3c3
runtime.chanrecv1(0xc0000b2000, 0xc0000a3f58)
	/usr/local/go/src/runtime/chan.go:442 +0x18
main.worker(0xc0000b2000)
	/home/user/proj/main.go:20 +0x45
created by main.main in goroutine 1
	/home/user/proj/main.go:15 +0x25
`

func TestParse_LabelSkipsRuntimeFrames(t *testing.T) {
	goroutines, err := parser.Parse(goroutineDeepRuntimeStack)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	g := goroutines[0]
	// Stack has runtime.gopark, runtime.chanrecv, runtime.chanrecv1, main.worker
	// Label should be "main.worker" — the entry function, not any runtime frame
	if g.Label != "main.worker" {
		t.Errorf("want label=main.worker, got %q", g.Label)
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
