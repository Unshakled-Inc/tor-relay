package config

import (
	"strings"
	"testing"
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
		{"bad role", append(base[:0:0], "TOR_ROLE=exit", "TOR_NICKNAME=TestRelay", "TOR_CONTACT=test@example.com"), `TOR_ROLE "exit" is not supported in v0`},
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
