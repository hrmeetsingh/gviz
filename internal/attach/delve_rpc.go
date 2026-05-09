package attach

import (
	"fmt"
	"net"
	"time"

	"github.com/go-delve/delve/pkg/goversion"
	"github.com/go-delve/delve/service"
	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/debugger"
	"github.com/go-delve/delve/service/rpc2"
	"github.com/go-delve/delve/service/rpccommon"
)

// delveRPCClient wraps a real rpc2.RPCClient and implements DelveClient.
type delveRPCClient struct {
	rpc       *rpc2.RPCClient
	server    *rpccommon.ServerImpl
	goVersion *goversion.GoVersion
}

func (c *delveRPCClient) ListGoroutines() ([]*DelveGoroutine, error) {
	gs, _, err := c.rpc.ListGoroutines(0, 10000)
	if err != nil {
		return nil, err
	}
	result := make([]*DelveGoroutine, len(gs))
	for i, g := range gs {
		result[i] = apiGoroutineToDelve(g, c.goVersion)
	}
	return result, nil
}

func (c *delveRPCClient) Stacktrace(goroutineID int64, depth int) ([]DelveFrame, error) {
	frames, err := c.rpc.Stacktrace(goroutineID, depth, 0, api.StacktraceOptions(0), nil)
	if err != nil {
		return nil, err
	}
	result := make([]DelveFrame, len(frames))
	for i, f := range frames {
		result[i] = DelveFrame{
			Function: f.Function.Name(),
			File:     f.File,
			Line:     f.Line,
		}
	}
	return result, nil
}

func (c *delveRPCClient) Close() error {
	if c.rpc != nil {
		c.rpc.Detach(false)
	}
	if c.server != nil {
		return c.server.Stop()
	}
	return nil
}

func apiGoroutineToDelve(g *api.Goroutine, goVer *goversion.GoVersion) *DelveGoroutine {
	dg := &DelveGoroutine{
		ID:       g.ID,
		Status:   int(g.Status),
		ThreadID: g.ThreadID,
		UserCurrentLoc: DelveLoc{
			Function: g.UserCurrentLoc.Function.Name(),
			File:     g.UserCurrentLoc.File,
			Line:     g.UserCurrentLoc.Line,
		},
		GoStatementLoc: DelveLoc{
			Function: g.GoStatementLoc.Function.Name(),
			File:     g.GoStatementLoc.File,
			Line:     g.GoStatementLoc.Line,
		},
	}
	if g.WaitReason > 0 && goVer != nil {
		dg.WaitReason = api.WaitReasonString(goVer, g.WaitReason)
	}
	return dg
}

// DelveServerConfig holds the parameters needed to start a Delve debug server.
type DelveServerConfig struct {
	PID        int    // attach to existing process
	BinaryPath string // launch this binary under Delve
	Addr       string // connect to an already-running headless server (no server started)
}

func newDelveRPCClient(rpcClient *rpc2.RPCClient, server *rpccommon.ServerImpl) *delveRPCClient {
	c := &delveRPCClient{rpc: rpcClient, server: server}
	if ver := rpcClient.GetVersion(); ver != nil && ver.TargetGoVersion != "" {
		parsed, ok := goversion.Parse(ver.TargetGoVersion)
		if ok {
			c.goVersion = &parsed
		}
	}
	if c.goVersion == nil {
		c.goVersion = &goversion.GoVersion{Major: 1, Minor: 25}
	}
	return c
}

// StartDelveServer starts a Delve headless debug server and returns a
// DelveClient connected to it. The caller must call Close() when done.
func StartDelveServer(cfg DelveServerConfig) (DelveClient, error) {
	if cfg.Addr != "" {
		rpcClient := rpc2.NewClient(cfg.Addr)
		return newDelveRPCClient(rpcClient, nil), nil
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("delve listen: %w", err)
	}

	dbgCfg := debugger.Config{
		AttachPid:      cfg.PID,
		Backend:        "default",
		CheckGoVersion: true,
	}

	var processArgs []string
	if cfg.BinaryPath != "" {
		processArgs = []string{cfg.BinaryPath}
		dbgCfg.AttachPid = 0
	}

	srvCfg := &service.Config{
		Debugger:    dbgCfg,
		Listener:    listener,
		ProcessArgs: processArgs,
		APIVersion:  2,
	}

	srv := rpccommon.NewServer(srvCfg)

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Run() }()

	// Give the server a moment to start and check for immediate failure.
	select {
	case err := <-errCh:
		return nil, fmt.Errorf("delve server start: %w", err)
	case <-time.After(2 * time.Second):
	}

	rpcClient := rpc2.NewClient(listener.Addr().String())
	return newDelveRPCClient(rpcClient, srv), nil
}
