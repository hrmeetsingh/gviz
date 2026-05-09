# CHANGELOG

## v33 - README (Delve) - 2026-05-08T19:55

### What changed
- Removed "(planned)" from Delve attach section
- Documented `--dlv-addr` flag (connect to existing headless server)
- Added `dlv attach` + `gviz --dlv-addr` example
- Updated Options table with `--dlv-addr`
- Updated Architecture section: attach package now includes RPC adapter

### Files touched
- `README.md`

### Test status
- 90 / 90 passing

---

## v32 - Refactor (Delve) - 2026-05-08T19:52

### What changed
- R1: Created `internal/model/frames.go` with shared `EntryLabel`, `IsRuntimeFrame`, `RuntimePrefixes`
- R2: Parser and Delve fetcher now both call `model.EntryLabel` — removed duplicate `entryLabel`, `delveEntryLabel`, `runtimePrefixes`, `stdlibPrefixes`, `isRuntimeFrame`, `isRuntimeFunc`
- R3: `delveRPCClient` now queries `GetVersion().TargetGoVersion` at connect time and caches `goVersion` — falls back to Go 1.25 if query fails

### Files touched
- `internal/model/frames.go` (new)
- `internal/parser/parser.go`
- `internal/attach/delve.go`
- `internal/attach/delve_rpc.go`

### Test status
- 90 / 90 passing

### Summary
The three refactors eliminate all code duplication in the entry-label and runtime-frame-detection logic. `model.EntryLabel` and `model.IsRuntimeFrame` are now the single source of truth used by both the pprof parser and the Delve fetcher. The Go version query replaces the hardcoded `1.25` with a runtime-detected version from the target binary, ensuring wait reason strings are always resolved using the correct table.

---

## v31 - Refactor Plan (Delve) - 2026-05-08T19:48

### What changed
- Refactor plan confirmed: (1) shared isRuntimeFrame + runtimePrefixes in model, (2) shared entryLabel in model, (3) query Go version from Delve at connect time

### Files touched
- `CHANGELOG.md`

### Test status
- 90 / 90 passing

### Summary
Three refactors: (1) move `isRuntimeFrame`/`runtimePrefixes` from parser.go and `isRuntimeFunc`/`stdlibPrefixes` from delve.go to `internal/model/frames.go` as shared exported functions, eliminating duplication. (2) Move `entryLabel` and `delveEntryLabel` to model as a single `EntryLabel(stack []Frame) string`. (3) Replace hardcoded Go 1.25 in `delve_rpc.go` with a `goVersion` field on `delveRPCClient` populated by querying the target via `rpc.GetVersion()` at connection time.

---

## v30 - Minimal Implementation (Delve) - 2026-05-08T19:45

### What changed
- `delve_rpc.go`: real `delveRPCClient` adapter wrapping `rpc2.RPCClient` — implements `ListGoroutines`, `Stacktrace`, `Close`
- `StartDelveServer`: launches headless Delve server (PID attach or binary exec) via `rpccommon.NewServer`, returns connected `DelveClient`; also supports connecting to an existing server via `--dlv-addr`
- `delveToModel` enhanced: maps `WaitReason` string to specific `GoroutineState` (chan send/recv, select, sleep, etc.), populates `ThreadID`, `ParentID` from `AncestorGID`
- `DelveFetcher.Fetch` now calls `Stacktrace` per goroutine, builds full `model.Frame` stack, derives entry-function label
- `AutoDetect` updated: tries pprof → Delve addr → Delve PID → Delve binary
- CLI: added `--binary` and `--dlv-addr` flags

### Files touched
- `internal/attach/delve.go`
- `internal/attach/delve_rpc.go` (new)
- `internal/attach/attach.go`
- `internal/model/goroutine.go`
- `cmd/gviz/main.go`
- `go.mod`, `go.sum`

### Test status
- 90 / 90 passing

### Summary
The Delve attach path is now fully functional. `delveRPCClient` wraps `rpc2.RPCClient` and translates the Delve API types (`api.Goroutine`, `api.Stackframe`) to gviz's internal `DelveGoroutine`/`DelveFrame` types, keeping the `DelveClient` interface testable via mocks. `StartDelveServer` handles all three modes: PID attach (spawns a headless Delve server bound to an ephemeral port), binary exec (same server but with `ProcessArgs`), and connection to an existing headless server. Wait reasons from Delve's numeric enum are translated via `api.WaitReasonString` using Go 1.25 wait reason table, then mapped to gviz `GoroutineState` constants (chan receive, chan send, select, sleep, etc.). The 2-second startup timeout in `StartDelveServer` catches immediate failures without blocking indefinitely.

---

## v29 - Tests First (Delve implementation) - 2026-05-08T19:38

