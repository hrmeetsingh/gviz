// Package main is a demo target for gviz. It deliberately creates a variety of
// goroutine patterns — fan-out worker pools, nested sub-goroutines, producer/
// consumer channel pairs, context-cancelled routines, and a long-sleeper — so
// that the full gviz tree, channel inference, and alert features can be
// exercised against a live process.
//
// Run it, then in another terminal:
//
//	gviz --url http://localhost:6060
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof" // registers /debug/pprof handlers
	"sync"
	"time"
)

const pprofAddr = ":6060"

func main() {
	// Start pprof endpoint.
	go func() {
		log.Printf("pprof listening on http://localhost%s", pprofAddr)
		if err := http.ListenAndServe(pprofAddr, nil); err != nil {
			log.Fatalf("pprof server: %v", err)
		}
	}()

	jobs := make(chan int, 20)
	results := make(chan string, 20)

	var wg sync.WaitGroup

	// Fan-out: 4 workers, each spawning a sub-worker.
	for i := 1; i <= 4; i++ {
		wg.Add(1)
		go worker(i, jobs, results, &wg)
	}

	// Dispatcher: sends job IDs to workers.
	go dispatcher(jobs)

	// Aggregator: reads results and prints them.
	go aggregator(results)

	// Producer/consumer pair sharing a dedicated channel.
	pipe := make(chan []byte, 5)
	go producer(pipe)
	go consumer(pipe)

	// Long-sleeper: demonstrates a goroutine blocked in sleep.
	go longSleeper()

	// Context-bound worker: exits after 30 s, then is re-spawned.
	go func() {
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			contextWorker(ctx)
			cancel()
			time.Sleep(500 * time.Millisecond)
		}
	}()

	// Metrics reporter goroutine.
	go metricsReporter()

	// Keep main alive.
	wg.Wait()
}

// worker processes jobs and spawns a sub-worker for each.
func worker(id int, jobs <-chan int, results chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {
		done := make(chan struct{})
		go subWorker(id, job, done)
		<-done
		results <- fmt.Sprintf("worker-%d finished job %d", id, job)
	}
}

// subWorker simulates processing inside a worker.
func subWorker(workerID, job int, done chan<- struct{}) {
	delay := time.Duration(50+rand.Intn(150)) * time.Millisecond
	time.Sleep(delay)
	close(done)
}

// dispatcher sends a continuous stream of job IDs to workers.
func dispatcher(jobs chan<- int) {
	for id := 1; ; id++ {
		jobs <- id
		time.Sleep(200 * time.Millisecond)
	}
}

// aggregator drains the results channel.
func aggregator(results <-chan string) {
	for range results {
		// intentionally silent — just drains the channel
	}
}

// producer writes payloads to a channel at irregular intervals.
func producer(pipe chan<- []byte) {
	for {
		size := 256 + rand.Intn(1024)
		payload := make([]byte, size)
		pipe <- payload
		time.Sleep(time.Duration(100+rand.Intn(300)) * time.Millisecond)
	}
}

// consumer reads payloads and simulates processing.
func consumer(pipe <-chan []byte) {
	for payload := range pipe {
		time.Sleep(time.Duration(len(payload)/100) * time.Millisecond)
	}
}

// longSleeper stays blocked in time.Sleep — shows as "sleep" state in gviz.
func longSleeper() {
	for {
		time.Sleep(10 * time.Minute)
	}
}

// contextWorker runs until its context is cancelled.
func contextWorker(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// simulate light work
		}
	}
}

// metricsReporter periodically prints a summary of live goroutines to stdout.
func metricsReporter() {
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()
	for range tick.C {
		log.Printf("example still running — attach gviz: gviz --url http://localhost%s", pprofAddr)
	}
}
