# gviz — Architecture Diagrams

All architecture diagrams for the project, in chronological order.

---

## Diagram 1 — Initial Architecture (v2, 2026-05-08)

Core system: pprof/Delve attach, stack parser, analyzer, Bubble Tea TUI.

```mermaid
graph TB
    cli["CLI cmd/gviz\nflags: --pid, --url, --interval"]

    subgraph attachLayer["Attach Layer"]
        autoDetect["auto-detect\npprof → delve fallback"]
        pprofAdapter["pprof adapter\nGET /debug/pprof/goroutine?debug=2"]
        delveAdapter["delve adapter\nattach by PID via go-delve/delve"]
    end

    subgraph analyzer["Analyzer"]
        stackParser["stack parser\ntext → Goroutine structs"]
        parentHeuristic["parent heuristic\n'created by' frame matching"]
        channelInferrer["channel inferrer\nchan send/recv in stacks"]
        stateTracker["state tracker\nsnapshot diff → new/ended/changed"]
    end

    subgraph model["Data Model"]
        goroutineStruct["Goroutine\n{id, label, state, parentID,\nctxTimeout, channels, stack}"]
        goroutineTree["GoroutineTree\nnested children"]
        snapshot["Snapshot\ncurrent + prev diff"]
    end

    subgraph tui["TUI  Bubble Tea + Lip Gloss"]
        treeView["tree view\ngit-branch style lines\nmatching color per routine + text"]
        chanLines["channel dotted-lines\norigin ←···→ destination"]
        filterBar["filter bar\nname / package / state / regex"]
        detailPanel["detail panel\nselected goroutine full stack"]
        statusBar["status bar\ncount · refresh · alerts"]
        ticker["tick loop\nrefreshes at --interval"]
    end

    subgraph milestones["Additional Features milestones"]
        flamegraph["flamegraph overlay\ncpu profile per goroutine"]
        timeline["history timeline\nscrollable goroutine lifecycle"]
        leakAlert["leak detection\ngoroutine count threshold alert"]
        mutexViz["mutex contention\npprof mutex profile"]
        memAllocs["per-goroutine allocs\npprof allocs profile"]
        svgExport["SVG/PNG export\nsnapshot to file"]
    end

    cli --> autoDetect
    autoDetect --> pprofAdapter
    autoDetect --> delveAdapter
    pprofAdapter --> stackParser
    delveAdapter --> stackParser
    stackParser --> parentHeuristic
    stackParser --> channelInferrer
    parentHeuristic --> goroutineTree
    channelInferrer --> goroutineStruct
    goroutineTree --> stateTracker
    stateTracker --> snapshot
    snapshot --> treeView
    snapshot --> chanLines
    snapshot --> detailPanel
    snapshot --> statusBar
    filterBar --> treeView
    ticker --> autoDetect
    treeView --> tui
    chanLines --> tui
    filterBar --> tui
    detailPanel --> tui
    statusBar --> tui
    snapshot --> milestones
```

---

## Diagram 2 — Full System with All Milestones (v9, 2026-05-08)

Extends Diagram 1 with all 8 milestones wired in.