### What changed
- Extended DelveClient interface: added Stacktrace method
- Extended DelveGoroutine: added WaitReason, ThreadID, AncestorGID fields
- Added DelveFrame type
- Added ThreadID to model.Goroutine
- 8 new failing tests: FullStackFromStacktrace, WaitReasonMapped, WaitReasonChanSend, WaitReasonSelect, WaitReasonSleep, LabelFromEntryFunction, ParentIDFromGoStatement, ThreadIDPopulated

### Files touched
- `internal/attach/delve.go`
- `internal/attach/delve_test.go`
- `internal/model/goroutine.go`

### Test status
- 82 passing / 90 total (8 new failing)

---

## v28 - Diagram (Delve implementation) - 2026-05-08T19:35

### What changed
- Architecture diagram confirmed: StartServer lifecycle, rpc2 adapter, full stacktrace, AutoDetect fallback chain

### Files touched
- `CHANGELOG.md`

### Test status
- 82 / 82 passing

---

## v27 - Clarify (Delve implementation) - 2026-05-08T19:33

### What changed
- Requirements gathered for real Delve attach implementation

### Files touched
- `CHANGELOG.md`

### Test status
- 82 / 82 passing

### Answers
- **attach_modes**: PID attach + binary launch + connect to existing headless server
- **delve_dep**: Go library import (github.com/go-delve/delve) — direct RPC calls, already in go.mod
- **goroutine_detail**: extended — populate ThreadID, full stacktrace, wait reason from Delve API
- **lifecycle**: gviz starts/stops the dlv headless server automatically

---

## v26 - README (label fix) - 2026-05-08T19:28

### What changed
- Added "Entry-function labels" to Key decisions section in README

### Files touched
- `README.md`

### Test status
- 82 / 82 passing

---

## v25 - Refactor (label fix) - 2026-05-08T19:26

### What changed
- Extracted `runtimePrefixes` package-level var from inline slice in `isRuntimeFrame`
- Extracted `entryLabel(stack []model.Frame) string` from inline logic in `parseBlock`

### Files touched
- `internal/parser/parser.go`

### Test status
- 82 / 82 passing

### Summary
Two clean extractions: `runtimePrefixes` is now a package-level var making it trivial to extend the list without touching function internals; `entryLabel` is a standalone function that encapsulates the bottom-up stack walk, keeping `parseBlock` focused on block parsing. Both refactors are behavior-preserving with all tests green.

---

## v24 - Refactor Plan (label fix) - 2026-05-08T19:24

### What changed
- Refactor plan confirmed: (1) extract runtimePrefixes to package var, (2) extract entryLabel(stack) as standalone function

### Files touched
- `CHANGELOG.md`

### Test status
- 82 / 82 passing

### Summary
Two refactors: move the inline prefix slice in `isRuntimeFrame` to a package-level `runtimePrefixes` var for extensibility, and extract the bottom-up label walk into `func entryLabel(stack []model.Frame) string` to keep `parseBlock` focused on parsing and make the label logic independently testable.

---

## v23 - Minimal Implementation (label fix + example rewrite) - 2026-05-08T19:22

### What changed
- Parser label: walks stack bottom-up, picks last non-runtime frame as Label (entry function)
- Added `isRuntimeFrame` helper matching `runtime.`, `internal/`, `time.`, `sync.`, `net.`, `syscall.`, `os.`, `io.` prefixes
- Example rewrite: unbuffered channels with no sleep between ops — goroutines reliably block on chan send/recv
- Worker/consumer sleep 2s after each receive — ensures dispatcher/producer stay blocked on send

### Files touched
- `internal/parser/parser.go`
- `examples/simple/main.go`

### Test status
- 82 / 82 passing

### Summary
The label bug was caused by `parseBlock` setting `Label = Stack[0].Function` — the top-of-stack frame, which is typically a runtime function like `time.Sleep` or `runtime.gopark`. The fix walks the stack bottom-up and picks the last frame whose function name doesn't match runtime/stdlib prefixes. This gives user-meaningful labels like `main.worker` instead of `runtime.chanrecv1`. The example was rewritten to use unbuffered channels with deliberate processing delays so that pprof snapshots reliably capture goroutines in `chan send` and `chan receive` states, making channel pair annotations visible.

---

## v22 - Tests First (label fix) - 2026-05-08T19:18

### What changed
- Added 3 label tests: TestParse_LabelIsEntryFunction, TestParse_LabelEntryFunctionMultiFrame, TestParse_LabelSkipsRuntimeFrames
- Added goroutineDeepRuntimeStack test fixture (deep runtime frames before user entry function)

### Files touched
- `internal/parser/parser_test.go`

### Test status
- 80 passing / 82 total (2 new failing)

---

## v21 - Diagram (label + example fix) - 2026-05-08T19:16

### What changed
- Architecture diagram confirmed: parser label from entry function, example with unbuffered blocking channels

### Files touched
- `CHANGELOG.md`

