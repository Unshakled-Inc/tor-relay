package bridge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testFingerprint = "0123456789ABCDEF0123456789ABCDEF01234567"

func TestLine(t *testing.T) {
	dir := t.TempDir()
	bridgeFile := filepath.Join(dir, "obfs4_bridgeline.txt")
	fingerprintFile := filepath.Join(dir, "fingerprint")

	content := "# obfs4 torrc client bridge line\n" +
		"#\n" +
		"Bridge obfs4 <IP ADDRESS>:<PORT> <FINGERPRINT> cert=abcdef iat-mode=0\n"
	if err := os.WriteFile(bridgeFile, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fingerprintFile, []byte("TestBridge "+testFingerprint+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	line, err := Line(bridgeFile, fingerprintFile, "203.0.113.10", 8443)
	if err != nil {
		t.Fatalf("Line: %v", err)
	}
	want := "Bridge obfs4 203.0.113.10:8443 " + testFingerprint + " cert=abcdef iat-mode=0"
	if line != want {
		t.Fatalf("Line = %q, want %q", line, want)
	}

	line, err = Line(bridgeFile, fingerprintFile, "", 0)
	if err != nil {
		t.Fatalf("Line without address: %v", err)
	}
	want = "Bridge obfs4 <IP ADDRESS>:<PORT> " + testFingerprint + " cert=abcdef iat-mode=0"
	if line != want {
		t.Fatalf("Line = %q, want %q", line, want)
	}
}

func TestLineErrors(t *testing.T) {
	dir := t.TempDir()
	bridgeFile := filepath.Join(dir, "obfs4_bridgeline.txt")
	fingerprintFile := filepath.Join(dir, "fingerprint")

	if err := os.WriteFile(fingerprintFile, []byte("TestBridge "+testFingerprint+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := Line(filepath.Join(dir, "missing"), fingerprintFile, "", 0); err == nil {
		t.Fatal("want error for a missing bridge file")
	} else if !strings.Contains(err.Error(), "read bridge line") {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := os.WriteFile(bridgeFile, []byte("# no bridge line here\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Line(bridgeFile, fingerprintFile, "", 0); err == nil {
		t.Fatal("want error for a file without a bridge line")
	} else if !strings.Contains(err.Error(), "does not contain a Bridge obfs4 line") {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := os.WriteFile(bridgeFile, []byte("Bridge obfs4 <IP ADDRESS>:<PORT> <FINGERPRINT> cert=x iat-mode=0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Line(bridgeFile, filepath.Join(dir, "missing-fingerprint"), "", 0); err == nil {
		t.Fatal("want error for a missing fingerprint file")
	} else if !strings.Contains(err.Error(), "read fingerprint") {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := os.WriteFile(fingerprintFile, []byte("TestBridge\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Line(bridgeFile, fingerprintFile, "", 0); err == nil {
		t.Fatal("want error for a fingerprint file without a fingerprint")
	}
}
