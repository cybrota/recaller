package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
)

func execCommandInPTY(command string) {
	cmd := exec.Command("sh", "-c", command)

	// Start the command in a pseudo-terminal.
	ptyFile, err := pty.Start(cmd)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to start PTY:", err)
		os.Exit(1)
	}
	defer ptyFile.Close()

	// Set up signal handling (we forward SIGINT and SIGTERM).
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		for sig := range sigChan {
			if cmd.Process != nil {
				// Forward the signal to the entire process group.
				if err := syscall.Kill(-cmd.Process.Pid, sig.(syscall.Signal)); err != nil {
					if err != syscall.ESRCH && err != syscall.EPERM {
						fmt.Fprintf(os.Stderr, "failed to forward signal %v: %v\n", sig, err)
					}
				}
			}
		}
	}()

	// Copy data between the PTY and the real terminal.
	go func() {
		_, _ = io.Copy(os.Stdout, ptyFile)
	}()

	// Wait for the command to complete.
	if err := cmd.Wait(); err != nil {
		fmt.Fprintln(os.Stderr, "Command error:", err)
	}

	// Clean up signal handling.
	signal.Stop(sigChan)
	close(sigChan)

	// Now prompt the user.
	fmt.Print("\nHit <Return/Enter> then <Ctrl/Cmd> + c to exit...")
	bufio.NewReader(os.Stdin).ReadString('\n')
}
