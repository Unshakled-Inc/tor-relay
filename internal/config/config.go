package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const RoleMiddle = "middle"

var nicknameRE = regexp.MustCompile(`^[A-Za-z0-9]{1,19}$`)

var allowed = map[string]bool{
	"TOR_ROLE":            true,
	"TOR_NICKNAME":        true,
	"TOR_CONTACT":         true,
	"TOR_ORPORT":          true,
	"TOR_ADDRESS":         true,
	"TOR_BANDWIDTH_RATE":  true,
	"TOR_BANDWIDTH_BURST": true,
	"TOR_SANDBOX":         true,
	"TOR_LOG_LEVEL":       true,
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
	Role           string
	Nickname       string
	Contact        string
	ORPort         int
	Address        string
	BandwidthRate  string
	BandwidthBurst string
	Sandbox        bool
	LogLevel       string
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
	if role != RoleMiddle {
		return nil, fmt.Errorf("TOR_ROLE %q is not supported in v0, only %q", role, RoleMiddle)
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

	return &Config{
		Role:           role,
		Nickname:       nickname,
		Contact:        contact,
		ORPort:         port,
		Address:        address,
		BandwidthRate:  rate,
		BandwidthBurst: burst,
		Sandbox:        sandbox,
		LogLevel:       logLevel,
	}, nil
}
