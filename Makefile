BINARY      := ./bin/gviz
EXAMPLE_PKG := ./examples/simple
PPROF_PORT  ?= 6060
PPROF_URL   := http://localhost:$(PPROF_PORT)
INTERVAL    ?= 1s

.PHONY: build test test-verbose test-race lint clean \
        run-example attach attach-pid \
        export-svg help

## ── Build ────────────────────────────────────────────────────────────────────

# Build the gviz binary into ./bin/
build:
	@mkdir -p bin
	go build -o $(BINARY) ./cmd/gviz
	@echo "Built $(BINARY)"

## ── Test ─────────────────────────────────────────────────────────────────────

# Run all tests
test:
	go test ./...

# Run all tests with verbose output
test-verbose:
	go test -v ./...

# Run all tests with race detector
test-race:
	go test -race ./...

## ── Example ──────────────────────────────────────────────────────────────────

# Run the demo target program (exposes pprof on :6060)
# Leave this running in one terminal, then use `make attach` in another.
run-example:
	@echo "Starting example on $(PPROF_URL)/debug/pprof ..."
	go run $(EXAMPLE_PKG)

# Attach gviz to the running example (requires `make run-example` in another terminal)
# Builds gviz first if the binary is missing.
attach: $(BINARY)
	$(BINARY) --url $(PPROF_URL) --interval $(INTERVAL)

# Attach by PID instead of pprof URL.
# Usage: make attach-pid PID=<pid>
attach-pid: $(BINARY)
ifndef PID
	$(error PID is required: make attach-pid PID=<pid>)
endif
	$(BINARY) --pid $(PID) --interval $(INTERVAL)

# Attach with leak detection enabled (threshold=50, growth window=5 ticks)
attach-leak-check: $(BINARY)
	$(BINARY) --url $(PPROF_URL) --interval $(INTERVAL) \
	          --leak-threshold 50 --leak-window 5

# Attach and export a single SVG snapshot to ./snapshot.svg, then quit.
export-svg: $(BINARY)
	$(BINARY) --url $(PPROF_URL) --interval $(INTERVAL) \
	          --export ./snapshot.svg

## ── Utilities ────────────────────────────────────────────────────────────────

# Find the PID of a running example process (macOS/Linux)
example-pid:
	@pgrep -f "examples/simple" && echo "(above is the example PID)" || echo "example is not running"

# Remove build artifacts
clean:
	rm -rf bin snapshot.svg

## ── Implicit rules ───────────────────────────────────────────────────────────

# Build the binary automatically when a target depends on it.
$(BINARY): build

## ── Help ─────────────────────────────────────────────────────────────────────

help:
	@echo ""
	@echo "gviz Makefile targets"
	@echo ""
	@echo "  Build"
	@echo "    make build              build ./bin/gviz"
	@echo "    make clean              remove ./bin and snapshot.svg"
	@echo ""
	@echo "  Test"
	@echo "    make test               run all tests"
	@echo "    make test-verbose       run all tests with -v"
	@echo "    make test-race          run all tests with -race"
	@echo ""
	@echo "  Example workflow"
	@echo "    make run-example        start the demo target (pprof on :6060)"
	@echo "    make attach             attach gviz via pprof (default port 6060)"
	@echo "    make attach-pid PID=N   attach gviz via Delve to process N"
	@echo "    make attach-leak-check  attach with leak detection enabled"
	@echo "    make export-svg         capture one SVG snapshot to ./snapshot.svg"
	@echo "    make example-pid        print PID of the running example"
	@echo ""
	@echo "  Variables (override with make <target> VAR=value)"
	@echo "    PPROF_PORT  default: 6060"
	@echo "    INTERVAL    default: 1s"
	@echo ""
