package mcphealth

import (
	"fmt"
	"maps"
	"net"
	"net/url"
	"os"
	"os/exec"
	"slices"
	"strings"
)

// ValidateConfig checks server configurations without starting any processes.
func ValidateConfig(servers map[string]ServerConfig) []ConfigWarning {
	var warnings []ConfigWarning

	envSet := buildEnvSet()

	for name, cfg := range servers {
		if name == "" {
			warnings = append(warnings, ConfigWarning{
				Server:   "(empty)",
				Severity: SeverityError,
				Message:  "server name is empty",
			})
			continue
		}

		// A remote (Streamable HTTP/SSE) server has a URL instead of a command.
		if cfg.URL != "" {
			warnings = append(warnings, validateURL(name, expandVars(cfg.URL, os.LookupEnv))...)
		} else {
			warnings = append(warnings, validateCommand(name, expandVars(cfg.Command, os.LookupEnv))...)
		}
		warnings = append(warnings, validateRequiredEnv(name, cfg.RequiredEnv, envSet)...)
		warnings = append(warnings, validateEnvRefs(name, cfg, envSet)...)
	}

	return warnings
}

func validateCommand(server, command string) []ConfigWarning {
	if command == "" {
		return []ConfigWarning{{
			Server:      server,
			Severity:    SeverityError,
			Message:     "command is empty",
			Remediation: "specify a command binary for this MCP server",
		}}
	}

	if _, err := exec.LookPath(command); err != nil {
		return []ConfigWarning{{
			Server:      server,
			Severity:    SeverityError,
			Message:     fmt.Sprintf("command %q not found on PATH", command),
			Remediation: fmt.Sprintf("install %q or add it to PATH", command),
		}}
	}

	return nil
}

// validateURL checks a remote server URL: it must be absolute with a host, and
// use https, or plain http only for a loopback host.
func validateURL(server, raw string) []ConfigWarning {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return []ConfigWarning{{
			Server:      server,
			Severity:    SeverityError,
			Message:     fmt.Sprintf("url %q is not a valid http(s) URL", raw),
			Remediation: "specify an absolute https:// URL for this MCP server",
		}}
	}
	if u.Scheme == "http" && !isLoopbackHost(u.Hostname()) {
		return []ConfigWarning{{
			Server:      server,
			Severity:    SeverityError,
			Message:     fmt.Sprintf("url %q uses plain http to a non-local host", raw),
			Remediation: "use https:// (plain http is only acceptable for localhost)",
		}}
	}
	return nil
}

// isLoopbackHost reports whether host is localhost or a loopback IP address.
func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func validateRequiredEnv(server string, required []string, envSet map[string]bool) []ConfigWarning {
	var warnings []ConfigWarning
	for _, key := range required {
		if !envSet[key] {
			warnings = append(warnings, ConfigWarning{
				Server:      server,
				Severity:    SeverityWarning,
				Message:     fmt.Sprintf("required environment variable %q is not set", key),
				Remediation: fmt.Sprintf("set %s in your environment or .env file", key),
			})
		}
	}
	return warnings
}

// validateEnvRefs warns about ${VAR} references to unset variables in every
// field that is expanded (command, args, url, env and header values). A
// ${VAR:-default} reference always resolves, so it is never reported.
func validateEnvRefs(server string, cfg ServerConfig, envSet map[string]bool) []ConfigWarning {
	type field struct{ label, value string }
	fields := []field{{"command", cfg.Command}, {"url", cfg.URL}}
	for i, a := range cfg.Args {
		fields = append(fields, field{fmt.Sprintf("arg %d", i), a})
	}
	for _, k := range slices.Sorted(maps.Keys(cfg.Env)) {
		fields = append(fields, field{fmt.Sprintf("env %q", k), cfg.Env[k]})
	}
	for _, k := range slices.Sorted(maps.Keys(cfg.Headers)) {
		fields = append(fields, field{fmt.Sprintf("header %q", k), cfg.Headers[k]})
	}

	var warnings []ConfigWarning
	for _, f := range fields {
		for _, match := range envRefPattern.FindAllStringSubmatch(f.value, -1) {
			refVar, hasDefault := match[1], match[2] != ""
			if hasDefault || envSet[refVar] {
				continue
			}
			warnings = append(warnings, ConfigWarning{
				Server:      server,
				Severity:    SeverityWarning,
				Message:     fmt.Sprintf("%s references ${%s} which is not set", f.label, refVar),
				Remediation: fmt.Sprintf("set %s in your environment or .env file", refVar),
			})
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
