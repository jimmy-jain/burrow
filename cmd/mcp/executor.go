// Package main provides the burrow-mcp binary that exposes burrow CLI commands
// as MCP (Model Context Protocol) tools for Claude integration.
package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"
)

const (
	// commandTimeout is the maximum duration for any burrow CLI command.
	commandTimeout = 60 * time.Second

	// binaryName is the expected burrow CLI binary name.
	binaryName = "burrow"

	// fallbackPath is used when the binary is not found in PATH or BURROW_PATH.
	fallbackPath = "/usr/local/bin/burrow"

	// envBurrowPath is the environment variable for overriding the binary path.
	envBurrowPath = "BURROW_PATH"
)

// ErrNotConfirmed is returned when a destructive tool is called without
// the confirmed parameter set to true.
var ErrNotConfirmed = errors.New("destructive operation requires confirmed=true")

// CommandResult holds the output from a burrow CLI invocation.
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Executor resolves and runs burrow CLI commands, capturing their output.
type Executor struct {
	binaryPath string
}

// NewExecutor creates an Executor by resolving the burrow binary location.
// Resolution order: $BURROW_PATH env var -> $PATH lookup -> /usr/local/bin/burrow.
func NewExecutor() (*Executor, error) {
	path, err := resolveBinary()
	if err != nil {
		return nil, fmt.Errorf("resolve burrow binary: %w", err)
	}
	return &Executor{binaryPath: path}, nil
}

// resolveBinary finds the burrow binary using the resolution chain.
func resolveBinary() (string, error) {
	// 1. Check BURROW_PATH environment variable.
	if envPath := os.Getenv(envBurrowPath); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}
	}

	// 2. Look up in $PATH.
	if pathBin, err := exec.LookPath(binaryName); err == nil {
		return pathBin, nil
	}

	// 3. Fall back to well-known location.
	if _, err := os.Stat(fallbackPath); err == nil {
		return fallbackPath, nil
	}

	return "", fmt.Errorf(
		"burrow binary not found: set %s, add to PATH, or install to %s",
		envBurrowPath, fallbackPath,
	)
}

// Run executes a burrow CLI command with the given arguments.
// It enforces a timeout via the provided context and returns structured output.
func (e *Executor) Run(ctx context.Context, args ...string) (*CommandResult, error) {
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, e.binaryPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := &CommandResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
			return result, nil
		}
		if ctx.Err() == context.DeadlineExceeded {
			return result, fmt.Errorf("command timed out after %s", commandTimeout)
		}
		return result, fmt.Errorf("execute burrow: %w", err)
	}

	return result, nil
}

// BinaryPath returns the resolved path to the burrow binary.
func (e *Executor) BinaryPath() string {
	return e.binaryPath
}
