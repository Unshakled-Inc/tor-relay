package bridge

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var fingerprintRE = regexp.MustCompile(`^[0-9A-Fa-f]{40}$`)

// Line reads the obfs4 bridge line written by lyrebird and fills in the
// fingerprint. When address and obfs4Port are set, it also fills in the
// <IP ADDRESS>:<PORT> placeholder.
func Line(bridgeFile, fingerprintFile, address string, obfs4Port int) (string, error) {
	data, err := os.ReadFile(bridgeFile)
	if err != nil {
		return "", fmt.Errorf("read bridge line %s: %w", bridgeFile, err)
	}
	fp, err := readFingerprint(fingerprintFile)
	if err != nil {
		return "", err
	}

	var line string
	for _, raw := range strings.Split(string(data), "\n") {
		raw = strings.TrimSpace(raw)
		if strings.HasPrefix(raw, "Bridge obfs4 ") {
			line = raw
			break
		}
	}
	if line == "" {
		return "", fmt.Errorf("%s does not contain a Bridge obfs4 line", bridgeFile)
	}

	line = strings.ReplaceAll(line, "<FINGERPRINT>", fp)
	if address != "" && obfs4Port > 0 {
		line = strings.ReplaceAll(line, "<IP ADDRESS>:<PORT>", fmt.Sprintf("%s:%d", address, obfs4Port))
	}
	return line, nil
}

func readFingerprint(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read fingerprint %s: %w", path, err)
	}
	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return "", fmt.Errorf("%s: expected a nickname and a fingerprint", path)
	}
	fp := strings.ToUpper(strings.TrimPrefix(fields[1], "$"))
	if !fingerprintRE.MatchString(fp) {
		return "", fmt.Errorf("%s: %q is not a 40-hex fingerprint", path, fields[1])
	}
	return fp, nil
}