```mermaid
graph TB
    cli["CLI cmd/gviz\n--url --pid --binary --interval\n--leak-threshold --export"]

    subgraph attachLayer["Attach Layer"]
        autoDetect["auto-detect\npprof → delve fallback"]
        pprofFetcher["PprofFetcher\nGET /debug/pprof/goroutine?debug=2"]
        delveFetcher["DelveFetcher  M1\nRPC attach by PID or binary+args"]
    end

    subgraph profileLayer["Profile Fetchers  pprof only"]
        mutexFetcher["MutexFetcher  M5\nGET /debug/pprof/mutex"]
        allocFetcher["AllocFetcher  M6\nGET /debug/pprof/allocs"]
        cpuFetcher["CPUFetcher  M8\nGET /debug/pprof/profile"]
    end

    subgraph parserLayer["Parser"]
        goroutineParser["goroutine parser\ndump text → Goroutine structs"]
        mutexParser["mutex parser  M5\n→ MutexRecord list"]
        allocParser["alloc parser  M6\n→ AllocRecord list"]
        cpuParser["cpu parser  M8\n→ CallTree weighted by samples"]
    end

    subgraph analyzerLayer["Analyzer"]
        buildTree["BuildTree\nparent-child from created-by"]
        channelInfer["InferChannelPairs\nsend/recv siblings"]
        diff["Diff\nnew / ended IDs"]
        buildSnap["BuildSnapshot"]
    end

    subgraph appLayer["App coordinator"]
        collector["CollectSnapshot\nfetch → parse → snapshot"]
        leakDetector["LeakDetector  M2\ncount threshold + growth window"]
        ringBuffer["RingBuffer  M4\ncapped snapshot history"]
    end

    subgraph exportLayer["Export  M7"]
        svgRenderer["SVGRenderer\nsnapshot → SVG file"]
    end

    subgraph tuiLayer["TUI  Bubble Tea + Lip Gloss"]
        treeView["tree view\ngit-branch style, color-matched"]
        channelLines["dotted channel lines"]
        filterBar["filter bar  M3\ntextinput + regex + state dropdown"]
        detailPanel["detail panel\nfull stack on select"]
        timelineView["timeline view  M4\nscrollable history + sparkline"]
        mutexOverlay["mutex overlay  M5\nwait duration on nodes"]
        allocOverlay["alloc overlay  M6\nalloc bytes on frames"]
        flamegraphView["flamegraph view  M8\nCPU weight overlay"]
        alertBadge["alert badge  M2\nleak warning in status bar"]
        statusBar["status bar\ncount · interval · alerts · mode"]
    end

    cli --> autoDetect
    autoDetect --> pprofFetcher
    autoDetect --> delveFetcher
    pprofFetcher --> goroutineParser
    delveFetcher --> goroutineParser
    pprofFetcher --> mutexFetcher
    pprofFetcher --> allocFetcher
    pprofFetcher --> cpuFetcher
    mutexFetcher --> mutexParser
    allocFetcher --> allocParser
    cpuFetcher --> cpuParser
    goroutineParser --> buildTree
    buildTree --> channelInfer
    channelInfer --> diff
    diff --> buildSnap
    buildSnap --> collector
    collector --> leakDetector
    collector --> ringBuffer
    leakDetector --> alertBadge
    ringBuffer --> timelineView
    collector --> treeView
    collector --> channelLines
    mutexParser --> mutexOverlay
    allocParser --> allocOverlay
    cpuParser --> flamegraphView
    treeView --> tuiLayer
    channelLines --> tuiLayer
    filterBar --> tuiLayer
    detailPanel --> tuiLayer
    timelineView --> tuiLayer
    mutexOverlay --> tuiLayer
    allocOverlay --> tuiLayer
    flamegraphView --> tuiLayer
    alertBadge --> tuiLayer
    statusBar --> tuiLayer
    collector --> svgRenderer
```

---

## Diagram 3 — M1: Delve Attach

```mermaid
graph TB
    cli["CLI\n--pid 1234\n--binary ./myapp --args flag1"]
    autoDetect["AutoDetect\ntry pprof first"]
    pprofFetcher["PprofFetcher\n(fails or not provided)"]
    delveFetcher["DelveFetcher"]
    delveClientIface["DelveClient interface\nListGoroutines()"]
    rpcClient["rpc2.RPCClient\ngo-delve/delve"]
    launchConfig["LaunchConfig\n{PID | Binary+Args}"]
    goroutineAdapter["goroutineAdapter\napi.Goroutine → model.Goroutine"]
    model["model.Goroutine"]

    cli --> autoDetect
    autoDetect --> pprofFetcher
    pprofFetcher -->|"fails"| delveFetcher
    cli --> launchConfig
    launchConfig --> delveFetcher
    delveFetcher --> delveClientIface
    delveClientIface --> rpcClient
    rpcClient -->|"api.Goroutine list"| goroutineAdapter
    goroutineAdapter --> model
```

---

## Diagram 4 — M2: Leak Detection

```mermaid
graph TB
    collector["app.CollectSnapshot\n(each tick)"]
    snapshot["model.Snapshot\n{ByID map, At time}"]
    leakDetector["LeakDetector\n{threshold, window, history []int}"]
    thresholdCheck["threshold check\nlen > threshold → alert"]
    growthCheck["growth check\nN consecutive increases → alert"]
    alert["Alert\n{Level, Message, Count}"]
    tui["TUI alert badge\nstatus bar highlight"]

    collector --> snapshot
    snapshot --> leakDetector
    leakDetector --> thresholdCheck
    leakDetector --> growthCheck
    thresholdCheck --> alert
    growthCheck --> alert
    alert --> tui
```

---

## Diagram 5 — M3: Filter UI

