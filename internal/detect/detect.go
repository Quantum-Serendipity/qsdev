// Package detect scans a project directory to identify programming languages,
// build systems, and existing configuration state. The results are returned
// as a [types.DetectedProject] that drives the rest of the init wizard.
package detect

import (
	"context"
	"log/slog"
	"os/user"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/container"
	"github.com/Quantum-Serendipity/qsdev/internal/sysinfo"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ContainerProbeTimeout bounds container runtime probing. Detect shells out to
// `podman info` and similar commands, which can block indefinitely on a stale
// storage lock or an unreachable podman machine; detection runs on every
// init/update/status and inside MCP tool handlers, so it must not hang.
const ContainerProbeTimeout = 3 * time.Second

// Option customizes Detect. Options exist so tests can replace the host
// probes (container runtime, OS, current user) with deterministic fakes.
type Option func(*options)

type options struct {
	prober       container.Prober
	probeTimeout time.Duration
	detectOS     func() *sysinfo.OSInfo
	username     func() (string, error)
}

// WithContainerProber replaces the prober used for container runtime
// detection.
func WithContainerProber(p container.Prober) Option {
	return func(o *options) { o.prober = p }
}

// WithContainerProbeTimeout overrides ContainerProbeTimeout.
func WithContainerProbeTimeout(d time.Duration) Option {
	return func(o *options) { o.probeTimeout = d }
}

// WithOSDetector replaces the host OS detection.
func WithOSDetector(fn func() *sysinfo.OSInfo) Option {
	return func(o *options) { o.detectOS = fn }
}

// WithUserLookup replaces the current-username lookup.
func WithUserLookup(fn func() (string, error)) Option {
	return func(o *options) { o.username = fn }
}

func currentUsername() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return u.Username, nil
}

// Detect scans projectRoot for language markers, lockfiles, configuration
// files, and git metadata, returning a fully populated DetectedProject.
// Container runtime probing honours ctx and is additionally bounded by
// ContainerProbeTimeout.
func Detect(ctx context.Context, projectRoot string, opts ...Option) types.DetectedProject {
	if ctx == nil {
		// A cobra command run without a context (e.g. RunE invoked directly)
		// reports a nil cmd.Context(); detection must still be bounded.
		ctx = context.Background()
	}
	o := options{
		prober:       &container.ExecProber{},
		probeTimeout: ContainerProbeTimeout,
		detectOS:     sysinfo.DetectOS,
		username:     currentUsername,
	}
	for _, opt := range opts {
		opt(&o)
	}

	registry := ecosystem.DefaultRegistry()
	summary := registry.DetectWithEnvironment(projectRoot)

	summary.Project.ContainerRuntime = detectContainerRuntime(ctx, o)

	if osInfo := o.detectOS(); osInfo != nil {
		summary.Project.OSFamily = osInfo.Family
	}
	if name, err := o.username(); err == nil {
		summary.Project.Username = name
	}

	slog.Debug("project detection complete",
		"ecosystems", len(summary.Project.Ecosystems),
		"has_go", summary.Project.HasGoMod,
		"has_node", summary.Project.HasPackageJSON,
		"has_container", summary.Project.HasDockerfile,
		"container_runtime", summary.Project.ContainerRuntime)
	return summary.Project
}

// detectContainerRuntime returns the active container runtime name, or "" when
// none is found. Probe commands cut short by the timeout are treated like
// failed commands, so the result is the best information gathered in time.
func detectContainerRuntime(ctx context.Context, o options) string {
	probeCtx, cancel := context.WithTimeout(ctx, o.probeTimeout)
	defer cancel()

	rtInfo, err := container.Detect(probeCtx, o.prober)
	if probeCtx.Err() != nil {
		slog.Debug("container runtime probe cut short", "error", probeCtx.Err())
	}
	if err != nil || rtInfo.Active == container.RuntimeNone {
		return ""
	}
	return string(rtInfo.Active)
}
