package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Unshakled-Inc/tor-relay/internal/config"
)

func TestGolden(t *testing.T) {
	tests := []struct {
		name string
		env  []string
		file string
	}{
		{
			name: "middle minimal",
			env: []string{
				"TOR_ROLE=middle",
				"TOR_NICKNAME=TestRelay",
				"TOR_CONTACT=test@example.com",
			},
			file: "middle-minimal.torrc",
		},
		{
			name: "middle all options",
			env: []string{
				"TOR_ROLE=middle",
				"TOR_NICKNAME=TestRelay",
				"TOR_CONTACT=test@example.com",
				"TOR_ORPORT=19001",
				"TOR_ADDRESS=203.0.113.10",
				"TOR_BANDWIDTH_RATE=50 MBytes",
				"TOR_BANDWIDTH_BURST=100 MBytes",
				"TOR_SANDBOX=0",
				"TOR_LOG_LEVEL=debug",
			},
			file: "middle-all.torrc",
		},
		{
			name: "middle family",
			env: []string{
				"TOR_ROLE=middle",
				"TOR_NICKNAME=TestRelay",
				"TOR_CONTACT=test@example.com",
				"TOR_FAMILY_ID=id1 id2",
			},
			file: "middle-family.torrc",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := config.Load(tc.env)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			want, err := os.ReadFile(filepath.Join("..", "..", "testdata", "golden", tc.file))
			if err != nil {
				t.Fatalf("read golden: %v", err)
			}
			got := Render(cfg, "/var/lib/tor", false)
			if got != string(want) {
				t.Errorf("rendered torrc mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
			}
		})
	}
}

func TestRenderInclude(t *testing.T) {
	cfg, err := config.Load([]string{
		"TOR_ROLE=middle",
		"TOR_NICKNAME=TestRelay",
		"TOR_CONTACT=test@example.com",
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := Render(cfg, "/var/lib/tor", true)
	if !strings.Contains(got, "%include /etc/tor/torrc.d/\n") {
		t.Fatalf("missing include line:\n%s", got)
	}
	if !strings.HasSuffix(got, "%include /etc/tor/torrc.d/\n") {
		t.Fatalf("include line is not last:\n%s", got)
	}
}

func TestIncludeTorrcD(t *testing.T) {
	dir := t.TempDir()

	if IncludeTorrcD(dir) {
		t.Error("empty dir should not be included")
	}

	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if IncludeTorrcD(dir) {
		t.Error("dir without .conf should not be included")
	}

	if err := os.WriteFile(filepath.Join(dir, "local.conf"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if !IncludeTorrcD(dir) {
		t.Error("dir with .conf should be included")
	}

	if IncludeTorrcD(filepath.Join(dir, "missing")) {
		t.Error("missing dir should not be included")
	}
}
