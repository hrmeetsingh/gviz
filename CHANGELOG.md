# CHANGELOG

## v12 - README (milestones) - 2026-05-08T08:48

### What changed
- Updated README.md: all milestones marked ✅, new flags documented, package layout expanded

### Files touched
- `README.md`

### Test status
- 77 / 77 passing

---

## v11 - Minimal Implementation (milestones) - 2026-05-08T08:47

### What changed
- Implemented all 8 milestone packages

### Files touched
- `internal/attach/delve.go` — M1: DelveFetcher, DelveGoroutine, DelveLoc, AutoDetectWithDelve
- `internal/alerts/alerts.go` — M2: LeakDetector, Alert, Config
- `internal/tui/filter.go` — M3: FilterGoroutinesRegex, FilterByState
- `internal/history/ringbuffer.go` — M4: RingBuffer[T] generic ring buffer
- `internal/profiles/mutex.go` — M5: ParseMutexProfile, MutexRecord
- `internal/profiles/allocs.go` — M6: ParseAllocsProfile, AllocRecord
- `internal/export/svg.go` — M7: SVGRenderer, layout engine
- `internal/profiles/cpu.go` — M8: BuildCallTree, CPUSample, CallNode, TotalWeight, FlatWeights
- `flows/overview.md` — all 10 architecture diagrams

### Test status
- 77 / 77 passing

### Summary
M1 introduces a `DelveClient` interface so `DelveFetcher` is fully testable without a live process; `AutoDetectWithDelve` accepts an injected client, making the fallback path unit-testable. M2 `LeakDetector` tracks a rolling count history and fires on threshold breach or monotonic growth over a configurable window. M3 adds `FilterGoroutinesRegex` (compiles query as regex, falls back to literal on invalid pattern) and `FilterByState` for exact state matching. M4 is a generic `RingBuffer[T]` with O(1) push and a clean `All()`/`Latest()` API. M5 and M6 parse pprof mutex and heap/allocs text profiles into typed records with top-function attribution. M7 `SVGRenderer` lays out the goroutine tree with a recursive subtree-width algorithm, draws edges before nodes, and uses Lipgloss-consistent colors in the SVG. M8 `BuildCallTree` aggregates `CPUSample` slices bottom-up into a `CallNode` tree; `FlatWeights` and `TotalWeight` provide summary views suitable for flamegraph and per-node CPU badges.

---

## v10 - Tests First (milestones) - 2026-05-08T08:46

### What changed
- Wrote failing tests for all 8 milestones

### Files touched
- `internal/attach/delve_test.go` — 6 tests
- `internal/alerts/alerts_test.go` — 6 tests
- `internal/tui/filter_test.go` — 6 tests
- `internal/history/ringbuffer_test.go` — 7 tests
- `internal/profiles/mutex_test.go` — 5 tests
- `internal/profiles/allocs_test.go` — 5 tests
- `internal/export/svg_test.go` — 6 tests
- `internal/profiles/cpu_test.go` — 5 tests

### Test status
- 31 passing (existing) / 46 new failing (build errors — implementations absent)

---

## v9 - Diagram (milestones) - 2026-05-08T08:45

### What changed
- Full milestone architecture diagram confirmed
- flows/overview.md created with all diagrams

### Files touched
- `CHANGELOG.md`
- `flows/overview.md`

### Test status
- 31 / 31 passing

---

## v8 - Clarify (milestones) - 2026-05-08T08:43

### What changed
- Requirements gathered for all 8 milestones + flows folder

### Files touched
- `CHANGELOG.md`

### Test status
- 31 / 31 passing

### Answers
- **milestones**: all 8 — Delve, leak alerts, filter UI, timeline, mutex, allocs, SVG export, flamegraph
- **flows_format**: single `flows/overview.md` with all diagrams
- **delve_scope**: PID + binary path (full launch support)

---

## v7 - README - 2026-05-08T08:41

### What changed
- Wrote README.md

### Files touched
- `README.md`

### Test status
- 31 / 31 passing

---

## v6 - Refactor - 2026-05-08T08:40

