// Package main provides the MCP server for GDB.
package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yuang-w/cc-market/plugins/autogdb/src/internal/gdb"
)

// Server instructions for MCP clients.
const serverInstructions = `MCP server that wraps GDB over the CLI. One GDB session per server process. Call create_session (empty socket_path spawns a local GDB subprocess; or set socket_path to a Unix socket where external GDB runs autogdb-listen after loading gdb_bridge.py). Then drive GDB only via gdb_command (e.g. file /path/to/binary, break main, run, bt, info registers). Use session_status / stop_session as needed. All tools return a JSON object (dict).`

// Session state
var (
	controller gdb.Controller
	ctrlMu     sync.Mutex
)

// --- Input types for tools ---

type createSessionInput struct {
	SocketPath string  `json:"socket_path" jsonschema:"optional Unix socket path; empty for subprocess mode"`
	Cwd        string  `json:"cwd" jsonschema:"optional working directory for subprocess mode; defaults to current directory; no  effect in socket mode"`
	TimeoutSec float64 `json:"timeout_sec" jsonschema:"timeout in seconds for session creation"`
}

type sessionStatusInput struct{}

type stopSessionInput struct{}

type gdbCommandInput struct {
	Command    string  `json:"command" jsonschema:"the GDB CLI command to execute"`
	TimeoutSec float64 `json:"timeout_sec" jsonschema:"timeout in seconds for the command"`
}

// --- Output types for tools ---

type createSessionOutput struct {
	Mode       string `json:"mode"`
	Banner     string `json:"banner"`
	SocketPath string `json:"socket_path,omitempty"`
	Error      string `json:"error,omitempty"`
}

type sessionStatusOutput struct {
	Alive bool   `json:"alive"`
	Error string `json:"error,omitempty"`
}

type stopSessionOutput struct {
	Stopped bool   `json:"stopped"`
	Error   string `json:"error,omitempty"`
}

type gdbCommandOutput struct {
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
}

// --- Tool handlers ---

func createSessionHandler(ctx context.Context, req *mcp.CallToolRequest, args createSessionInput) (*mcp.CallToolResult, createSessionOutput, error) {
	ctrlMu.Lock()
	defer ctrlMu.Unlock()

	// Set default timeout
	timeout := time.Duration(args.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = gdb.DefaultTimeout
	}

	// Stop any existing session
	if controller != nil {
		controller.Exit()
		controller = nil
	}

	socketPath := strings.TrimSpace(args.SocketPath)
	var result createSessionOutput

	if socketPath != "" {
		// Socket mode
		c, err := gdb.NewSocketGdbController(socketPath)
		if err != nil {
			return nil, createSessionOutput{Error: fmt.Sprintf("could not connect to socket: %s", err)}, nil
		}
		controller = c

		// Run auto-config
		runAutoConfig(controller, timeout)

		// Get banner
		banner, err := controller.RunCLI("show version", timeout)
		if err != nil {
			controller.Exit()
			controller = nil
			return nil, createSessionOutput{Error: fmt.Sprintf("timeout or error on show version (is autogdb-listen running?): %s", err)}, nil
		}
		banner = strings.TrimSpace(banner)
		if banner == "" {
			banner = "(connected)"
		}
		result = createSessionOutput{
			Mode:       "socket",
			SocketPath: socketPath,
			Banner:     banner,
		}
	} else {
		// Subprocess mode
		cwd := args.Cwd
		if cwd == "" {
			cwd = ""
		}
		c, err := gdb.NewGdbCliController(cwd)
		if err != nil {
			return nil, createSessionOutput{Error: err.Error()}, nil
		}
		controller = c

		// Get banner
		banner, err := controller.RunCLI("show version", timeout)
		if err != nil {
			controller.Exit()
			controller = nil
			return nil, createSessionOutput{Error: fmt.Sprintf("timeout or error during startup: %s", err)}, nil
		}
		banner = strings.TrimSpace(banner)
		if banner == "" {
			banner = "(gdb subprocess started)"
		}
		result = createSessionOutput{
			Mode:   "subprocess",
			Banner: banner,
		}
	}

	return nil, result, nil
}

func runAutoConfig(ctrl gdb.Controller, timeout time.Duration) {
	configCmds := []string{
		"set pagination off",
		"set confirm off",
		"set breakpoint pending on",
	}
	for _, cmd := range configCmds {
		// Ignore errors during auto-config
		_, _ = ctrl.RunCLI(cmd, timeout)
	}
}

func sessionStatusHandler(ctx context.Context, req *mcp.CallToolRequest, args sessionStatusInput) (*mcp.CallToolResult, sessionStatusOutput, error) {
	ctrlMu.Lock()
	defer ctrlMu.Unlock()

	if controller == nil {
		return nil, sessionStatusOutput{Alive: false, Error: "no session"}, nil
	}

	alive := controller.IsAlive()
	if !alive {
		controller = nil
	}
	return nil, sessionStatusOutput{Alive: alive}, nil
}

func stopSessionHandler(ctx context.Context, req *mcp.CallToolRequest, args stopSessionInput) (*mcp.CallToolResult, stopSessionOutput, error) {
	ctrlMu.Lock()
	defer ctrlMu.Unlock()

	if controller == nil {
		return nil, stopSessionOutput{Stopped: false, Error: "no session"}, nil
	}

	ctrl := controller
	controller = nil
	ctrl.Exit()

	return nil, stopSessionOutput{Stopped: true}, nil
}

func gdbCommandHandler(ctx context.Context, req *mcp.CallToolRequest, args gdbCommandInput) (*mcp.CallToolResult, gdbCommandOutput, error) {
	ctrlMu.Lock()
	defer ctrlMu.Unlock()

	// Set default timeout
	timeout := time.Duration(args.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = gdb.DefaultTimeout
	}

	if controller == nil {
		return nil, gdbCommandOutput{Output: "", Error: "no session"}, nil
	}

	if !controller.IsAlive() {
		controller = nil
		return nil, gdbCommandOutput{Output: "", Error: "session dead"}, nil
	}

	output, err := controller.RunCLI(args.Command, timeout)
	if err != nil {
		// Check for timeout or bridge error
		if err == gdb.ErrTimeout {
			return nil, gdbCommandOutput{Output: "", Error: fmt.Sprintf("timeout: %s", err)}, nil
		}
		if bridgeErr, ok := err.(*gdb.BridgeError); ok {
			return nil, gdbCommandOutput{Output: bridgeErr.Output, Error: bridgeErr.Message}, nil
		}
		return nil, gdbCommandOutput{Output: "", Error: err.Error()}, nil
	}

	return nil, gdbCommandOutput{Output: output}, nil
}

func main() {
	opts := &mcp.ServerOptions{
		Instructions: serverInstructions,
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "autogdb",
		Version: "1.0.0",
	}, opts)

	// Register tools
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_session",
		Description: "Create or replace the GDB session (subprocess or Unix socket). Returns mode, banner; socket_path in socket mode.",
	}, createSessionHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "session_status",
		Description: "Check if the current GDB session is still alive. Returns alive (bool), optional error.",
	}, sessionStatusHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "stop_session",
		Description: "Shut down the GDB session and clear it. Returns stopped (bool), optional error.",
	}, stopSessionHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "gdb_command",
		Description: "Execute one GDB CLI command. Examples: file /path/exe, set args --foo, break main, run, next, print x, bt, info locals, attach 1234, core /path/core.",
	}, gdbCommandHandler)

	// Run the server on stdio transport
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
