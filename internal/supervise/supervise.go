package supervise

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
)

// Run starts cmd, records its PID, calls onStart with the PID when it is not
// nil, forwards termination and reload signals, and returns the child's exit
// code. A signaled child is reported as 128+signal, matching shell
// conventions.
func Run(cmd *exec.Cmd, pidPath string, onStart func(pid int)) (int, error) {
	if err := cmd.Start(); err != nil {
		return 1, fmt.Errorf("start tor: %w", err)
	}

	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(cmd.Process.Pid)+"\n"), 0600); err != nil {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		return 1, fmt.Errorf("write pid file: %w", err)
	}

	if onStart != nil {
		onStart(cmd.Process.Pid)
	}

	signals := make(chan os.Signal, 4)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	defer signal.Stop(signals)

	go func() {
		for sig := range signals {
			_ = cmd.Process.Signal(sig)
		}
	}()

	err := cmd.Wait()
	if err == nil {
		return 0, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if code := exitErr.ExitCode(); code >= 0 {
			return code, nil
		}
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			return 128 + int(status.Signal()), nil
		}
		return 1, nil
	}

	return 1, err
}
