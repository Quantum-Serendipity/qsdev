package profile

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Resolve returns a copy of the profile with the project's infrastructure
// endpoints (.qsdev.yaml infrastructure:) applied, for an explicitly
// selected infra profile. It fails when an endpoint the profile needs is
// missing, malformed or a placeholder, so selecting a profile never silently
// leaves package installs on the public registries:
//
//   - a pull-through registry proxy needs infrastructure.registry_proxy (or a
//     per-ecosystem override) for each proxied ecosystem the project uses;
//   - a Nix binary cache needs infrastructure.nix_cache and
//     infrastructure.nix_cache_public_key.
//
// "none" (types.InfraDisabled) as registry_proxy or nix_cache opts the
// project out of that component. in names the ecosystems the project uses.
func (p *InfraProfile) Resolve(infra types.InfraConfig, in ProjectInputs) (*InfraProfile, error) {
	r := p.clone()

	r.Registry.URL = infra.RegistryProxyBase()
	r.Registry.Overrides = maps.Clone(infra.RegistryProxyOverrides)
	r.Registry.Paths = maps.Clone(infra.RegistryProxyPaths)
	if infra.RegistryProxy == types.InfraDisabled {
		r.Registry.Type = RegistryNone
	}

	r.applyNixCache(infra)

	if infra.BuildCache != "" {
		r.BuildCache.Type = BuildCacheType(infra.BuildCache)
	}
	r.BuildCache.URL = infra.BuildCacheURL

	if err := r.validate(in); err != nil {
		return nil, fmt.Errorf("infra profile %q: %w\nconfigure your organization's real endpoints under `infrastructure:` in .qsdev.yaml "+
			"(or `qsdev init --registry-proxy/--nix-cache/--nix-cache-public-key`), set a component to \"none\" to opt out, "+
			"or choose another --infra-profile", p.Name, err)
	}
	return r, nil
}

// applyNixCache sets the profile's Nix cache from infrastructure.nix_cache
// (a URL, or a bare Cachix cache name for a Cachix profile) and its key.
func (p *InfraProfile) applyNixCache(infra types.InfraConfig) {
	p.NixCache.URL, p.NixCache.CacheName, p.NixCache.PublicKey = "", "", infra.NixCachePublicKey
	switch nc := infra.NixCacheURL(); {
	case infra.NixCache == types.InfraDisabled:
		p.NixCache.Type = NixCacheNone
	case p.NixCache.Type == NixCacheCachix && cachixNameRe.MatchString(nc):
		p.NixCache.CacheName = nc
	default:
		p.NixCache.URL = nc
	}
}

// ResolveProjectInfrastructure validates the project's own infrastructure
// settings when no infra profile is selected. Nothing is required, but a
// configured Nix binary cache (infrastructure.nix_cache) is written into
// devenv.nix and the nix.conf guide, so it gets the same checks as a
// profile's: a real https URL or Cachix cache name plus its public key, no
// placeholders. A bare Cachix cache name becomes its substituter URL.
func ResolveProjectInfrastructure(infra types.InfraConfig) (types.InfraConfig, error) {
	if infra.NixCacheURL() == "" {
		return infra, nil
	}
	p := &InfraProfile{NixCache: NixCacheConfig{Type: NixCacheCachix}}
	p.applyNixCache(infra)
	if err := errors.Join(p.validateNixCache()...); err != nil {
		return infra, fmt.Errorf("%w\nset infrastructure.nix_cache and nix_cache_public_key in .qsdev.yaml to your binary cache "+
			"(or `qsdev init --nix-cache/--nix-cache-public-key`), or remove nix_cache", err)
	}
	infra.NixCache, infra.NixCachePublicKey = p.NixCacheNixConfig()
	return infra, nil
}

// ConfigOnly returns a copy of the profile with its registry proxy, Nix
// cache and build cache switched off: the implicit default profile, which
// contributes only its CI, dependency-update and documentation files and
// must not describe (or list credentials for) components it never applied.
func (p *InfraProfile) ConfigOnly() *InfraProfile {
	c := p.clone()
	c.Registry.Type = RegistryNone
	c.NixCache.Type = NixCacheNone
	c.BuildCache.Type = BuildCacheNone
	return c
}

