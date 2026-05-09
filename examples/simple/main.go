// Package main is a demo target for gviz. It spawns a small set of goroutines
// that exercise every state gviz can visualize: channel send/recv pairs
// (blocking on unbuffered channels), a sleeping goroutine, and a select-based
// context worker. Run it and attach gviz in another terminal:
//
//	gviz --url http://localhost:6060
package main

import (
	"context"
	"log"
	"net/http"
	_ "net/http/pprof"
	"time"
)

const pprofAddr = ":6060"

func main() {
	go func() {
		log.Printf("pprof listening on http://localhost%s", pprofAddr)
		if err := http.ListenAndServe(pprofAddr, nil); err != nil {
			log.Fatalf("pprof server: %v", err)
		}
	}()

	// Unbuffered channels — sender blocks until receiver is ready and vice versa,
	// so pprof snapshots reliably capture chan send / chan receive states.
	jobs := make(chan int)
	go dispatcher(jobs)
	go worker(jobs)

	pipe := make(chan []byte)
	go producer(pipe)
	go consumer(pipe)

	go sleeper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	contextWorker(ctx)
}

// dispatcher blocks on send until worker is ready to receive.
func dispatcher(jobs chan<- int) {
	for id := 1; ; id++ {
		jobs <- id
	}
}

// worker receives a job, processes it (sleep simulates work), then blocks on
// the next receive. The sleep ensures the dispatcher stays blocked on send
// between bursts.
func worker(jobs <-chan int) {
	for range jobs {
		time.Sleep(2 * time.Second)
	}
}

// producer blocks on send until consumer is ready.
func producer(pipe chan<- []byte) {
	for {
		pipe <- make([]byte, 512)
	}
}

// consumer receives a payload, processes it, then blocks on the next receive.
// The sleep ensures the producer stays blocked on send between bursts.
func consumer(pipe <-chan []byte) {
	for range pipe {
		time.Sleep(2 * time.Second)
	}
}

// sleeper stays blocked — shows as "sleep" state in gviz.
func sleeper() {
	time.Sleep(10 * time.Minute)
}

// contextWorker blocks in a select until context expires.
func contextWorker(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			log.Printf("example running — attach gviz: gviz --url http://localhost%s", pprofAddr)
		}
	}
}
