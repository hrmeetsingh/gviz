# gviz — goroutine visualizer

`gviz` is a terminal UI that attaches to a running Go process and renders its
goroutines as an interactive, auto-refreshing tree — styled like `git log --graph`.

```
└── [1] main.main  •  select
    ├── [6] main.dispatcher  •  chan send
    │       send ···> goroutine 7
    ├── [7] main.worker  •  chan receive
    │       recv ···> goroutine 6
    ├── [8] main.producer  •  chan send
    │       send ···> goroutine 9
    ├── [9] main.consumer  •  chan receive
    │       recv ···> goroutine 8
    └── [10] main.sleeper  •  sleep (10m0s)
```

Each node is colored consistently: the branch line and the goroutine label share
the same color. Channel relationships are shown as dotted annotations between
goroutines. The view refreshes at a configurable interval and highlights newly
spawned and recently ended goroutines.

---

## Features

- pprof HTTP attach
- Delve PID + binary attach
- Parent/child tree from `created by` frames
- Channel inference (send / recv siblings)
- Snapshot diff (new / ended goroutines)
- Live filter by name or state
- Regex filter + state dropdown
- Configurable refresh interval
- Goroutine leak alerts
- Timeline ring-buffer
- Mutex contention view
- Per-goroutine allocs overlay
- SVG export
- CPU flamegraph call tree

---

## Requirements

- Go 1.23+
- Target process must expose a pprof endpoint **or** be attachable via Delve

---

## Install

```bash
go install github.com/hrmeetsingh/gviz/cmd/gviz@latest
```

Or build from source:

```bash
git clone https://github.com/hrmeetsingh/gviz
cd gviz
go build -o gviz ./cmd/gviz
```

---

## Quick start with the example

The `examples/simple` program is a ready-made target that spawns a small set of
goroutines covering every state gviz visualizes: channel send/recv pairs
(dispatcher/worker and producer/consumer), a sleeping goroutine, and a
select-based context worker. It exposes pprof on `:6060` automatically.

**Terminal 1 — start the target:**

```bash
make run-example
# or: go run ./examples/simple
```

**Terminal 2 — attach gviz:**

```bash
make attach
# or: make build && ./bin/gviz --url http://localhost:6060
```

You should see a tree like:

```
└── [1] main.main  •  select
    ├── [6] main.dispatcher  •  chan send
    │       send ···> goroutine 7
    ├── [7] main.worker  •  chan receive
    │       recv ···> goroutine 6
    ├── [8] main.producer  •  chan send
    │       send ···> goroutine 9
    ├── [9] main.consumer  •  chan receive
    │       recv ···> goroutine 8
    └── [10] main.sleeper  •  sleep (10m0s)
```

Other useful Makefile targets:

```bash
make attach-leak-check     # alert when goroutine count > 50 or grows 5 ticks
make export-svg            # write snapshot.svg to project root, then quit
make attach-pid PID=1234   # attach via Delve instead of pprof
make example-pid           # find the PID of a running example
```

---

## Usage

### pprof endpoint (recommended)

Add this to your target program (no-op in production if behind a flag):

```go
import _ "net/http/pprof"

go http.ListenAndServe(":6060", nil)
```

Then run:

```bash
gviz --url http://localhost:6060
```

### Delve attach

The target binary must be built without compiler optimizations so Delve can
read its debug info. For the bundled example, use the Makefile target:

```bash
make build-example-debug   # builds ./bin/example with -gcflags="all=-N -l"
./bin/example &
make attach-pid PID=$!
```

For your own binary, disable optimizations manually:

```bash
go build -gcflags="all=-N -l" -o myapp ./cmd/myapp
```

Then attach by PID:

```bash
gviz --pid 12345
```

Or let gviz launch the binary itself:

```bash
gviz --binary ./myapp
```

Or connect to an existing Delve headless server:

```bash
dlv attach 12345 --headless --listen=:4321 --api-version=2
gviz --dlv-addr 127.0.0.1:4321
```

### Options

| Flag | Default | Description |
|---|---|---|
| `--url` | — | pprof base URL |
| `--pid` | `0` | target process PID for Delve attach |
| `--binary` | — | binary path for Delve launch-and-attach |
| `--dlv-addr` | — | address of existing Delve headless server (e.g. `127.0.0.1:4321`) |
| `--interval` | `1s` | refresh interval (e.g. `500ms`, `2s`) |
| `--leak-threshold` | `0` | alert when goroutine count exceeds N (0 = off) |
| `--leak-window` | `5` | alert on N consecutive count increases |
| `--export` | — | write SVG snapshot to file on next refresh |

### Keybindings

| Key | Action |
|---|---|
| type | filter goroutines by name or state |
| `Backspace` | delete last filter character |
| `Esc` | clear filter / deselect |
| `?` | toggle help |
| `q` / `Ctrl+C` | quit |

---

## Run tests

```bash
make test          # go test ./...
make test-verbose  # go test -v ./...
make test-race     # go test -race ./...
```

---

## Architecture

```
gviz/
├── cmd/gviz/           CLI entrypoint — flags, attach, launch TUI
├── flows/              Architecture diagrams (overview.md)
├── internal/
│   ├── model/          Core types: Goroutine, GoroutineTree, Snapshot
│   ├── parser/         pprof dump text → []Goroutine (ID, state, stack, parentID, label)
│   ├── analyzer/       BuildTree, InferChannelPairs, Diff, BuildSnapshot
│   ├── attach/         Fetcher interface, PprofFetcher, DelveFetcher + RPC adapter, AutoDetect
│   ├── app/            CollectSnapshot coordinator (fetch → parse → snapshot)
│   ├── alerts/         LeakDetector — threshold + monotonic growth detection
│   ├── history/        RingBuffer[T] — capped snapshot history for timeline view
│   ├── profiles/       Mutex / allocs / CPU profile parsers (pprof text format)
│   ├── export/         SVGRenderer — snapshot → SVG document
│   └── tui/            Bubble Tea model, Lip Gloss renderer, filter, detail panel
└── CHANGELOG.md
```

### Key decisions

**No instrumentation required.** Parent/child relationships and channel
inferences are derived entirely from goroutine stack text. This means zero
changes to the target program when using the pprof adapter.

**`internal/app` as coordinator.** The fetch→snapshot pipeline lives in
`internal/app`, not inside the TUI. This keeps the Bubble Tea model purely
presentational and makes the pipeline independently testable.

**Deterministic ordering.** `BuildTree` sorts children by goroutine ID so the
tree layout is stable across refreshes, preventing visual jitter.

**Entry-function labels.** Goroutine labels are derived from the entry function
(the deepest non-runtime frame in the stack), not the top-of-stack frame. This
means a goroutine blocked in `runtime.chanrecv1` is labeled `main.worker`
rather than a cryptic runtime symbol.

**Fetcher interface.** `attach.Fetcher` is a narrow interface (`Fetch() ([]*model.Goroutine, error)`).
The pprof and Delve adapters both implement it; adding a file-replay adapter
for tests or a remote gRPC adapter requires only this one method.
