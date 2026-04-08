//go:build !windows

package gdb

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
)

// SocketGdbController connects to an external GDB via Unix socket
// (where gdb_bridge.py is listening).
type SocketGdbController struct {
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
	alive  bool
}

// socketRequest represents a JSON request sent to gdb_bridge.
type socketRequest struct {
	Command string  `json:"command"`
	Timeout float64 `json:"timeout"`
}

// socketResponse represents a JSON response from gdb_bridge.
type socketResponse struct {
	Output string `json:"output"`
	Error  string `json:"error"`
}

// NewSocketGdbController creates a new socket controller connected to the given Unix socket path.
func NewSocketGdbController(socketPath string) (*SocketGdbController, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to socket %s: %w", socketPath, err)
	}

	c := &SocketGdbController{
		conn:   conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
		alive:  true,
	}

	return c, nil
}

// RunCLI executes one GDB CLI command and returns the output.
func (c *SocketGdbController) RunCLI(command string, timeout time.Duration) (string, error) {
	if !c.alive {
		return "", ErrSessionDead
	}

	cliStr := strings.TrimRight(command, "\n")
	timeoutSec := timeout.Seconds()

	// Set socket deadline
	deadline := time.Now().Add(timeout + 2*time.Second)
	if err := c.conn.SetDeadline(deadline); err != nil {
		c.markDead()
		return "", &BridgeError{Message: fmt.Sprintf("failed to set deadline: %v", err)}
	}

	// Send request
	req := socketRequest{
		Command: cliStr,
		Timeout: timeoutSec,
	}
	reqData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	if _, err := c.writer.Write(reqData); err != nil {
		c.markDead()
		return "", &BridgeError{Message: fmt.Sprintf("write error: %v", err)}
	}
	if err := c.writer.WriteByte('\n'); err != nil {
		c.markDead()
		return "", &BridgeError{Message: fmt.Sprintf("write error: %v", err)}
	}
	if err := c.writer.Flush(); err != nil {
		c.markDead()
		return "", &BridgeError{Message: fmt.Sprintf("flush error: %v", err)}
	}

	// Read response
	line, err := c.reader.ReadString('\n')
	if err != nil {
		c.markDead()
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return "", ErrTimeout
		}
		return "", &BridgeError{Message: fmt.Sprintf("connection closed before response: %v", err)}
	}

	// Parse response
	var resp socketResponse
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		return "", &BridgeError{Message: fmt.Sprintf("invalid bridge response: %v", err)}
	}

	// Check for bridge error
	if resp.Error != "" {
		return "", &BridgeError{
			Message: resp.Error,
			Output:  StripANSI(resp.Output),
		}
	}

	return StripANSI(strings.TrimSpace(resp.Output)), nil
}

// Exit shuts down the socket connection.
func (c *SocketGdbController) Exit() {
	c.markDead()
	if c.writer != nil {
		c.writer.Flush()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}

// IsAlive returns true if the socket connection is still alive.
func (c *SocketGdbController) IsAlive() bool {
	return c.alive
}

// Process returns a ProcessStatus for session status checks.
func (c *SocketGdbController) Process() ProcessStatus {
	return &socketProcessStatus{c: c}
}

// markDead marks the connection as dead.
func (c *SocketGdbController) markDead() {
	c.alive = false
}

// socketProcessStatus implements ProcessStatus for SocketGdbController.
type socketProcessStatus struct {
	c *SocketGdbController
}

// Poll returns nil if the connection is alive, or -1 if dead.
func (p *socketProcessStatus) Poll() *int {
	if p.c.alive {
		return nil
	}
	code := -1
	return &code
}