### What changed
- R1: removed dead `lines`/`idx` params from `parseCreatedBy`
- R2: `BuildTree` now sorts children by goroutine ID for stable rendering
- R3: extracted `renderChannelLine` helper in `tui/renderer.go`
- R4: moved `Fetcher` interface to `internal/attach/fetcher.go`
- R5: parser auto-populates `Goroutine.Label` from top stack frame
- R6: extracted `app.CollectSnapshot` coordinator; TUI model is now purely presentational

### Files touched
- `internal/parser/parser.go`
- `internal/analyzer/analyzer.go`
- `internal/attach/attach.go`
- `internal/attach/fetcher.go` ← new
- `internal/tui/renderer.go`
- `internal/tui/model.go`
- `internal/app/collector.go` ← new

### Test status
- 31 / 31 passing

### Summary
All six refactors applied without breaking any tests. The key structural win is R6: `internal/app` now owns the fetch→parse→snapshot pipeline, so `internal/tui` depends only on the `model.Snapshot` type and `attach.Fetcher` interface rather than on `analyzer` internals. R4 makes the `Fetcher` interface a first-class citizen ready for the Delve adapter. R2 ensures the rendered tree is deterministic across refreshes.

---

## v5 - Refactor Plan - 2026-05-08T08:39

### What changed
- Refactor plan confirmed

### Files touched
- `CHANGELOG.md`

### Test status
- 31 / 31 passing

### Summary
Six refactors approved: (1) clean dead params from parseCreatedBy, (2) sort children by ID in BuildTree, (3) extract renderChannelLine in renderer, (4) move Fetcher interface to attach/fetcher.go, (5) auto-populate Label in parser, (6) extract fetch→snapshot pipeline into internal/app coordinator package.

---

## v4 - Minimal Implementation - 2026-05-08T08:36

### What changed
- Implemented all four packages to pass tests
- Wired up CLI entrypoint and Bubble Tea TUI model

### Files touched
- `internal/parser/parser.go`
- `internal/analyzer/analyzer.go`
- `internal/attach/attach.go`
- `internal/tui/renderer.go`
- `internal/tui/model.go`
- `cmd/gviz/main.go`

### Test status
- 31 / 31 passing

### Summary
Parser splits pprof goroutine dump text into `Goroutine` structs, extracting ID, state, wait reason, stack frames, and parent ID via `created by` frame matching. Analyzer builds a tree from those structs, infers channel pairs by grouping sibling send/recv goroutines, diffs consecutive snapshots for new/ended IDs, and assembles a `Snapshot`. Attach package provides a `PprofFetcher` (HTTP client) and `AutoDetect` that tries pprof then reports failure if unreachable (Delve hook point left for next milestone). TUI is a Bubble Tea program: a tick loop re-fetches on `--interval`, renders goroutine tree in git-branch style with Lip Gloss colors, shows dotted channel annotations, a live filter bar, and a status bar with goroutine count and new/ended deltas.

---

## v3 - Tests First - 2026-05-08T08:35

### What changed
- Wrote failing tests for all four packages

### Files touched
- `internal/parser/parser_test.go` — 8 tests
- `internal/analyzer/analyzer_test.go` — 10 tests
- `internal/attach/attach_test.go` — 5 tests
- `internal/tui/renderer_test.go` — 8 tests

### Test status
- 0 passing / 31 total (build failures — implementations absent)

---

## v2 - Diagram - 2026-05-08T08:34

### What changed
- Architecture diagram confirmed by user

### Files touched
- `CHANGELOG.md`

### Test status
- no tests yet

---

## v1 - Clarify - 2026-05-08T08:30

### What changed
- Requirements gathered

### Files touched
- `CHANGELOG.md`

### Test status
- no tests yet

### Answers
- **attach_mechanism**: all three — pprof HTTP, Delve (PID), auto-detect (pprof → delve fallback)
- **channel_tracking**: infer from stack traces (best-effort, no instrumentation)
- **parent_child**: heuristic — parse `created by` stack frames to build tree
- **tui_lib**: Bubble Tea + Lip Gloss
- **go_version**: Go 1.23+
- **refresh**: configurable via `--interval` flag, default 1s
- **additional_ideas**: flamegraph overlay, filter/search, timeline/history, SVG export, leak alerts, mutex contention, per-goroutine allocs, interactive detail panel
