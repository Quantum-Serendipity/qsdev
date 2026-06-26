package container

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
)

// Defaults for the generated gateway deployment. They are overridable per call
// via GenerateOptions; the constants keep a single source of truth.
const (
	// DefaultImage is the published gateway image reference.
	DefaultImage = "ghcr.io/quantum-serendipity/qsdev:latest"
	// DefaultServiceName is the docker-compose service name.
	DefaultServiceName = "qsdev-mcp-gateway"
	// DefaultMCPServerName is the key under which the gateway is registered in
	// a framework's .mcp.json.
	DefaultMCPServerName = "qsdev-gateway"
	// DefaultGatewayPort is the Streamable HTTP port the gateway listens on.
	DefaultGatewayPort = 8765
	// ContainerWorkspace is the in-container mount point for the project root.
	ContainerWorkspace = "/workspace"
	// ComposeFileName is the conventional filename for the generated fragment.
	ComposeFileName = "docker-compose.gateway.yaml"
	// envDeployMode / envProjectRoot are the container's deploy-mode env keys.
	envDeployMode   = "QSDEV_DEPLOY_MODE"
	envProjectRoot  = "QSDEV_PROJECT_ROOT"
	envGatewayAgent = "QSDEV_GATEWAY_AGENTS"
	// envTLSCert / envTLSKey / envTLSClientCA name the mTLS material the gateway
	// requires to boot (fail-closed). The names mirror internal/mcpserve's
	// tlsconfig.go EnvTLS* contract; they are redefined here — like
	// QSDEV_DEPLOY_MODE / QSDEV_PROJECT_ROOT above — to avoid an import cycle.
	envTLSCert     = "QSDEV_TLS_CERT"
	envTLSKey      = "QSDEV_TLS_KEY"
	envTLSClientCA = "QSDEV_TLS_CLIENT_CA"
	// ContainerTLSCert / ContainerTLSKey / ContainerTLSClientCA are the in-container
	// paths at which the mTLS material is mounted read-only.
	ContainerTLSCert     = "/tls/server.crt"
	ContainerTLSKey      = "/tls/server.key"
	ContainerTLSClientCA = "/tls/client-ca.crt"
	// GatewayBindAll is the reachable bind address the gateway uses so its
	// published port is usable from outside the container. A non-loopback bind
	// requires the mTLS material above, which the generated service supplies.
	GatewayBindAll = "0.0.0.0"
	// noNewPrivileges is the security_opt that prevents the container process from
	// gaining privileges via setuid/setgid binaries.
	noNewPrivileges = "no-new-privileges:true"
)

// FrameworkProfile is the minimal capability slice the generator needs to decide
// whether a framework requires the Gateway. It is intentionally tiny so callers
// can build it from any source (a detected adapter, a config file, a test).
type FrameworkProfile struct {
	ID          aiframework.FrameworkID
	Tier        aiframework.EnforcementTier
	NativeHooks bool
}

// NeedsGateway reports whether a framework's profile requires the qsdev MCP
// Gateway. The decision is DERIVED from the enforcement tier, never a hardcoded
// framework name list: kernel and hook tiers can enforce a pre-execution
// decision natively, so they do not need the gateway; policy, advisory, and
// external tiers cannot, so they do. Equivalent to "lacks native hook-or-better
// enforcement".
func (p FrameworkProfile) NeedsGateway() bool {
	return NeedsGatewayTier(p.Tier)
}

// NeedsGatewayTier is the pure tier predicate behind FrameworkProfile.NeedsGateway.
// A tier strictly weaker than TierHook (i.e. lower Strength) cannot enforce
// natively and therefore needs the gateway.
func NeedsGatewayTier(tier aiframework.EnforcementTier) bool {
	return tier.Strength() < aiframework.TierHook.Strength()
}

