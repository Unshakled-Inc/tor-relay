package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Unshakled-Inc/tor-relay/internal/config"
	"github.com/Unshakled-Inc/tor-relay/internal/render"
	"github.com/Unshakled-Inc/tor-relay/internal/supervise"
)

const (
	torBinary = "/usr/bin/tor"
	dataDir   = "/var/lib/tor"
	stateDir  = "/var/lib/tor/.torinit"
)

var version = "dev"

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		os.Exit(runCmd())
	}
	switch args[0] {
	case "render":
		os.Exit(renderCmd(args[1:]))
	case "verify":
		os.Exit(verifyCmd(args[1:]))
	case "run":
		os.Exit(runCmd())
	case "version":
		os.Exit(versionCmd())
	case "healthcheck":
		os.Exit(healthcheckCmd())
	default:
		fmt.Fprintf(os.Stderr, "torinit: unknown subcommand %q\n", args[0])
		os.Exit(2)
	}
}

func loadConfig() (*config.Config, bool) {
	cfg, err := config.Load(os.Environ())
	if err != nil {
		fmt.Fprintf(os.Stderr, "torinit: %v\n", err)
		return nil, false
	}
	return cfg, true
}

func renderCmd(args []string) int {
	if len(args) != 0 {
		fmt.Fprintf(os.Stderr, "torinit: render takes no arguments\n")
		return 2
	}
	cfg, ok := loadConfig()
	if !ok {
		return 1
	}
	fmt.Print(render.Render(cfg, dataDir, render.IncludeTorrcD(render.TorrcD)))
	return 0
}

func verifyCmd(args []string) int {
	if len(args) != 0 {
		fmt.Fprintf(os.Stderr, "torinit: verify takes no arguments\n")
		return 2
	}
	cfg, ok := loadConfig()
	if !ok {
		return 1
	}

	tmp, err := os.MkdirTemp("", "torinit-verify-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "torinit: %v\n", err)
		return 1
	}
	defer os.RemoveAll(tmp)

	verifyData := filepath.Join(tmp, "data")
	if err := os.Mkdir(verifyData, 0700); err != nil {
		fmt.Fprintf(os.Stderr, "torinit: %v\n", err)
		return 1
	}

	torrcPath := filepath.Join(tmp, "torrc")
	defaultsPath := filepath.Join(tmp, "defaults")
	if err := os.WriteFile(torrcPath, []byte(render.Render(cfg, verifyData, render.IncludeTorrcD(render.TorrcD))), 0600); err != nil {
		fmt.Fprintf(os.Stderr, "torinit: %v\n", err)
		return 1
	}
	if err := os.WriteFile(defaultsPath, nil, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "torinit: %v\n", err)
		return 1
	}

	cmd := exec.Command(torBinary, "--verify-config", "-f", torrcPath, "--defaults-torrc", defaultsPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return exitCode(cmd.Run())
}

func runCmd() int {
	cfg, ok := loadConfig()
	if !ok {
		return 1
	}

	if err := os.MkdirAll(dataDir, 0700); err != nil {
		fmt.Fprintf(os.Stderr, "torinit: %v\n", err)
		return 1
	}
	if err := os.Chmod(dataDir, 0700); err != nil {
		fmt.Fprintf(os.Stderr, "torinit: %v\n", err)
		return 1
	}
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		fmt.Fprintf(os.Stderr, "torinit: %v\n", err)
		return 1
	}
	if err := os.Chmod(stateDir, 0700); err != nil {
		fmt.Fprintf(os.Stderr, "torinit: %v\n", err)
		return 1
	}

	torrcPath := filepath.Join(stateDir, "torrc")
	defaultsPath := filepath.Join(stateDir, "defaults")
	if err := os.WriteFile(torrcPath, []byte(render.Render(cfg, dataDir, render.IncludeTorrcD(render.TorrcD))), 0600); err != nil {
		fmt.Fprintf(os.Stderr, "torinit: %v\n", err)
		return 1
	}
	if err := os.WriteFile(defaultsPath, nil, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "torinit: %v\n", err)
		return 1
	}

	cmd := exec.Command(torBinary, "-f", torrcPath, "--defaults-torrc", defaultsPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	code, err := supervise.Run(cmd, filepath.Join(stateDir, "pid"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "torinit: %v\n", err)
		return 1
	}
	return code
}

func versionCmd() int {
	fmt.Printf("torinit %s\n", version)
	cmd := exec.Command(torBinary, "--version")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return exitCode(cmd.Run())
}

func healthcheckCmd() int {
	data, err := os.ReadFile(filepath.Join(stateDir, "pid"))
	if err != nil {
		return 1
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 1
	}
	if _, err := os.Stat(filepath.Join("/proc", strconv.Itoa(pid))); err != nil {
		return 1
	}
	return 0
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if code := exitErr.ExitCode(); code >= 0 {
			return code
		}
		return 1
	}
	fmt.Fprintf(os.Stderr, "torinit: %v\n", err)
	return 1
}
