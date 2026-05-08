package attach_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harmeetsingh/gviz/internal/attach"
)

const pprofGoroutineDump = `goroutine profile: total 2
1 @ 0x43d1c6 0x43d22b 0x46def1 0x47043c 0x4704b7 0x472f76 0x4cf31e 0x4cf3ab 0x4e4e89 0x42ec21
#	0x46def0	runtime/pprof.writeGoroutineStacks+0xb0	/usr/local/go/src/runtime/pprof/pprof.go:700
#	0x47043b	net/http/pprof.handler.ServeHTTP+0x39b	/usr/local/go/src/net/http/pprof/pprof.go:253
#	0x4704b6	net/http/pprof.Index+0x76			/usr/local/go/src/net/http/pprof/pprof.go:96

goroutine 1 [running]:
main.main()
	/home/user/proj/main.go:10 +0x68

goroutine 18 [chan receive]:
main.producer()
	/home/user/proj/main.go:20 +0x45
created by main.main in goroutine 1
	/home/user/proj/main.go:15 +0x25
`

func TestPprofFetcher_FetchesGoroutines(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/debug/pprof/goroutine" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, pprofGoroutineDump)
	}))
	defer srv.Close()

	fetcher := attach.NewPprofFetcher(srv.URL)
	goroutines, err := fetcher.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(goroutines) == 0 {
		t.Error("expected at least one goroutine")
	}
}

func TestPprofFetcher_ErrorOnBadURL(t *testing.T) {
	fetcher := attach.NewPprofFetcher("http://127.0.0.1:1") // nothing listening
	_, err := fetcher.Fetch()
	if err == nil {
		t.Error("expected error for unreachable URL, got nil")
	}
}

func TestPprofFetcher_ErrorOnNonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	fetcher := attach.NewPprofFetcher(srv.URL)
	_, err := fetcher.Fetch()
	if err == nil {
		t.Error("expected error for non-200 status, got nil")
	}
}

func TestAutoDetect_UsesPprofWhenAvailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, pprofGoroutineDump)
	}))
	defer srv.Close()

	fetcher, err := attach.AutoDetect(attach.AutoDetectConfig{
		PprofURL: srv.URL,
		PID:      0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fetcher == nil {
		t.Fatal("expected non-nil fetcher")
	}
	goroutines, err := fetcher.Fetch()
	if err != nil {
		t.Fatalf("fetch error: %v", err)
	}
	if len(goroutines) == 0 {
		t.Error("expected goroutines from pprof")
	}
}

func TestAutoDetect_ErrorWhenNothingAvailable(t *testing.T) {
	_, err := attach.AutoDetect(attach.AutoDetectConfig{
		PprofURL: "http://127.0.0.1:1",
		PID:      0,
	})
	if err == nil {
		t.Error("expected error when nothing is reachable")
	}
}