// defaultProfiles is the authoritative enforcement-tier catalog for the
// supported frameworks, mirroring the validated capability data the per-adapter
// research encodes (internal/mcpserve/adapters/*): Claude Code enforces via
// PreToolUse hooks (TierHook); Codex ships a native sandbox (TierKernel); Cursor,
// Windsurf, and the Continue.dev family have no native pre-tool hook
// (TierAdvisory). It is a capability CATALOG, not a skip-list — NeedsGatewayTier
// derives the gateway decision from the tier.
var defaultProfiles = map[aiframework.FrameworkID]FrameworkProfile{
	aiframework.ClaudeCode:  {ID: aiframework.ClaudeCode, Tier: aiframework.TierHook, NativeHooks: true},
	aiframework.Codex:       {ID: aiframework.Codex, Tier: aiframework.TierKernel, NativeHooks: true},
	aiframework.Cursor:      {ID: aiframework.Cursor, Tier: aiframework.TierAdvisory},
	aiframework.Windsurf:    {ID: aiframework.Windsurf, Tier: aiframework.TierAdvisory},
	aiframework.ContinueDev: {ID: aiframework.ContinueDev, Tier: aiframework.TierAdvisory},
}

// ProfileFor returns the capability profile for a framework id. Known frameworks
// come from the validated catalog; an UNKNOWN id is treated conservatively as
// TierAdvisory (assume it cannot enforce natively, so it needs the gateway)
// rather than silently skipped — fail-safe toward enforcement, and not a
// hardcoded exclusion list.
func ProfileFor(id aiframework.FrameworkID) FrameworkProfile {
	if p, ok := defaultProfiles[id]; ok {
		return p
	}
	return FrameworkProfile{ID: id, Tier: aiframework.TierAdvisory}
}

// DeployMode is one of the three deployment models (see package doc).
type DeployMode string

const (
	// DeployNative runs qsdev as a local stdio process (default).
	DeployNative DeployMode = "native"
	// DeployGateway runs the container as an enforcing MCP proxy.
	DeployGateway DeployMode = "gateway"
	// DeployStandalone runs the container with an explicit mounted root.
	DeployStandalone DeployMode = "standalone"
)

// Valid reports whether m is one of the three recognized modes.
func (m DeployMode) Valid() bool {
	switch m {
	case DeployNative, DeployGateway, DeployStandalone:
		return true
	default:
		return false
	}
}

// ParseDeployMode resolves the effective deployment mode. The explicit flag
// value wins; an empty flag falls back to the QSDEV_DEPLOY_MODE env value
// (envVal); an empty fallback yields DeployNative. An unrecognized value is a
// clear error so a typo never silently degrades enforcement.
func ParseDeployMode(flag, envVal string) (DeployMode, error) {
	raw := strings.TrimSpace(flag)
	if raw == "" {
		raw = strings.TrimSpace(envVal)
	}
	if raw == "" {
		return DeployNative, nil
	}
	m := DeployMode(strings.ToLower(raw))
	if !m.Valid() {
		return "", fmt.Errorf("unknown deploy-mode %q: want %q, %q, or %q",
			raw, DeployNative, DeployGateway, DeployStandalone)
	}
	return m, nil
}

// ErrStandaloneRootRequired is returned when standalone mode is selected without
// an explicit project root. A container's working directory is not a meaningful
// project root, so CWD auto-detection is deliberately refused in this mode.
var ErrStandaloneRootRequired = errors.New(
	"standalone deploy-mode requires an explicit project root: pass --project-root " +
		"or set QSDEV_PROJECT_ROOT (a container working directory is not a meaningful project root)")

// StandaloneProjectRoot resolves the project root for standalone mode from the
// explicit sources ONLY (the --project-root flag, then QSDEV_PROJECT_ROOT). It
// never falls back to the working directory. getenv may be nil, in which case
// only the flag is consulted. It returns ErrStandaloneRootRequired when neither
// source supplies a value.
func StandaloneProjectRoot(flagRoot string, getenv func(string) string) (string, error) {
	if r := strings.TrimSpace(flagRoot); r != "" {
		return r, nil
	}
	if getenv != nil {
		if r := strings.TrimSpace(getenv(envProjectRoot)); r != "" {
			return r, nil
		}
	}
	return "", ErrStandaloneRootRequired
}

