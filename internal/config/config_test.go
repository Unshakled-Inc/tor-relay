package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadValid(t *testing.T) {
	cfg, err := Load([]string{
		"TOR_ROLE=middle",
		"TOR_NICKNAME=TestRelay",
		"TOR_CONTACT=test@example.com",
		"TOR_ORPORT=19001",
		"TOR_ADDRESS=203.0.113.10",
		"TOR_BANDWIDTH_RATE=50 MBytes",
		"TOR_BANDWIDTH_BURST=100 MBytes",
		"TOR_SANDBOX=0",
		"TOR_LOG_LEVEL=debug",
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Role != RoleMiddle || cfg.Nickname != "TestRelay" || cfg.Contact != "test@example.com" {
		t.Errorf("unexpected identity: %+v", cfg)
	}
	if cfg.ORPort != 19001 || cfg.Address != "203.0.113.10" {
		t.Errorf("unexpected network: %+v", cfg)
	}
	if cfg.BandwidthRate != "50 MBytes" || cfg.BandwidthBurst != "100 MBytes" {
		t.Errorf("unexpected bandwidth: %+v", cfg)
	}
	if cfg.Sandbox || cfg.LogLevel != "debug" {
		t.Errorf("unexpected flags: %+v", cfg)
	}
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load([]string{
		"TOR_ROLE=middle",
		"TOR_NICKNAME=TestRelay",
		"TOR_CONTACT=test@example.com",
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ORPort != 9001 || !cfg.Sandbox || cfg.LogLevel != "notice" {
		t.Errorf("unexpected defaults: %+v", cfg)
	}
	if cfg.FamilyKeyDir != "/run/secrets" {
		t.Errorf("FamilyKeyDir = %q, want /run/secrets", cfg.FamilyKeyDir)
	}
	if len(cfg.FamilyID) != 0 {
		t.Errorf("FamilyID = %v, want empty", cfg.FamilyID)
	}
	if len(cfg.MyFamily) != 0 {
		t.Errorf("MyFamily = %v, want empty", cfg.MyFamily)
	}
	if cfg.MyFamilyURL != "" {
		t.Errorf("MyFamilyURL = %q, want empty", cfg.MyFamilyURL)
	}
	if cfg.MyFamilyRefresh != time.Hour {
		t.Errorf("MyFamilyRefresh = %s, want 1h", cfg.MyFamilyRefresh)
	}
	if cfg.MyFamilyMaxDropPct != 20 {
		t.Errorf("MyFamilyMaxDropPct = %d, want 20", cfg.MyFamilyMaxDropPct)
	}
}

func TestLoadFamily(t *testing.T) {
	cfg, err := Load([]string{
		"TOR_ROLE=middle",
		"TOR_NICKNAME=TestRelay",
		"TOR_CONTACT=test@example.com",
		"TOR_FAMILY_ID=id1 id2",
		"TOR_FAMILY_KEY_DIR=/run/secrets",
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.FamilyID) != 2 || cfg.FamilyID[0] != "id1" || cfg.FamilyID[1] != "id2" {
		t.Errorf("FamilyID = %v, want [id1 id2]", cfg.FamilyID)
	}
	if cfg.FamilyKeyDir != "/run/secrets" {
		t.Errorf("FamilyKeyDir = %q, want /run/secrets", cfg.FamilyKeyDir)
	}
}

func TestLoadMyFamily(t *testing.T) {
	cfg, err := Load([]string{
		"TOR_ROLE=middle",
		"TOR_NICKNAME=TestRelay",
		"TOR_CONTACT=test@example.com",
		"TOR_MYFAMILY=$0123456789abcdef0123456789abcdef01234567,89ABCDEF0123456789ABCDEF0123456789ABCDEF",
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := []string{
		"0123456789ABCDEF0123456789ABCDEF01234567",
		"89ABCDEF0123456789ABCDEF0123456789ABCDEF",
	}
	if len(cfg.MyFamily) != 2 || cfg.MyFamily[0] != want[0] || cfg.MyFamily[1] != want[1] {
		t.Errorf("MyFamily = %v, want %v", cfg.MyFamily, want)
	}
	if cfg.MyFamilyURL != "" {
		t.Errorf("MyFamilyURL = %q, want empty", cfg.MyFamilyURL)
	}
}

func TestLoadMyFamilyURL(t *testing.T) {
	cfg, err := Load([]string{
		"TOR_ROLE=middle",
		"TOR_NICKNAME=TestRelay",
		"TOR_CONTACT=test@example.com",
		"TOR_MYFAMILY_URL=https://example.com/family.txt",
		"TOR_MYFAMILY_REFRESH=30m",
		"TOR_MYFAMILY_MAX_DROP_PCT=0",
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.MyFamilyURL != "https://example.com/family.txt" {
		t.Errorf("MyFamilyURL = %q", cfg.MyFamilyURL)
	}
	if len(cfg.MyFamily) != 0 {
		t.Errorf("MyFamily = %v, want empty", cfg.MyFamily)
	}
	if cfg.MyFamilyRefresh != 30*time.Minute {
		t.Errorf("MyFamilyRefresh = %s, want 30m", cfg.MyFamilyRefresh)
	}
	if cfg.MyFamilyMaxDropPct != 0 {
		t.Errorf("MyFamilyMaxDropPct = %d, want 0", cfg.MyFamilyMaxDropPct)
	}
}

func TestLoadExit(t *testing.T) {
	cfg, err := Load([]string{
		"TOR_ROLE=exit",
		"TOR_NICKNAME=TestRelay",
		"TOR_CONTACT=test@example.com",
		"I_UNDERSTAND_EXIT=true",
		"TOR_EXIT_POLICY=accept 80;accept 443;reject *:*",
		"TOR_EXIT_IPV6=1",
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Role != RoleExit {
		t.Errorf("Role = %q, want exit", cfg.Role)
	}
	want := []string{"accept 80", "accept 443", "reject *:*"}
	if len(cfg.ExitPolicy) != len(want) {
		t.Fatalf("ExitPolicy = %v, want %v", cfg.ExitPolicy, want)
	}
	for i := range want {
		if cfg.ExitPolicy[i] != want[i] {
			t.Errorf("ExitPolicy[%d] = %q, want %q", i, cfg.ExitPolicy[i], want[i])
		}
	}
	if !cfg.ExitIPv6 {
		t.Error("ExitIPv6 = false, want true")
	}

	defaults, err := Load([]string{
		"TOR_ROLE=exit",
		"TOR_NICKNAME=TestRelay",
		"TOR_CONTACT=test@example.com",
		"I_UNDERSTAND_EXIT=true",
		"TOR_EXIT_POLICY=accept 443",
	})
	if err != nil {
		t.Fatalf("Load defaults: %v", err)
	}
	if defaults.ExitIPv6 {
		t.Error("ExitIPv6 = true, want false by default")
	}
}

func TestValidateExitPolicy(t *testing.T) {
	dir := t.TempDir()

	withEnv := &Config{Role: RoleExit, ExitPolicy: []string{"accept 443"}}
	if err := ValidateExitPolicy(withEnv, dir); err != nil {
		t.Fatalf("env policy: %v", err)
	}

	noPolicy := &Config{Role: RoleExit}
	if err := ValidateExitPolicy(noPolicy, dir); err == nil {
		t.Fatal("want error for an exit relay without a policy")
	} else if !strings.Contains(err.Error(), "requires TOR_EXIT_POLICY") {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "exit.conf"), []byte("# comment\nExitPolicy accept 80\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateExitPolicy(noPolicy, dir); err != nil {
		t.Fatalf("torrc.d policy: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "exit.conf"), []byte("# ExitPolicy accept 80\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateExitPolicy(noPolicy, dir); err == nil {
		t.Fatal("commented ExitPolicy should not count")
	}

	if err := os.WriteFile(filepath.Join(dir, "exit.conf"), []byte("ExitPolicy\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateExitPolicy(noPolicy, dir); err == nil {
		t.Fatal("ExitPolicy without a value should not count")
	}

	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("ExitPolicy accept 80\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateExitPolicy(noPolicy, dir); err == nil {
		t.Fatal("only *.conf files should count")
	}

	middle := &Config{Role: RoleMiddle}
	if err := ValidateExitPolicy(middle, dir); err != nil {
		t.Fatalf("middle relay: %v", err)
	}
}

func TestLoadRejects(t *testing.T) {
	base := []string{
		"TOR_ROLE=middle",
		"TOR_NICKNAME=TestRelay",
		"TOR_CONTACT=test@example.com",
	}

	tests := []struct {
		name string
		env  []string
		want string
	}{
		{"missing role", []string{"TOR_NICKNAME=TestRelay", "TOR_CONTACT=test@example.com"}, "TOR_ROLE is required"},
		{"bad role", append(base[:0:0], "TOR_ROLE=bridge", "TOR_NICKNAME=TestRelay", "TOR_CONTACT=test@example.com"), `TOR_ROLE "bridge" is not supported`},
		{"missing nickname", []string{"TOR_ROLE=middle", "TOR_CONTACT=test@example.com"}, "TOR_NICKNAME is required"},
		{"empty nickname", []string{"TOR_ROLE=middle", "TOR_NICKNAME=", "TOR_CONTACT=test@example.com"}, "TOR_NICKNAME is required"},
		{"long nickname", []string{"TOR_ROLE=middle", "TOR_NICKNAME=12345678901234567890", "TOR_CONTACT=test@example.com"}, "must match"},
		{"bad nickname", []string{"TOR_ROLE=middle", "TOR_NICKNAME=bad-name", "TOR_CONTACT=test@example.com"}, "must match"},
		{"missing contact", []string{"TOR_ROLE=middle", "TOR_NICKNAME=TestRelay"}, "TOR_CONTACT is required"},
		{"multiline contact", []string{"TOR_ROLE=middle", "TOR_NICKNAME=TestRelay", "TOR_CONTACT=line1\nline2"}, "single line"},
		{"orport zero", append(base[:0:0], base...), "TOR_ORPORT 0 must be between 1 and 65535"},
		{"orport high", append(base[:0:0], base...), "TOR_ORPORT 65536 must be between 1 and 65535"},
		{"orport text", append(base[:0:0], base...), "TOR_ORPORT \"http\" is not a number"},
		{"bad sandbox", append(base[:0:0], base...), `TOR_SANDBOX "yes" must be 0 or 1`},
		{"bad log level", append(base[:0:0], base...), `TOR_LOG_LEVEL "verbose" must be one of`},
		{"unknown variable", append(base[:0:0], base...), "unknown environment variable TOR_NICKNMAE"},
		{
			"bad family id",
			[]string{"TOR_ROLE=middle", "TOR_NICKNAME=TestRelay", "TOR_CONTACT=test@example.com", "TOR_FAMILY_ID=bad!id"},
			`TOR_FAMILY_ID "bad!id" must be base64-like`,
		},
		{
			"multiline family key dir",
			[]string{"TOR_ROLE=middle", "TOR_NICKNAME=TestRelay", "TOR_CONTACT=test@example.com", "TOR_FAMILY_KEY_DIR=/run\nsecrets"},
			"TOR_FAMILY_KEY_DIR must be a single line",
		},
		{
			"bad myfamily fingerprint",
			[]string{"TOR_ROLE=middle", "TOR_NICKNAME=TestRelay", "TOR_CONTACT=test@example.com", "TOR_MYFAMILY=$0123456789ABCDEF"},
			`TOR_MYFAMILY entry "0123456789ABCDEF" must be a 40-hex fingerprint`,
		},
		{
			"myfamily and url together",
			[]string{"TOR_ROLE=middle", "TOR_NICKNAME=TestRelay", "TOR_CONTACT=test@example.com", "TOR_MYFAMILY=0123456789ABCDEF0123456789ABCDEF01234567", "TOR_MYFAMILY_URL=https://example.com/family.txt"},
			"TOR_MYFAMILY and TOR_MYFAMILY_URL cannot both be set",
		},
		{
			"http myfamily url",
			[]string{"TOR_ROLE=middle", "TOR_NICKNAME=TestRelay", "TOR_CONTACT=test@example.com", "TOR_MYFAMILY_URL=http://example.com/family.txt"},
			"must be an https URL with a host",
		},
		{
			"myfamily url without host",
			[]string{"TOR_ROLE=middle", "TOR_NICKNAME=TestRelay", "TOR_CONTACT=test@example.com", "TOR_MYFAMILY_URL=https:///family.txt"},
			"must be an https URL with a host",
		},
		{
			"myfamily refresh too short",
			[]string{"TOR_ROLE=middle", "TOR_NICKNAME=TestRelay", "TOR_CONTACT=test@example.com", "TOR_MYFAMILY_URL=https://example.com/family.txt", "TOR_MYFAMILY_REFRESH=5m"},
			"TOR_MYFAMILY_REFRESH 5m0s must be at least 10m",
		},
		{
			"myfamily refresh not a duration",
			[]string{"TOR_ROLE=middle", "TOR_NICKNAME=TestRelay", "TOR_CONTACT=test@example.com", "TOR_MYFAMILY_URL=https://example.com/family.txt", "TOR_MYFAMILY_REFRESH=soon"},
			`TOR_MYFAMILY_REFRESH "soon" is not a duration`,
		},
		{
			"myfamily drop pct high",
			[]string{"TOR_ROLE=middle", "TOR_NICKNAME=TestRelay", "TOR_CONTACT=test@example.com", "TOR_MYFAMILY_URL=https://example.com/family.txt", "TOR_MYFAMILY_MAX_DROP_PCT=101"},
			"TOR_MYFAMILY_MAX_DROP_PCT 101 must be between 0 and 100",
		},
		{
			"myfamily drop pct text",
			[]string{"TOR_ROLE=middle", "TOR_NICKNAME=TestRelay", "TOR_CONTACT=test@example.com", "TOR_MYFAMILY_URL=https://example.com/family.txt", "TOR_MYFAMILY_MAX_DROP_PCT=lots"},
			`TOR_MYFAMILY_MAX_DROP_PCT "lots" is not a number`,
		},
		{
			"exit without ack",
			[]string{"TOR_ROLE=exit", "TOR_NICKNAME=TestRelay", "TOR_CONTACT=test@example.com", "TOR_EXIT_POLICY=accept 443"},
			"TOR_ROLE=exit requires I_UNDERSTAND_EXIT=true",
		},
		{
			"exit with wrong ack",
			[]string{"TOR_ROLE=exit", "TOR_NICKNAME=TestRelay", "TOR_CONTACT=test@example.com", "I_UNDERSTAND_EXIT=1", "TOR_EXIT_POLICY=accept 443"},
			"TOR_ROLE=exit requires I_UNDERSTAND_EXIT=true",
		},
		{
			"exit bad ipv6",
			[]string{"TOR_ROLE=exit", "TOR_NICKNAME=TestRelay", "TOR_CONTACT=test@example.com", "I_UNDERSTAND_EXIT=true", "TOR_EXIT_POLICY=accept 443", "TOR_EXIT_IPV6=yes"},
			`TOR_EXIT_IPV6 "yes" must be 0 or 1`,
		},
		{
			"exit empty policy entry",
			[]string{"TOR_ROLE=exit", "TOR_NICKNAME=TestRelay", "TOR_CONTACT=test@example.com", "I_UNDERSTAND_EXIT=true", "TOR_EXIT_POLICY=accept 80;;reject *:*"},
			"TOR_EXIT_POLICY contains an empty entry",
		},
	}

	// Fix up cases that need extra entries appended to the base.
	tests[8].env = append(tests[8].env, "TOR_ORPORT=0")
	tests[9].env = append(tests[9].env, "TOR_ORPORT=65536")
	tests[10].env = append(tests[10].env, "TOR_ORPORT=http")
	tests[11].env = append(tests[11].env, "TOR_SANDBOX=yes")
	tests[12].env = append(tests[12].env, "TOR_LOG_LEVEL=verbose")
	tests[13].env = append(tests[13].env, "TOR_NICKNMAE=TestRelay")

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Load(tc.env)
			if err == nil {
				t.Fatal("Load succeeded, want error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Load error = %q, want substring %q", err, tc.want)
			}
		})
	}
}
