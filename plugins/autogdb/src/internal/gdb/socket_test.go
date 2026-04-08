//go:build !windows

package gdb

import (
	"encoding/json"
	"net"
	"path/filepath"
	"testing"
	"time"
)

// mockBridgeServer simulates gdb_bridge.py for testing.
type mockBridgeServer struct {
	listener net.Listener
	responses chan socketResponse
	done      chan struct{}
}

func newMockBridgeServer(t *testing.T) *mockBridgeServer {
	// Create a temp directory for the socket
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to create Unix socket listener: %v", err)
	}

	s := &mockBridgeServer{
		listener:  listener,
		responses: make(chan socketResponse, 10),
		done:      make(chan struct{}),
	}

	go s.serve()

	return s
}

func (s *mockBridgeServer) socketPath() string {
	return s.listener.Addr().String()
}

func (s *mockBridgeServer) serve() {
	for {
		select {
		case <-s.done:
			return
		default:
		}

		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn)
	}
}

func (s *mockBridgeServer) handleConn(conn net.Conn) {
	defer conn.Close()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		select {
		case <-s.done:
			return
		default:
		}

		var req socketRequest
		if err := decoder.Decode(&req); err != nil {
			return
		}

		// Get response from channel or use default
		var resp socketResponse
		select {
		case resp = <-s.responses:
		default:
			resp = socketResponse{Output: "mock output for: " + req.Command}
		}

		encoder.Encode(resp)
	}
}

func (s *mockBridgeServer) close() {
	close(s.done)
	s.listener.Close()
}

func TestSocketGdbControllerBasic(t *testing.T) {
	server := newMockBridgeServer(t)
	defer server.close()

	ctrl, err := NewSocketGdbController(server.socketPath())
	if err != nil {
		t.Fatalf("NewSocketGdbController() failed: %v", err)
	}
	defer ctrl.Exit()

	if !ctrl.IsAlive() {
		t.Error("Controller should be alive after connection")
	}

	poll := ctrl.Process().Poll()
	if poll != nil {
		t.Errorf("Process().Poll() = %d, want nil (alive)", *poll)
	}
}

func TestSocketGdbControllerRunCLI(t *testing.T) {
	server := newMockBridgeServer(t)
	defer server.close()

	ctrl, err := NewSocketGdbController(server.socketPath())
	if err != nil {
		t.Fatalf("NewSocketGdbController() failed: %v", err)
	}
	defer ctrl.Exit()

	// Queue a specific response
	server.responses <- socketResponse{Output: "test output\n"}

	output, err := ctrl.RunCLI("test command", 5*time.Second)
	if err != nil {
		t.Errorf("RunCLI() failed: %v", err)
	}
	if output != "test output" {
		t.Errorf("RunCLI() output = %q, want %q", output, "test output")
	}
}

func TestSocketGdbControllerError(t *testing.T) {
	server := newMockBridgeServer(t)
	defer server.close()

	ctrl, err := NewSocketGdbController(server.socketPath())
	if err != nil {
		t.Fatalf("NewSocketGdbController() failed: %v", err)
	}
	defer ctrl.Exit()

	// Queue an error response
	server.responses <- socketResponse{Output: "partial", Error: "command failed"}

	output, err := ctrl.RunCLI("bad command", 5*time.Second)

	if err == nil {
		t.Error("RunCLI() should return error when bridge returns error")
	}

	bridgeErr, ok := err.(*BridgeError)
	if !ok {
		t.Errorf("RunCLI() error = %T, want *BridgeError", err)
	} else {
		if bridgeErr.Message != "command failed" {
			t.Errorf("BridgeError.Message = %q, want %q", bridgeErr.Message, "command failed")
		}
		if bridgeErr.Output != "partial" {
			t.Errorf("BridgeError.Output = %q, want %q", bridgeErr.Output, "partial")
		}
	}

	_ = output // may be empty
}

func TestSocketGdbControllerExit(t *testing.T) {
	server := newMockBridgeServer(t)
	defer server.close()

	ctrl, err := NewSocketGdbController(server.socketPath())
	if err != nil {
		t.Fatalf("NewSocketGdbController() failed: %v", err)
	}

	if !ctrl.IsAlive() {
		t.Error("Controller should be alive before Exit()")
	}

	ctrl.Exit()

	if ctrl.IsAlive() {
		t.Error("Controller should not be alive after Exit()")
	}

	poll := ctrl.Process().Poll()
	if poll == nil {
		t.Error("Process().Poll() should return non-nil after Exit()")
	}
}

func TestSocketGdbControllerInvalidPath(t *testing.T) {
	_, err := NewSocketGdbController("/nonexistent/socket/path")
	if err == nil {
		t.Error("NewSocketGdbController() should fail for non-existent socket")
	}
}

func TestSocketGdbControllerAfterExit(t *testing.T) {
	server := newMockBridgeServer(t)
	defer server.close()

	ctrl, err := NewSocketGdbController(server.socketPath())
	if err != nil {
		t.Fatalf("NewSocketGdbController() failed: %v", err)
	}

	ctrl.Exit()

	_, err = ctrl.RunCLI("test", 1*time.Second)
	if err != ErrSessionDead {
		t.Errorf("RunCLI on dead controller: error = %v, want ErrSessionDead", err)
	}
}

func TestSocketGdbControllerDeadConnection(t *testing.T) {
	server := newMockBridgeServer(t)

	ctrl, err := NewSocketGdbController(server.socketPath())
	if err != nil {
		t.Fatalf("NewSocketGdbController() failed: %v", err)
	}

	// Close the server while client is connected
	server.close()

	// Give it a moment for the connection to be detected
	time.Sleep(100 * time.Millisecond)

	// Try to run a command - should fail
	_, err = ctrl.RunCLI("test", 1*time.Second)
	if err == nil {
		t.Error("RunCLI() should fail when server is closed")
	}

	ctrl.Exit()
}