package profile

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
)

var (
	// ErrPlaceholderEndpoint reports an example or placeholder endpoint
	// (example.com, the "myorg" Cachix cache, an all-zero signing key) where
	// the organization's real value is required.
	ErrPlaceholderEndpoint = errors.New("placeholder endpoint")
	// ErrInvalidEndpoint reports an endpoint that is malformed or insecure.
	ErrInvalidEndpoint = errors.New("invalid endpoint")
	// ErrEndpointNotConfigured reports an endpoint an explicitly selected
	// infra profile needs but the project does not configure.
	ErrEndpointNotConfigured = errors.New("endpoint not configured")
)

// placeholderCacheNames are Cachix cache names used as examples; a profile
// pointing at one would trust (and pull from) a cache the user does not own.
var placeholderCacheNames = map[string]bool{"myorg": true, "my-org": true, "example": true}

// cachixNameRe matches a Cachix cache name.
var cachixNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// checkEndpointURL validates a user-supplied endpoint URL: absolute, https
// (plain http only to a loopback host), without embedded credentials, and
// not an example/placeholder host.
func checkEndpointURL(field, raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return fmt.Errorf("%w: %s %q is not an absolute http(s) URL", ErrInvalidEndpoint, field, raw)
	}
	host := strings.ToLower(u.Hostname())
	if u.Scheme == "http" && !isLoopbackHost(host) {
		return fmt.Errorf("%w: %s %q uses plain http to a non-local host; use https", ErrInvalidEndpoint, field, raw)
	}
	if u.User != nil {
		return fmt.Errorf("%w: %s must not embed credentials; supply them through the environment", ErrInvalidEndpoint, field)
	}
	if isPlaceholderHost(host) {
		return fmt.Errorf("%w: %s %q is an example host; set it to your organization's real endpoint", ErrPlaceholderEndpoint, field, raw)
	}
	if name, ok := strings.CutSuffix(host, ".cachix.org"); ok && placeholderCacheNames[name] {
		return fmt.Errorf("%w: %s %q is the example Cachix cache; set it to your own cache", ErrPlaceholderEndpoint, field, raw)
	}
	return nil
}

// isPlaceholderHost reports whether host is reserved for documentation
// (RFC 2606/6761): example.com/.net/.org and the .example, .invalid and .test
// top-level domains.
func isPlaceholderHost(host string) bool {
	host = strings.TrimSuffix(host, ".")
	for _, d := range []string{"example.com", "example.net", "example.org"} {
		if host == d || strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	for _, tld := range []string{".example", ".invalid", ".test"} {
		if strings.HasSuffix(host, tld) {
			return true
		}
	}
	return false
}

// isLoopbackHost reports whether host is localhost or a loopback IP address.
func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// checkNixPublicKey validates a Nix binary cache public key: "name:base64"
// with a 32-byte Ed25519 key that is not the all-zero placeholder.
func checkNixPublicKey(field, key string) error {
	name, b64, ok := strings.Cut(key, ":")
	if !ok || name == "" || strings.ContainsAny(key, " \t\n\"") {
		return fmt.Errorf("%w: %s %q is not a Nix public key (name:base64)", ErrInvalidEndpoint, field, key)
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil || len(raw) != 32 {
		return fmt.Errorf("%w: %s %q does not hold a 32-byte Ed25519 key", ErrInvalidEndpoint, field, key)
	}
	for _, b := range raw {
		if b != 0 {
			return nil
		}
	}
	return fmt.Errorf("%w: %s is an all-zero placeholder key; set it to your cache's real public key", ErrPlaceholderEndpoint, field)
}