```mermaid
graph TB
    keyInput["keyboard input"]
    filterModel["FilterModel\n{query string, regex *Regexp, stateFilter}"]
    regexCompile["regex compile\nerror → fallback to literal"]
    stateDropdown["state dropdown\n(running/waiting/chan/select/...)"]
    filterFn["FilterGoroutines\nquery + state applied recursively"]
    roots["model.Goroutine roots"]
    filtered["filtered subtree"]
    renderer["RenderTree\n(filtered roots)"]

    keyInput --> filterModel
    filterModel --> regexCompile
    filterModel --> stateDropdown
    filterModel --> filterFn
    roots --> filterFn
    regexCompile --> filterFn
    stateDropdown --> filterFn
    filterFn --> filtered
    filtered --> renderer
```

---

## Diagram 6 — M4: Timeline / History

```mermaid
graph TB
    collector["app.CollectSnapshot"]
    ringBuffer["RingBuffer cap=N\nPush snapshot each tick"]
    timelineView["timeline view\ntoggle with 't' key"]
    sparkline["sparkline renderer\ngoroutine count per snapshot"]
    scrollCursor["scroll cursor\nj/k to navigate"]
    snapshotDetail["selected snapshot\nrenders full tree at that point-in-time"]

    collector --> ringBuffer
    ringBuffer --> timelineView
    timelineView --> sparkline
    timelineView --> scrollCursor
    scrollCursor --> snapshotDetail
    snapshotDetail --> snapshotDetail
```

---

## Diagram 7 — M5: Mutex Contention

```mermaid
graph TB
    pprofFetcher["PprofFetcher"]
    mutexEndpoint["GET /debug/pprof/mutex?debug=1"]
    mutexParser["mutex profile parser\ntext → MutexRecord list"]
    mutexRecord["MutexRecord\n{GoroutineID, Cycles, WaitDuration}"]
    contention["contention map\ngoroutineID → total wait"]
    treeRenderer["RenderTree\nannotate nodes with wait badge"]
    mutexOverlay["mutex overlay\n⏱ 12ms beside goroutine label"]

    pprofFetcher --> mutexEndpoint
    mutexEndpoint --> mutexParser
    mutexParser --> mutexRecord
    mutexRecord --> contention
    contention --> treeRenderer
    treeRenderer --> mutexOverlay
```

---

## Diagram 8 — M6: Per-Goroutine Allocs

```mermaid
graph TB
    pprofFetcher["PprofFetcher"]
    allocEndpoint["GET /debug/pprof/allocs?debug=1"]
    allocParser["allocs profile parser\ntext → AllocRecord list"]
    allocRecord["AllocRecord\n{Function, InUseBytes, AllocBytes, Count}"]
    allocMap["alloc map\nfunction → bytes"]
    treeRenderer["RenderTree\nannotate top frame with alloc badge"]
    allocOverlay["alloc overlay\n📦 4.2 MB beside goroutine label"]

    pprofFetcher --> allocEndpoint
    allocEndpoint --> allocParser
    allocParser --> allocRecord
    allocRecord --> allocMap
    allocMap --> treeRenderer
    treeRenderer --> allocOverlay
```

---

## Diagram 9 — M7: SVG/PNG Export

```mermaid
graph TB
    snapshot["model.Snapshot"]
    svgRenderer["SVGRenderer"]
    nodeLayout["node layout engine\ntree → x/y coordinates"]
    svgEdges["edge renderer\nparent-child lines + dotted channels"]
    svgLabels["label renderer\ngoroutine ID + state + color"]
    svgDoc["SVG document\n<svg> string"]
    fileWriter["file writer\n--export path or 'e' keybinding"]
    pngConvert["PNG convert\noptional: rsvg-convert / inkscape"]

    snapshot --> svgRenderer
    svgRenderer --> nodeLayout
    nodeLayout --> svgEdges
    nodeLayout --> svgLabels
    svgEdges --> svgDoc
    svgLabels --> svgDoc
    svgDoc --> fileWriter
    fileWriter --> pngConvert
```

---

## Diagram 10 — M8: Flamegraph Overlay

```mermaid
graph TB
    pprofFetcher["PprofFetcher"]
    cpuEndpoint["GET /debug/pprof/profile?seconds=5"]
    cpuParser["CPU profile parser\nbinary protobuf → pprof.Profile"]
    callTree["CallTree\n{Function, Samples, Children}"]
    weightMap["weight map\nfunction → CPU samples"]
    treeRenderer["RenderTree\nannotate nodes with CPU weight"]
    flamegraphView["flamegraph view\ntoggle with 'f' key"]
    cpuBadge["CPU badge\n🔥 34% beside goroutine label"]

    pprofFetcher --> cpuEndpoint
    cpuEndpoint --> cpuParser
    cpuParser --> callTree
    callTree --> weightMap
    weightMap --> treeRenderer
    treeRenderer --> flamegraphView
    flamegraphView --> cpuBadge
```
