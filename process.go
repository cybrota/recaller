package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
)

// ProcessConfig holds configuration for process execution
type ProcessConfig struct {
	Timeout       time.Duration
	MaxOutputSize int64
	KillOnTimeout bool
}

// DefaultProcessConfig returns sensible defaults
func DefaultProcessConfig() *ProcessConfig {
	return &ProcessConfig{
		Timeout:       5 * time.Minute,  // 5 minutes default
		MaxOutputSize: 10 * 1024 * 1024, // 10MB limit
		KillOnTimeout: true,
	}
}

// ProcessManager tracks active processes for cleanup
type ProcessManager struct {
	processes map[int]*exec.Cmd
	mu        sync.RWMutex
}

var globalProcessManager = &ProcessManager{
	processes: make(map[int]*exec.Cmd),
}

func (pm *ProcessManager) addProcess(cmd *exec.Cmd) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if cmd.Process != nil {
		pm.processes[cmd.Process.Pid] = cmd
	}
}

func (pm *ProcessManager) removeProcess(pid int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	delete(pm.processes, pid)
}

func (pm *ProcessManager) killAll() {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	for _, cmd := range pm.processes {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}
}

func execCommandInPTY(command string) {
	execCommandInPTYWithConfig(command, DefaultProcessConfig())
}

func execCommandInPTYWithConfig(command string, config *ProcessConfig) {
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	// Use /bin/bash instead of sh, or detect the shell from environment
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash" // fallback to bash
	}
	cmd := exec.CommandContext(ctx, shell, "-c", command)

	// Set up process group for better signal handling
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Set up signal handling BEFORE starting process
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	// Try to start the command in a pseudo-terminal, fallback to regular execution
	ptyFile, err := pty.Start(cmd)
	usePTY := err == nil

	if !usePTY {
		// Fallback to regular execution without PTY
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err := cmd.Start(); err != nil {
			fmt.Fprintln(os.Stderr, "Failed to start command:", err)
			os.Exit(1)
		}
	}

	// Ensure cleanup happens
	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			if usePTY && ptyFile != nil {
				ptyFile.Close()
			}
			if cmd.Process != nil {
				globalProcessManager.removeProcess(cmd.Process.Pid)
			}
			close(sigChan)
		})
	}
	defer cleanup()

	// Track the process
	globalProcessManager.addProcess(cmd)

	// Handle signals in a separate goroutine
	go func() {
		for sig := range sigChan {
			if cmd.Process != nil {
				// Forward signal to the entire process group
				if err := syscall.Kill(-cmd.Process.Pid, sig.(syscall.Signal)); err != nil {
					// Only log unexpected errors
					if err != syscall.ESRCH && err != syscall.EPERM {
						fmt.Fprintf(os.Stderr, "failed to forward signal %v: %v\n", sig, err)
					}
				}
			}
		}
	}()

	// Copy data between PTY and terminal with size limiting (only if using PTY)
	if usePTY {
		go func() {
			limitedReader := &io.LimitedReader{R: ptyFile, N: config.MaxOutputSize}
			_, _ = io.Copy(os.Stdout, limitedReader)
			if limitedReader.N == 0 {
				fmt.Fprintln(os.Stderr, "\n[WARNING: Output truncated - exceeded size limit]")
			}
		}()
	}

	// Wait for command completion or timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			fmt.Fprintln(os.Stderr, "Command error:", err)
		}
	case <-ctx.Done():
		if config.KillOnTimeout && cmd.Process != nil {
			fmt.Fprintln(os.Stderr, "\n[TIMEOUT: Command exceeded time limit, killing process]")
			_ = cmd.Process.Kill()
		}
		<-done // Wait for process to actually exit
	}

	// Now prompt the user
	fmt.Print("\nHit <Return/Enter> then <Ctrl/Cmd> + c to exit...")
	bufio.NewReader(os.Stdin).ReadString('\n')
}