### Test status
- 79 / 79 passing

---

## v20 - Clarify (label + example + runtime fix) - 2026-05-08T19:15

### What changed
- Requirements gathered for three fixes: label extraction, example blocking, runtime goroutine visibility

### Files touched
- `CHANGELOG.md`

### Test status
- 79 / 79 passing

### Answers
- **label_strategy**: always use entry function (last user frame before `created by`), including goroutine 1 → `main.main`
- **runtime_filter**: show all goroutines, fix labels so they're identifiable; let user filter manually
- **example_blocking**: remove `time.Sleep` between channel ops; unbuffered channels naturally block sender until receiver ready

---

## v19 - Refactor (channel fix) - 2026-05-08T19:12

### What changed
- Reset Goroutine.Channels alongside Children in BuildTree loop

### Files touched
- `internal/analyzer/analyzer.go`

### Test status
- 79 / 79 passing

### Summary
Added `g.Channels = nil` to the existing reset loop in `BuildTree` so both relationship fields (`Children` and `Channels`) are cleared together before re-wiring. Prevents stale channel annotations when the same Goroutine pointers are passed to `BuildSnapshot` across refresh cycles.

---

## v18 - Refactor Plan (channel fix) - 2026-05-08T19:10

### What changed
- Refactor plan confirmed: reset Goroutine.Channels alongside Children in BuildTree

### Files touched
- `CHANGELOG.md`

### Test status
- 79 / 79 passing

### Summary
Single refactor: add `g.Channels = nil` reset in the loop inside `BuildTree` that already resets `g.Children`. Keeps both relationship fields reset together, preventing stale channel annotations on repeated calls with the same Goroutine pointers.

---

## v17 - Minimal Implementation (channel fix + example rewrite) - 2026-05-08T19:08

### What changed
- Fixed BuildSnapshot: now calls InferChannelPairs and wires results into Goroutine.Channels
- Rewrote examples/simple — 6 goroutines: dispatcher/worker (chan send/recv), producer/consumer (chan send/recv), sleeper (sleep), contextWorker (select)
- Updated both README sample tree outputs to match new example

### Files touched
- `internal/analyzer/analyzer.go`
- `examples/simple/main.go`
- `README.md`

### Test status
- 79 / 79 passing

### Summary
The root cause was that `BuildSnapshot` never called `InferChannelPairs`, so `Goroutine.Channels` was always empty and the renderer's dotted-line annotations never fired. The fix adds a call to `InferChannelPairs` inside `BuildSnapshot` and iterates the returned pairs, populating the sender's `Channels` slice with `{Direction: "send", PeerID: receiverID}` and the receiver's with `{Direction: "recv", PeerID: senderID}`. The example was simplified from 10+ goroutines to 6, each demonstrating a distinct state visible in gviz.

---

## v16 - Tests First (channel fix) - 2026-05-08T19:04

### What changed
- Added TestBuildSnapshot_WiresChannelPairs (failing — sender.Channels is empty)
- Added TestBuildSnapshot_NoChannelsForNonChanState (passing — sleep goroutine has no channels)

### Files touched
- `internal/analyzer/analyzer_test.go`

### Test status
- 78 passing / 79 total (1 new failing)

---

## v15 - Diagram (channel fix + example rewrite) - 2026-05-08T19:02

### What changed
- Architecture diagram confirmed: BuildSnapshot → InferChannelPairs → wire to Goroutine.Channels; 6-goroutine example

### Files touched
- `CHANGELOG.md`

### Test status
- 77 / 77 passing

---

## v14 - Clarify (channel fix + example rewrite) - 2026-05-08T19:00

### What changed
- Requirements gathered for channel inference pipeline fix + example rewrite

### Files touched
- `CHANGELOG.md`

### Test status
- 77 / 77 passing

### Answers
- **scope**: fix pipeline bug (BuildSnapshot never calls InferChannelPairs / results not wired to Goroutine.Channels) AND rewrite example
- **example_goroutines**: moderate — 1 worker + dispatcher, 1 producer + 1 consumer, 1 sleeper, 1 context worker (~6 goroutines)
- **readme_tree**: yes, update both sample trees in README

---

## v13 - Bug fixes - 2026-05-08T18:06

### What changed
- Fixed `make attach-leak-check` and `make export-svg`: `--leak-threshold`, `--leak-window`, and `--export` flags were referenced in the Makefile but never registered in the CLI
- Added `tui.Options` struct; `tui.New` now accepts it instead of positional args
- Wired `LeakDetector` into the TUI model: alert message appears in the status bar on breach
- Wired `--export`: on the first successful fetch the SVG is written to the given path, then the program quits
- README: removed all emojis, converted features table to a bullet list

### Files touched
- `cmd/gviz/main.go`
- `internal/tui/model.go`
- `README.md`

### Test status
- 77 / 77 passing

---

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
