package container

// Runtime identifies a container runtime backend.
type Runtime string

const (
	RuntimePodmanRootless Runtime = "podman-rootless"
	RuntimePodmanRootful  Runtime = "podman-rootful"
	RuntimeDocker         Runtime = "docker"
	RuntimeNspawn         Runtime = "nspawn"
	RuntimeNone           Runtime = "none"
)

func (r Runtime) String() string { return string(r) }

// IsPodman returns true for both rootless and rootful Podman runtimes.
func (r Runtime) IsPodman() bool {
	return r == RuntimePodmanRootless || r == RuntimePodmanRootful
}

// IsRootless returns true only for Podman rootless.
func (r Runtime) IsRootless() bool {
	return r == RuntimePodmanRootless
}

// RuntimeInfo holds the result of container runtime detection.
type RuntimeInfo struct {
	Active          Runtime   // The selected/preferred runtime
	Available       []Runtime // All detected runtimes
	Version         string    // Version of the active runtime
	Path            string    // Absolute path to the active runtime binary
	Rootless        bool      // Whether the active runtime runs rootless
	SocketPath      string    // Unix socket path for API access
	ComposeMethod   string    // "podman-compose", "docker-compose-via-socket", "docker-compose", "none"
	HasDockerCompat bool      // docker command is a Podman alias
}

// Capabilities describes what the detected container runtime can do.
type Capabilities struct {
	GPUPassthrough          bool // NVIDIA or AMD GPU devices detected
	NFSMounts               bool // NFS mounts overlap with project paths
	PrivilegedPorts         bool // Can bind ports < 1024
	RootlessSupported       bool // Rootless mode is available
	UserNamespaceConfigured bool // subuid/subgid configured for current user
	CgroupsV2               bool // cgroups v2 is active
}

// NeedsRootfulFallback returns human-readable reasons why rootless mode
// is insufficient for the detected project. An empty slice means rootless
// is fully sufficient.
func (c *Capabilities) NeedsRootfulFallback() []string {
	if c == nil {
		return nil
	}
	var reasons []string
	if c.GPUPassthrough {
		reasons = append(reasons, "NVIDIA/AMD GPU detected: rootless containers cannot pass through GPU devices (requires CAP_MKNOD)")
	}
	if c.NFSMounts {
		reasons = append(reasons, "NFS mounts detected: rootless containers cannot bind-mount NFS paths due to UID remapping incompatibility")
	}
	return reasons
}
