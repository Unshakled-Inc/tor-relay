package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	RoleMiddle = "middle"
	RoleExit   = "exit"
)

var nicknameRE = regexp.MustCompile(`^[A-Za-z0-9]{1,19}$`)
var familyIDRE = regexp.MustCompile(`^[A-Za-z0-9+/]+={0,2}$`)
var fingerprintRE = regexp.MustCompile(`^[0-9A-Fa-f]{40}$`)

var allowed = map[string]bool{
	"TOR_ROLE":                  true,
	"TOR_NICKNAME":              true,
	"TOR_CONTACT":               true,
	"TOR_ORPORT":                true,
	"TOR_ADDRESS":               true,
	"TOR_BANDWIDTH_RATE":        true,
	"TOR_BANDWIDTH_BURST":       true,
	"TOR_SANDBOX":               true,
	"TOR_LOG_LEVEL":             true,
	"TOR_FAMILY_ID":             true,
	"TOR_FAMILY_KEY_DIR":        true,
	"TOR_MYFAMILY":              true,
	"TOR_MYFAMILY_URL":          true,
	"TOR_MYFAMILY_REFRESH":      true,
	"TOR_MYFAMILY_MAX_DROP_PCT": true,
	"TOR_EXIT_POLICY":           true,
	"TOR_EXIT_IPV6":             true,
}

var logLevels = map[string]bool{
	"debug":  true,
	"info":   true,
	"notice": true,
	"warn":   true,
	"err":    true,
}

// Config is the validated torinit configuration for v0.
type Config struct {
	Role               string
	Nickname           string
	Contact            string
	ORPort             int
	Address            string
	BandwidthRate      string
	BandwidthBurst     string
	Sandbox            bool
	LogLevel           string
	FamilyID           []string
	FamilyKeyDir       string
	MyFamily           []string
	MyFamilyURL        string
	MyFamilyRefresh    time.Duration
	MyFamilyMaxDropPct int
	ExitPolicy         []string
	ExitIPv6           bool
}

