package devenv

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// cachixNameRe matches a Cachix cache name (the host label before .cachix.org).
var cachixNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// nixPublicKeyRe matches a Nix binary cache public key ("name:base64").
var nixPublicKeyRe = regexp.MustCompile(`^[A-Za-z0-9._-]+:[A-Za-z0-9+/]+={0,2}$`)

// projectNixCache returns the project's binary cache (infrastructure.nix_cache
// with its public key), or false when none is configured or the values are
// not a well-formed https URL and key. An explicit infra profile has already
// validated them (profile.InfraProfile.Resolve); the check here keeps a
// hand-edited .qsdev.yaml from reaching the rendered Nix and nix.conf text.
func projectNixCache(infra types.InfraConfig) (CacheEntry, bool) {
	raw, key := infra.NixCacheURL(), infra.NixCachePublicKey
	if raw == "" || !nixPublicKeyRe.MatchString(key) {
		return CacheEntry{}, false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || strings.ContainsAny(raw, " \t\n\"'\\$") {
		return CacheEntry{}, false
	}
	return CacheEntry{
		Name:      u.Host,
		URL:       raw,
		PublicKey: key,
		Purpose:   "Project binary cache (infrastructure.nix_cache)",
	}, true
}

// cachixPullCaches returns the Cachix caches devenv should pull from: the
// project cache when it is hosted on cachix.org. devenv's cachix module adds
// it as a substituter for the shell; the Nix daemon honors it only when the
// system trusts it (docs/nix-conf-hardening.md).
func cachixPullCaches(infra types.InfraConfig) []string {
	c, ok := projectNixCache(infra)
	if !ok {
		return nil
	}
	name, found := strings.CutSuffix(strings.ToLower(c.Name), ".cachix.org")
	if !found || !cachixNameRe.MatchString(name) {
		return nil
	}
	return []string{name}
}
