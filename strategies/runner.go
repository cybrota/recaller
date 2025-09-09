// Copyright 2025 Naren Yellavula
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package strategies

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"time"
)

const (
	DefaultCmdTimeout = 30 * time.Second
	FastCmdTimeout    = 5 * time.Second
	GitCmdTimeout     = 15 * time.Second
	HttpTimeout       = 10 * time.Second
	MaxOutputSize     = 1024 * 1024 // 1MB
	MaxTldrSize       = 512 * 1024  // 512KB
)

// CommandRunner handles command execution with timeouts and size limits
type CommandRunner struct{}

// NewCommandRunner creates a new command runner
func NewCommandRunner() *CommandRunner {
	return &CommandRunner{}
}

// RunWithTimeout runs a command with specified timeout and size limit
func (cr *CommandRunner) RunWithTimeout(timeout time.Duration, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)

	var buf bytes.Buffer
	limitedWriter := &LimitedWriter{w: &buf, limit: MaxOutputSize}
	cmd.Stdout = limitedWriter
	cmd.Stderr = limitedWriter

	err := cmd.Run()
	result := buf.String()

	if limitedWriter.truncated {
		result += "\n[OUTPUT TRUNCATED - Size limit exceeded]"
	}

	return result, err
}

// Run runs a command with default timeout
func (cr *CommandRunner) Run(name string, args ...string) (string, error) {
	return cr.RunWithTimeout(DefaultCmdTimeout, name, args...)
}

// RunFast runs a command with short timeout
func (cr *CommandRunner) RunFast(name string, args ...string) (string, error) {
	return cr.RunWithTimeout(FastCmdTimeout, name, args...)
}

// CheckCommandExists checks if a command exists using "which" or similar
func (cr *CommandRunner) CheckCommandExists(cmd string) bool {
	_, err := cr.RunFast("which", cmd)
	return err == nil
}

// LimitedWriter implements io.Writer with size limiting
type LimitedWriter struct {
	w         io.Writer
	limit     int64
	written   int64
	truncated bool
}

func (lw *LimitedWriter) Write(p []byte) (n int, err error) {
	if lw.written >= lw.limit {
		lw.truncated = true
		return len(p), nil
	}

	remaining := lw.limit - lw.written
	if int64(len(p)) > remaining {
		lw.truncated = true
		n, err = lw.w.Write(p[:remaining])
		lw.written += int64(n)
		return len(p), err
	}

	n, err = lw.w.Write(p)
	lw.written += int64(n)
	return n, err
}
