package keys

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// TorBinary is the tor executable used by KeygenFamily. Tests can point it
// at a fake script.
var TorBinary = "/usr/bin/tor"

// KeyFile is a family key found in the source directory and the name it
// should have in the relay key directory.
type KeyFile struct {
	Source string
	Name   string
}

// InstallResult lists the names installed or replaced in the key directory.
// Only names are recorded, never key contents.
type InstallResult struct {
	Installed []string
	Replaced  []string
}

// Discover returns the family key files in dir. When familyIDSet is true,
// every regular file is accepted because Docker names a mounted secret after
// the secret name, not the source file. Names are normalized to end in
// .secret_family_key. When familyIDSet is false, only files that already end
// in .secret_family_key are returned.
func Discover(dir string, familyIDSet bool) ([]KeyFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read family key dir %s: %w", dir, err)
	}

	var files []KeyFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasSuffix(name, ".public_family_id") {
			continue
		}
		if !familyIDSet && !strings.HasSuffix(name, ".secret_family_key") {
			continue
		}

		destName := name
		if !strings.HasSuffix(destName, ".secret_family_key") {
			destName += ".secret_family_key"
		}

		files = append(files, KeyFile{
			Source: filepath.Join(dir, name),
			Name:   destName,
		})
	}

	return files, nil
}

// Validate enforces the FamilyId and key-file consistency rules. The error
// messages are the ones relay operators see.
func Validate(familyIDs []string, keyFiles []KeyFile, dir string) error {
	if len(familyIDs) > 0 && len(keyFiles) == 0 {
		return fmt.Errorf("FamilyId set but no *.secret_family_key found in %s", dir)
	}
	if len(familyIDs) == 0 && len(keyFiles) > 0 {
		return fmt.Errorf("family key present but TOR_FAMILY_ID not set")
	}
	return nil
}

// Install copies the family keys into destDir with mode 0600. The directory
// is created or reset to mode 0700. A destination file with different
// content is replaced, and its name is recorded in Replaced.
func Install(keyFiles []KeyFile, destDir string) (InstallResult, error) {
	var result InstallResult

	if err := os.MkdirAll(destDir, 0700); err != nil {
		return result, fmt.Errorf("create key dir %s: %w", destDir, err)
	}
	if err := os.Chmod(destDir, 0700); err != nil {
		return result, fmt.Errorf("chmod key dir %s: %w", destDir, err)
	}

	for _, keyFile := range keyFiles {
		data, err := os.ReadFile(keyFile.Source)
		if err != nil {
			return result, fmt.Errorf("read family key %s: %w", keyFile.Source, err)
		}

		dest := filepath.Join(destDir, keyFile.Name)
		existing, err := os.ReadFile(dest)
		if err == nil && bytes.Equal(existing, data) {
			continue
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return result, fmt.Errorf("read existing family key %s: %w", dest, err)
		}

		if err := os.Remove(dest); err != nil && !errors.Is(err, os.ErrNotExist) {
			return result, fmt.Errorf("remove existing family key %s: %w", dest, err)
		}
		if err := os.WriteFile(dest, data, 0600); err != nil {
			return result, fmt.Errorf("write family key %s: %w", dest, err)
		}
		if err := os.Chmod(dest, 0600); err != nil {
			return result, fmt.Errorf("chmod family key %s: %w", dest, err)
		}

		if existing != nil {
			result.Replaced = append(result.Replaced, keyFile.Name)
		} else {
			result.Installed = append(result.Installed, keyFile.Name)
		}
	}

	return result, nil
}

// Fingerprints reads the RSA and ed25519 relay fingerprints that tor writes
// after its first start. Each file contains "<nickname> <fingerprint>".
func Fingerprints(dir string) (rsa, ed25519 string, err error) {
	rsa, err = readFingerprint(filepath.Join(dir, "fingerprint"))
	if err != nil {
		return "", "", err
	}
	ed25519, err = readFingerprint(filepath.Join(dir, "fingerprint-ed25519"))
	if err != nil {
		return "", "", err
	}
	return rsa, ed25519, nil
}

func readFingerprint(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	fields := strings.Fields(string(data))
	if len(fields) != 2 {
		return "", fmt.Errorf("%s does not contain a nickname and fingerprint", path)
	}
	return fields[1], nil
}

// KeygenFamily runs tor --keygen-family in outDir and returns the FamilyId
// line it prints. It refuses to overwrite an existing key unless force is
// true.
func KeygenFamily(name, outDir string, force bool) (string, error) {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf("invalid family name %q", name)
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", fmt.Errorf("create output dir %s: %w", outDir, err)
	}

	keyPath := filepath.Join(outDir, name+".secret_family_key")
	publicPath := filepath.Join(outDir, name+".public_family_id")

	if _, err := os.Stat(keyPath); err == nil && !force {
		return "", fmt.Errorf("refusing to overwrite %s, use --force", keyPath)
	}

	if force {
		for _, path := range []string{keyPath, publicPath} {
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				return "", fmt.Errorf("remove %s: %w", path, err)
			}
		}
	}

	tmp, err := os.MkdirTemp("", "torinit-keygen-")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmp)

	emptyTorrc := filepath.Join(tmp, "torrc")
	emptyDefaults := filepath.Join(tmp, "defaults")
	if err := os.WriteFile(emptyTorrc, nil, 0600); err != nil {
		return "", fmt.Errorf("write empty torrc: %w", err)
	}
	if err := os.WriteFile(emptyDefaults, nil, 0600); err != nil {
		return "", fmt.Errorf("write empty defaults: %w", err)
	}

	cmd := exec.Command(TorBinary,
		"--defaults-torrc", emptyDefaults,
		"-f", emptyTorrc,
		"--keygen-family", name,
	)
	cmd.Dir = outDir
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("tor --keygen-family %s: %w", name, err)
	}

	var familyLine string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "FamilyId ") {
			familyLine = line
			break
		}
	}
	if familyLine == "" {
		return "", fmt.Errorf("tor --keygen-family %s did not print a FamilyId line", name)
	}

	if _, err := os.Stat(keyPath); err != nil {
		return "", fmt.Errorf("expected key file %s: %w", keyPath, err)
	}

	return familyLine, nil
}