// MCPServerConfig is the .mcp.json entry that points a framework at the running
// gateway container. The generator emits the Streamable HTTP form (Type+URL);
// the docker-exec command form is documented in DockerExecEntry for clients
// that prefer to attach to the container over stdio.
type MCPServerConfig struct {
	Type    string   `json:"type,omitempty"`
	URL     string   `json:"url,omitempty"`
	Command string   `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
}

// GenerateOptions is the input to ContainerConfigGenerator.Generate.
type GenerateOptions struct {
	// ProjectRoot is the host path mounted read-only into the container. Empty
	// yields a "." host mount (compose resolves it relative to the file).
	ProjectRoot string
	// Frameworks is the set of frameworks detected for the project. Only those
	// whose profile NeedsGateway influences the output.
	Frameworks []FrameworkProfile
	// Port is the host:container Streamable HTTP port (default DefaultGatewayPort).
	Port int
	// Image overrides the container image (default DefaultImage).
	Image string
	// ServiceName overrides the compose service name (default DefaultServiceName).
	ServiceName string
	// MCPServerName overrides the .mcp.json key (default DefaultMCPServerName).
	MCPServerName string
}

// Artifacts is the pure, in-memory result of container-config generation. The
// caller decides what to persist; nothing here touches the filesystem.
type Artifacts struct {
	// NeedsGateway is true when at least one supplied framework requires the
	// gateway. When false every other field is zero/empty.
	NeedsGateway bool
	// GatewayFrameworks lists the framework ids that drove gateway generation,
	// sorted for determinism.
	GatewayFrameworks []aiframework.FrameworkID
	// ComposeFileName is the conventional filename for ComposeYAML.
	ComposeFileName string
	// ComposeYAML is the docker-compose fragment for the gateway service.
	ComposeYAML string
	// MCPServerName is the key under which MCPServerConfig is registered.
	MCPServerName string
	// MCPServerConfig is the Streamable HTTP .mcp.json entry for the gateway.
	MCPServerConfig MCPServerConfig
	// MCPJSON is the full {"mcpServers": {name: cfg}} document.
	MCPJSON string
	// DockerExecEntry is an alternative stdio .mcp.json entry that attaches to
	// the running container via `docker exec` instead of HTTP.
	DockerExecEntry MCPServerConfig
	// EnvVars are the container's deploy env settings.
	EnvVars map[string]string
}

// ContainerConfigGenerator turns a set of detected framework profiles into the
// deployment artifacts (docker-compose fragment, .mcp.json entry, env config)
// for the Gateway model. It is a pure value: construct the zero value and call
// Generate.
type ContainerConfigGenerator struct{}

// composeFile / composeService are the typed compose model the generator
// marshals with yaml.v3, so callers (and tests) can round-trip the output.
type composeFile struct {
	Services map[string]composeService `yaml:"services"`
}

type composeService struct {
	Image       string            `yaml:"image"`
	Command     []string          `yaml:"command,omitempty"`
	Ports       []string          `yaml:"ports,omitempty"`
	Volumes     []string          `yaml:"volumes,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty"`
	ReadOnly    bool              `yaml:"read_only,omitempty"`
	SecurityOpt []string          `yaml:"security_opt,omitempty"`
	Restart     string            `yaml:"restart,omitempty"`
}

// Generate produces the gateway deployment artifacts. When none of the supplied
// frameworks needs the gateway it returns an Artifacts with NeedsGateway=false
// and emits no compose/.mcp.json content — the caller should then write nothing.
func (ContainerConfigGenerator) Generate(opts GenerateOptions) (*Artifacts, error) {
	port := opts.Port
	if port <= 0 {
		port = DefaultGatewayPort
	}
	image := orDefault(opts.Image, DefaultImage)
	service := orDefault(opts.ServiceName, DefaultServiceName)
	mcpName := orDefault(opts.MCPServerName, DefaultMCPServerName)

	gateway := gatewayFrameworks(opts.Frameworks)
	env := map[string]string{
		envDeployMode:  string(DeployGateway),
		envProjectRoot: ContainerWorkspace,
	}
	art := &Artifacts{
		NeedsGateway:      len(gateway) > 0,
		GatewayFrameworks: gateway,
		EnvVars:           env,
	}
	if !art.NeedsGateway {
		return art, nil
	}

	composeYAML, err := renderCompose(service, image, port, hostMount(opts.ProjectRoot))
	if err != nil {
		return nil, err
	}

	httpEntry := MCPServerConfig{
		Type: "http",
		URL:  fmt.Sprintf("http://localhost:%d/mcp", port),
	}
	mcpJSON, err := renderMCPJSON(mcpName, httpEntry)
	if err != nil {
		return nil, err
	}

	art.ComposeFileName = ComposeFileName
	art.ComposeYAML = composeYAML
	art.MCPServerName = mcpName
	art.MCPServerConfig = httpEntry
	art.MCPJSON = mcpJSON
	art.DockerExecEntry = MCPServerConfig{
		Command: "docker",
		Args: []string{
			"exec", "-i", service,
			"/qsdev", "mcp", "serve", "--deploy-mode", string(DeployGateway),
		},
	}
	return art, nil
}

// gatewayFrameworks filters profiles to those needing the gateway, returning
// their ids sorted for deterministic output.
func gatewayFrameworks(profiles []FrameworkProfile) []aiframework.FrameworkID {
	var ids []aiframework.FrameworkID
	seen := make(map[aiframework.FrameworkID]struct{}, len(profiles))
	for _, p := range profiles {
		if !p.NeedsGateway() {
			continue
		}
		if _, dup := seen[p.ID]; dup {
			continue
		}
		seen[p.ID] = struct{}{}
		ids = append(ids, p.ID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// renderCompose marshals the gateway service into a docker-compose fragment.
//
// The image ENTRYPOINT already supplies "/qsdev mcp serve", and docker APPENDS
// command to (never replaces) the ENTRYPOINT, so command here carries ONLY flags
// — a leading "mcp"/"serve" would double the subcommand. The service binds
// 0.0.0.0 so the published port is reachable, and mounts the mTLS material the
// gateway requires to boot: it fails closed (refuses to start) if any of the
// cert/key/client-CA files is absent, so the secure path is the default.
func renderCompose(service, image string, port int, hostPath string) (string, error) {
	portMap := fmt.Sprintf("%d:%d", port, port)
	cf := composeFile{
		Services: map[string]composeService{
			service: {
				Image: image,
				Command: []string{
					"--deploy-mode", string(DeployGateway),
					"--transport", "http",
					"--port", strconv.Itoa(port),
					"--project-root", ContainerWorkspace,
					"--bind", GatewayBindAll,
				},
				Ports: []string{portMap},
				Volumes: []string{
					hostPath + ":" + ContainerWorkspace + ":ro",
					// mTLS material, read-only. Host paths default to ./tls/* so an
					// operator drops their certs in and runs; override via env.
					"${QSDEV_TLS_CERT:-./tls/server.crt}:" + ContainerTLSCert + ":ro",
					"${QSDEV_TLS_KEY:-./tls/server.key}:" + ContainerTLSKey + ":ro",
					"${QSDEV_TLS_CLIENT_CA:-./tls/client-ca.crt}:" + ContainerTLSClientCA + ":ro",
				},
				Environment: map[string]string{
					envDeployMode:  string(DeployGateway),
					envProjectRoot: ContainerWorkspace,
					// In-container paths to the mounted mTLS material. Authentication
					// is the client certificate; the gateway will not start without
					// all three files present (fail-closed).
					envTLSCert:     ContainerTLSCert,
					envTLSKey:      ContainerTLSKey,
					envTLSClientCA: ContainerTLSClientCA,
					// Authorization allow-list of client-cert CNs. Empty admits any
					// certificate signed by the client CA; set it to restrict to
					// named CNs. Authentication itself is mTLS (above), not this list.
					envGatewayAgent: "",
				},
				ReadOnly:    true,
				SecurityOpt: []string{noNewPrivileges},
				Restart:     "unless-stopped",
			},
		},
	}
	out, err := yaml.Marshal(cf)
	if err != nil {
		return "", fmt.Errorf("marshaling gateway compose: %w", err)
	}
	return string(out), nil
}

// renderMCPJSON marshals the {"mcpServers": {name: cfg}} document.
func renderMCPJSON(name string, cfg MCPServerConfig) (string, error) {
	doc := map[string]any{"mcpServers": map[string]MCPServerConfig{name: cfg}}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling gateway .mcp.json entry: %w", err)
	}
	return string(out) + "\n", nil
}

// hostMount returns the compose host path for the project mount, defaulting to
// "." when no root is supplied.
func hostMount(projectRoot string) string {
	if r := strings.TrimSpace(projectRoot); r != "" {
		return r
	}
	return "."
}

func orDefault(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}
