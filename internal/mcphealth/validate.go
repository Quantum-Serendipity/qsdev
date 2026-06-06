package mcphealth

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var envRefPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// ValidateConfig checks server configurations without starting any processes.
func ValidateConfig(servers map[string]ServerConfig) []ConfigWarning {
	var warnings []ConfigWarning

	envSet := buildEnvSet()

	for name, cfg := range servers {
		if name == "" {
			warnings = append(warnings, ConfigWarning{
				Server:   "(empty)",
				Severity: "error",
				Message:  "server name is empty",
			})
			continue
		}

		warnings = append(warnings, validateCommand(name, cfg.Command)...)
		warnings = append(warnings, validateRequiredEnv(name, cfg.RequiredEnv, envSet)...)
		warnings = append(warnings, validateEnvRefs(name, cfg.Env, envSet)...)
	}

	return warnings
}

func validateCommand(server, command string) []ConfigWarning {
	if command == "" {
		return []ConfigWarning{{
			Server:      server,
			Severity:    "error",
			Message:     "command is empty",
			Remediation: "specify a command binary for this MCP server",
		}}
	}

	if _, err := exec.LookPath(command); err != nil {
		return []ConfigWarning{{
			Server:      server,
			Severity:    "error",
			Message:     fmt.Sprintf("command %q not found on PATH", command),
			Remediation: fmt.Sprintf("install %q or add it to PATH", command),
		}}
	}

	return nil
}

func validateRequiredEnv(server string, required []string, envSet map[string]bool) []ConfigWarning {
	var warnings []ConfigWarning
	for _, key := range required {
		if !envSet[key] {
			warnings = append(warnings, ConfigWarning{
				Server:      server,
				Severity:    "warning",
				Message:     fmt.Sprintf("required environment variable %q is not set", key),
				Remediation: fmt.Sprintf("set %s in your environment or .env file", key),
			})
		}
	}
	return warnings
}

func validateEnvRefs(server string, env map[string]string, envSet map[string]bool) []ConfigWarning {
	var warnings []ConfigWarning
	for key, val := range env {
		matches := envRefPattern.FindAllStringSubmatch(val, -1)
		for _, match := range matches {
			refVar := match[1]
			if !envSet[refVar] {
				warnings = append(warnings, ConfigWarning{
					Server:      server,
					Severity:    "warning",
					Message:     fmt.Sprintf("env %q references ${%s} which is not set", key, refVar),
					Remediation: fmt.Sprintf("set %s in your environment or .env file", refVar),
				})
			}
		}
	}
	return warnings
}

func buildEnvSet() map[string]bool {
	set := make(map[string]bool)
	for _, entry := range os.Environ() {
		if k, _, ok := strings.Cut(entry, "="); ok {
			set[k] = true
		}
	}
	return set
}