// Load parses and validates a list of KEY=VALUE environment entries.
// Any TOR_ variable that is not in the v0 set is rejected, so typos fail
// loudly instead of being silently ignored.
func Load(environ []string) (*Config, error) {
	values := make(map[string]string)
	for _, entry := range environ {
		name, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if strings.HasPrefix(name, "TOR_") && !allowed[name] {
			return nil, fmt.Errorf("unknown environment variable %s", name)
		}
		values[name] = value
	}

	role := values["TOR_ROLE"]
	if role == "" {
		return nil, fmt.Errorf("TOR_ROLE is required")
	}
	switch role {
	case RoleMiddle, RoleExit:
	default:
		return nil, fmt.Errorf("TOR_ROLE %q is not supported, use %q or %q", role, RoleMiddle, RoleExit)
	}
	if role == RoleExit && values["I_UNDERSTAND_EXIT"] != "true" {
		return nil, fmt.Errorf("TOR_ROLE=exit requires I_UNDERSTAND_EXIT=true")
	}

	nickname := values["TOR_NICKNAME"]
	if nickname == "" {
		return nil, fmt.Errorf("TOR_NICKNAME is required")
	}
	if !nicknameRE.MatchString(nickname) {
		return nil, fmt.Errorf("TOR_NICKNAME %q must match %s", nickname, nicknameRE.String())
	}

	contact := values["TOR_CONTACT"]
	if contact == "" {
		return nil, fmt.Errorf("TOR_CONTACT is required")
	}
	if strings.ContainsAny(contact, "\r\n") {
		return nil, fmt.Errorf("TOR_CONTACT must be a single line")
	}

	port := 9001
	if value := values["TOR_ORPORT"]; value != "" {
		n, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("TOR_ORPORT %q is not a number", value)
		}
		if n < 1 || n > 65535 {
			return nil, fmt.Errorf("TOR_ORPORT %d must be between 1 and 65535", n)
		}
		port = n
	}

	address := values["TOR_ADDRESS"]
	if strings.ContainsAny(address, "\r\n") {
		return nil, fmt.Errorf("TOR_ADDRESS must be a single line")
	}

	rate := values["TOR_BANDWIDTH_RATE"]
	if strings.ContainsAny(rate, "\r\n") {
		return nil, fmt.Errorf("TOR_BANDWIDTH_RATE must be a single line")
	}

	burst := values["TOR_BANDWIDTH_BURST"]
	if strings.ContainsAny(burst, "\r\n") {
		return nil, fmt.Errorf("TOR_BANDWIDTH_BURST must be a single line")
	}

	sandbox := true
	if value := values["TOR_SANDBOX"]; value != "" {
		switch value {
		case "0":
			sandbox = false
		case "1":
			sandbox = true
		default:
			return nil, fmt.Errorf("TOR_SANDBOX %q must be 0 or 1", value)
		}
	}

	logLevel := "notice"
	if value := values["TOR_LOG_LEVEL"]; value != "" {
		if !logLevels[value] {
			return nil, fmt.Errorf("TOR_LOG_LEVEL %q must be one of debug, info, notice, warn, err", value)
		}
		logLevel = value
	}

	familyID := []string{}
	if value := values["TOR_FAMILY_ID"]; value != "" {
		for _, id := range strings.Fields(value) {
			if !familyIDRE.MatchString(id) {
				return nil, fmt.Errorf("TOR_FAMILY_ID %q must be base64-like with no whitespace", id)
			}
			familyID = append(familyID, id)
		}
	}

	familyKeyDir := "/run/secrets"
	if value := values["TOR_FAMILY_KEY_DIR"]; value != "" {
		if strings.ContainsAny(value, "\r\n") {
			return nil, fmt.Errorf("TOR_FAMILY_KEY_DIR must be a single line")
		}
		familyKeyDir = value
	}

	myFamily := []string{}
	if value := values["TOR_MYFAMILY"]; value != "" {
		if strings.ContainsAny(value, "\r\n") {
			return nil, fmt.Errorf("TOR_MYFAMILY must be a single line")
		}
		for _, entry := range strings.Split(value, ",") {
			entry = strings.TrimSpace(entry)
			entry = strings.TrimPrefix(entry, "$")
			if !fingerprintRE.MatchString(entry) {
				return nil, fmt.Errorf("TOR_MYFAMILY entry %q must be a 40-hex fingerprint", entry)
			}
			myFamily = append(myFamily, strings.ToUpper(entry))
		}
	}

	myFamilyURL := values["TOR_MYFAMILY_URL"]
	if myFamilyURL != "" {
		if strings.ContainsAny(myFamilyURL, "\r\n") {
			return nil, fmt.Errorf("TOR_MYFAMILY_URL must be a single line")
		}
		parsed, err := url.Parse(myFamilyURL)
		if err != nil {
			return nil, fmt.Errorf("TOR_MYFAMILY_URL %q is not a URL: %v", myFamilyURL, err)
		}
		if parsed.Scheme != "https" || parsed.Host == "" {
			return nil, fmt.Errorf("TOR_MYFAMILY_URL %q must be an https URL with a host", myFamilyURL)
		}
	}
	if len(myFamily) > 0 && myFamilyURL != "" {
		return nil, fmt.Errorf("TOR_MYFAMILY and TOR_MYFAMILY_URL cannot both be set")
	}

	myFamilyRefresh := time.Hour
	if value := values["TOR_MYFAMILY_REFRESH"]; value != "" {
		d, err := time.ParseDuration(value)
		if err != nil {
			return nil, fmt.Errorf("TOR_MYFAMILY_REFRESH %q is not a duration", value)
		}
		if d < 10*time.Minute {
			return nil, fmt.Errorf("TOR_MYFAMILY_REFRESH %s must be at least 10m", d)
		}
		myFamilyRefresh = d
	}

	myFamilyMaxDropPct := 20
	if value := values["TOR_MYFAMILY_MAX_DROP_PCT"]; value != "" {
		n, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("TOR_MYFAMILY_MAX_DROP_PCT %q is not a number", value)
		}
		if n < 0 || n > 100 {
			return nil, fmt.Errorf("TOR_MYFAMILY_MAX_DROP_PCT %d must be between 0 and 100", n)
		}
		myFamilyMaxDropPct = n
	}

	exitPolicy := []string{}
	if value := values["TOR_EXIT_POLICY"]; value != "" {
		if strings.ContainsAny(value, "\r\n") {
			return nil, fmt.Errorf("TOR_EXIT_POLICY must be a single line")
		}
		for _, policy := range strings.Split(value, ";") {
			policy = strings.TrimSpace(policy)
			if policy == "" {
				return nil, fmt.Errorf("TOR_EXIT_POLICY contains an empty entry")
			}
			exitPolicy = append(exitPolicy, policy)
		}
	}

	exitIPv6 := false
	if value := values["TOR_EXIT_IPV6"]; value != "" {
		switch value {
		case "0":
			exitIPv6 = false
		case "1":
			exitIPv6 = true
		default:
			return nil, fmt.Errorf("TOR_EXIT_IPV6 %q must be 0 or 1", value)
		}
	}

	return &Config{
		Role:               role,
		Nickname:           nickname,
		Contact:            contact,
		ORPort:             port,
		Address:            address,
		BandwidthRate:      rate,
		BandwidthBurst:     burst,
		Sandbox:            sandbox,
		LogLevel:           logLevel,
		FamilyID:           familyID,
		FamilyKeyDir:       familyKeyDir,
		MyFamily:           myFamily,
		MyFamilyURL:        myFamilyURL,
		MyFamilyRefresh:    myFamilyRefresh,
		MyFamilyMaxDropPct: myFamilyMaxDropPct,
		ExitPolicy:         exitPolicy,
		ExitIPv6:           exitIPv6,
	}, nil
}

// ValidateExitPolicy requires an exit relay to have an exit policy, either
// from TOR_EXIT_POLICY or from an ExitPolicy line in a *.conf file under
// torrcDDir. Middle relays are not checked.
func ValidateExitPolicy(cfg *Config, torrcDDir string) error {
	if cfg.Role != RoleExit {
		return nil
	}
	if len(cfg.ExitPolicy) > 0 {
		return nil
	}
	if hasExitPolicy(torrcDDir) {
		return nil
	}
	return fmt.Errorf("TOR_ROLE=exit requires TOR_EXIT_POLICY or an ExitPolicy line in %s/*.conf", torrcDDir)
}

func hasExitPolicy(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".conf") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "ExitPolicy") {
				continue
			}
			if len(strings.Fields(line)) >= 2 {
				return true
			}
		}
	}
	return false
}
