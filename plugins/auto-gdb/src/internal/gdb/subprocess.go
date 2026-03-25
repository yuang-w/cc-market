package gdb

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const promptSentinel = "__AUTO_GDB_EOS__"

// GdbCliController spawns GDB as a subprocess and synchronizes on a prompt sentinel.
type GdbCliController struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
	// stdout is the raw reader; we read from buf via the reader goroutine
	stdout io.Reader
	mu     sync.Mutex
	buf    strings.Builder
	done   bool // set when the process exits
}

// NewGdbCliController creates a new GDB subprocess controller.
// If cwd is non-empty, GDB is started in that directory.
func NewGdbCliController(cwd string) (*GdbCliController, error) {
	args := []string{
		"-q",
		"-nx",
		"--interpreter=console",
		"-iex", "set pagination off",
		"-iex", "set confirm off",
		"-iex", "set breakpoint pending on",
		"-iex", fmt.Sprintf("set prompt \\n%s\\n", promptSentinel),
	}

	cmd := exec.Command("gdb", args...)
	if cwd != "" {
		cmd.Dir = cwd
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Merge stderr into stdout (like Python's stderr=STDOUT)
	cmd.Stderr = cmd.Stdout

	c := &GdbCliController{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
	}

	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("failed to start gdb: %w", err)
	}

	// Start the reader goroutine
	go c.readLoop()

	// Consume startup banner up to first prompt
	// Use a shorter timeout for startup; some GDB configs may not produce output
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()
	_, _ = c.readUntilPrompt(ctx) // ignore timeout on startup

	return c, nil
}

// readLoop continuously reads from stdout and appends to the buffer.
func (c *GdbCliController) readLoop() {
	reader := bufio.NewReader(c.stdout)
	for {
		// Read a reasonable chunk
		buf := make([]byte, 4096)
		n, err := reader.Read(buf)
		if n > 0 {
			c.mu.Lock()
			c.buf.Write(buf[:n])
			c.mu.Unlock()
		}
		if err != nil {
			// Process exited or error
			c.mu.Lock()
			c.done = true
			c.mu.Unlock()
			return
		}
	}
}

// readUntilPrompt reads output until the prompt sentinel is found.
// Returns the output before the prompt, stripped of trailing whitespace.
func (c *GdbCliController) readUntilPrompt(ctx context.Context) (string, error) {
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()

	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return "", ErrTimeout
		default:
		}

		c.mu.Lock()
		// Check if prompt is in buffer
		b := c.buf.String()
		if idx := strings.Index(b, promptSentinel); idx != -1 {
			// Found prompt - extract output and reset buffer
			out := b[:idx]
			remaining := b[idx+len(promptSentinel):]
			c.buf.Reset()
			c.buf.WriteString(remaining)
			c.mu.Unlock()
			return strings.TrimRight(out, "\n\r"), nil
		}

		// Check if process is dead
		if c.done {
			c.mu.Unlock()
			return "", ErrSessionDead
		}
		c.mu.Unlock()

		// Wait for new data
		select {
		case <-ctx.Done():
			return "", ErrTimeout
		case <-ticker.C:
			// Continue polling
		}
	}
}

// RunCLI executes one GDB CLI command and returns the captured output.
func (c *GdbCliController) RunCLI(command string, timeout time.Duration) (string, error) {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return "", nil
	}

	c.mu.Lock()
	if c.cmd == nil || c.cmd.Process == nil {
		c.mu.Unlock()
		return "", ErrSessionDead
	}
	if c.done {
		c.mu.Unlock()
		return "", ErrSessionDead
	}
	c.mu.Unlock()

	// Write command to stdin
	_, err := c.stdin.Write([]byte(cmd + "\n"))
	if err != nil {
		return "", fmt.Errorf("failed to write to gdb stdin: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	out, err := c.readUntilPrompt(ctx)
	if err != nil {
		return "", err
	}

	return StripANSI(out), nil
}

// Exit shuts down the GDB process gracefully.
func (c *GdbCliController) Exit() {
	c.mu.Lock()
	if c.cmd == nil || c.cmd.Process == nil {
		c.mu.Unlock()
		return
	}
	if c.done {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()

	// Try graceful quit first
	c.mu.Lock()
	if c.stdin != nil {
		c.stdin.Write([]byte("quit\n"))
	}
	c.mu.Unlock()

	// Wait for process to exit with timeout
	done := make(chan error, 1)
	go func() {
		done <- c.cmd.Wait()
	}()

	select {
	case <-done:
		// Process exited cleanly
	case <-time.After(3 * time.Second):
		// Force kill
		c.cmd.Process.Kill()
		<-done
	}

	// Close pipes
	c.mu.Lock()
	if c.stdin != nil {
		c.stdin.Close()
	}
	c.done = true
	c.mu.Unlock()
}

// IsAlive returns true if the GDB process is still running.
func (c *GdbCliController) IsAlive() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cmd == nil || c.cmd.Process == nil {
		return false
	}
	return !c.done
}

// Process returns a ProcessStatus for session status checks.
func (c *GdbCliController) Process() ProcessStatus {
	return &gdbProcessStatus{c: c}
}

// gdbProcessStatus implements ProcessStatus for GdbCliController.
type gdbProcessStatus struct {
	c *GdbCliController
}

// Poll returns nil if the process is still running, or the exit code if dead.
func (p *gdbProcessStatus) Poll() *int {
	p.c.mu.Lock()
	defer p.c.mu.Unlock()

	if p.c.cmd == nil || p.c.cmd.Process == nil {
		code := -1
		return &code
	}

	if p.c.done {
		// Process has exited; try to get exit code
		if p.c.cmd.ProcessState != nil {
			code := p.c.cmd.ProcessState.ExitCode()
			return &code
		}
		code := -1
		return &code
	}

	return nil // still running
}