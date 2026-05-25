package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/gofrs/flock"
)

func main() {
	os.Exit(run())
}

func run() int {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "Usage: filelock <lock-file> <timeout-seconds> <command> [args...]")
		fmt.Fprintln(os.Stderr, "Example: filelock /path/to/.lock 10 cat file.txt")
		return 1
	}

	lockPath := os.Args[1]
	timeoutSec := os.Args[2]
	cmdName := os.Args[3]
	cmdArgs := os.Args[4:]

	// Parse timeout
	var timeout time.Duration
	if timeoutSec == "0" {
		timeout = 0 // wait forever
	} else {
		var seconds int
		if _, err := fmt.Sscanf(timeoutSec, "%d", &seconds); err != nil {
			fmt.Fprintf(os.Stderr, "Invalid timeout: %s\n", timeoutSec)
			return 1
		}
		timeout = time.Duration(seconds) * time.Second
	}

	// Create flock instance
	lock := flock.New(lockPath)

	// Try to acquire lock with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	locked, err := lock.TryLockContext(ctx, 100*time.Millisecond)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to acquire lock: %v\n", err)
		return 1
	}
	if !locked {
		fmt.Fprintln(os.Stderr, "Timeout waiting for lock")
		return 1
	}
	defer func() { _ = lock.Unlock() }()

	// Execute command (inherits process context, not the lock-acquire ctx).
	cmd := exec.CommandContext(context.Background(), cmdName, cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "Failed to execute command: %v\n", err)
		return 1
	}
	return 0
}
