package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Unshakled-Inc/tor-relay/internal/config"
	"github.com/Unshakled-Inc/tor-relay/internal/family"
	"github.com/Unshakled-Inc/tor-relay/internal/keys"
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
	case "keygen-family":
		os.Exit(keygenFamilyCmd(args[1:]))
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

func discoverAndValidateFamily(cfg *config.Config) ([]keys.KeyFile, bool) {
	keyFiles, err := keys.Discover(cfg.FamilyKeyDir, len(cfg.FamilyID) > 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "torinit: %v\n", err)
		return nil, false
	}
	if err := keys.Validate(cfg.FamilyID, keyFiles, cfg.FamilyKeyDir); err != nil {
		fmt.Fprintf(os.Stderr, "torinit: %v\n", err)
		return nil, false
	}
	return keyFiles, true
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
	if _, ok := discoverAndValidateFamily(cfg); !ok {
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
	if cfg.MyFamilyURL != "" {
		if err := family.Write(filepath.Join(verifyData, ".torinit", "family.conf"), nil); err != nil {
			fmt.Fprintf(os.Stderr, "torinit: %v\n", err)
			return 1
		}
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
	keyFiles, ok := discoverAndValidateFamily(cfg)
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

	installResult, err := keys.Install(keyFiles, filepath.Join(dataDir, "keys"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "torinit: %v\n", err)
		return 1
	}
	for _, name := range installResult.Installed {
		fmt.Fprintf(os.Stderr, "torinit: installed family key %s\n", name)
	}
	for _, name := range installResult.Replaced {
		fmt.Fprintf(os.Stderr, "torinit: replaced family key %s\n", name)
	}

	var fam *family.Manager
	if cfg.MyFamilyURL != "" {
		fam = family.New(family.Options{
			URL:             cfg.MyFamilyURL,
			Interval:        cfg.MyFamilyRefresh,
			MaxDropPct:      cfg.MyFamilyMaxDropPct,
			Path:            filepath.Join(stateDir, "family.conf"),
			FingerprintPath: filepath.Join(dataDir, "fingerprint"),
		})
		fam.Startup()
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

	stop := make(chan struct{})
	code, err := supervise.Run(cmd, filepath.Join(stateDir, "pid"), func(pid int) {
		if fam != nil {
			go fam.Loop(pid, stop)
		}
	})
	close(stop)
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

func keygenFamilyCmd(args []string) int {
	fs := flag.NewFlagSet("keygen-family", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	outDir := fs.String("out", "/out", "output directory")
	force := fs.Bool("force", false, "overwrite an existing family key")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: torinit keygen-family [--out DIR] [--force] <name>")
		return 2
	}
	line, err := keys.KeygenFamily(fs.Arg(0), *outDir, *force)
	if err != nil {
		fmt.Fprintf(os.Stderr, "torinit: %v\n", err)
		return 1
	}
	fmt.Println(line)
	return 0
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