// validate checks the resolved endpoints; see Resolve.
func (p *InfraProfile) validate(in ProjectInputs) error {
	var errs []error
	errs = append(errs, p.validateRegistry(in)...)
	errs = append(errs, p.validateNixCache()...)
	if p.BuildCache.URL != "" {
		if err := checkEndpointURL("infrastructure.build_cache_url", p.BuildCache.URL); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (p *InfraProfile) validateRegistry(in ProjectInputs) []error {
	var errs []error
	if p.Registry.URL != "" {
		if err := checkEndpointURL("infrastructure.registry_proxy", p.Registry.URL); err != nil {
			errs = append(errs, err)
		}
	}
	for _, eco := range slices.Sorted(maps.Keys(p.Registry.Overrides)) {
		if err := checkEndpointURL("infrastructure.registry_proxy_overrides."+eco, p.Registry.Overrides[eco]); err != nil {
			errs = append(errs, err)
		}
	}
	if !p.Registry.IsProxy() {
		return errs
	}
	var missing []string
	for _, eco := range in.Ecosystems {
		if p.Registry.serves(eco) && p.Registry.EcosystemURL(eco) == "" {
			missing = append(missing, eco)
		}
	}
	if len(missing) > 0 {
		errs = append(errs, fmt.Errorf("%w: the profile routes %s installs through its %s registry proxy, but infrastructure.registry_proxy is not set",
			ErrEndpointNotConfigured, strings.Join(missing, ", "), p.Registry.Type))
	}
	return errs
}

func (p *InfraProfile) validateNixCache() []error {
	nc := p.NixCache
	if nc.Type == "" || nc.Type == NixCacheNone {
		return nil
	}
	var errs []error
	switch {
	case nc.CacheName != "":
		if placeholderCacheNames[nc.CacheName] {
			errs = append(errs, fmt.Errorf("%w: infrastructure.nix_cache %q is the example Cachix cache; set it to your own cache", ErrPlaceholderEndpoint, nc.CacheName))
		}
	case nc.URL != "":
		if err := checkEndpointURL("infrastructure.nix_cache", nc.URL); err != nil {
			errs = append(errs, err)
		}
	default:
		errs = append(errs, fmt.Errorf("%w: the profile uses a %s Nix binary cache, but infrastructure.nix_cache is not set", ErrEndpointNotConfigured, nc.Type))
	}
	if nc.PublicKey == "" {
		errs = append(errs, fmt.Errorf("%w: the Nix binary cache needs infrastructure.nix_cache_public_key, which is not set", ErrEndpointNotConfigured))
	} else if err := checkNixPublicKey("infrastructure.nix_cache_public_key", nc.PublicKey); err != nil {
		errs = append(errs, err)
	}
	return errs
}

// Infrastructure returns the effective infrastructure settings of a resolved
// profile, in the form the generators consume (ecosystem module registry
// configs, devenv.nix, the nix.conf guide): the project's endpoints with the
// registry type's repository layout and the profile's build cache applied.
func (p *InfraProfile) Infrastructure() types.InfraConfig {
	out := types.InfraConfig{
		RegistryProxy:          p.Registry.URL,
		RegistryProxyOverrides: maps.Clone(p.Registry.Overrides),
		RegistryProxyPaths:     maps.Clone(p.Registry.Paths),
		BuildCacheURL:          p.BuildCache.URL,
	}
	if p.Registry.IsProxy() {
		for eco, path := range p.Registry.Layout() {
			if _, set := out.RegistryProxyPaths[eco]; set {
				continue
			}
			if out.RegistryProxyPaths == nil {
				out.RegistryProxyPaths = make(map[string]string)
			}
			out.RegistryProxyPaths[eco] = path
		}
	}
	out.NixCache, out.NixCachePublicKey = p.NixCacheNixConfig()
	if p.BuildCache.Type != "" && p.BuildCache.Type != BuildCacheNone {
		out.BuildCache = string(p.BuildCache.Type)
	}
	return out
}

// clone returns a deep copy of the profile.
func (p *InfraProfile) clone() *InfraProfile {
	c := *p
	c.Registry.Ecosystems = slices.Clone(p.Registry.Ecosystems)
	c.Registry.Overrides = maps.Clone(p.Registry.Overrides)
	c.Registry.Paths = maps.Clone(p.Registry.Paths)
	c.BuildCache.AuthEnvVars = slices.Clone(p.BuildCache.AuthEnvVars)
	c.Updates.EcosystemOverrides = maps.Clone(p.Updates.EcosystemOverrides)
	return &c
}
