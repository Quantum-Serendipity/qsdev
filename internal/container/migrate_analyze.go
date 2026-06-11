package container

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ComposeFileNames are conventional compose file names to scan.
var ComposeFileNames = []string{
	"docker-compose.yml",
	"docker-compose.yaml",
	"compose.yaml",
	"compose.yml",
}

// Analyze scans a project directory for Docker Compose files and returns a
// migration report detailing incompatibilities with Podman rootless.
func Analyze(ctx context.Context, projectRoot string, prober Prober) (*MigrationReport, error) {
	info, err := Detect(ctx, prober)
	if err != nil {
		return nil, fmt.Errorf("detecting container runtime: %w", err)
	}

	caps, err := DetectCapabilities(ctx, prober, info)
	if err != nil {
		return nil, fmt.Errorf("detecting capabilities: %w", err)
	}

	composeFiles := findComposeFiles(projectRoot)

	report := &MigrationReport{
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		ProjectRoot:   projectRoot,
		SourceRuntime: RuntimeDocker,
		TargetRuntime: RuntimePodmanRootless,
		ComposeFiles:  composeFiles,
		RuntimeInfo:   info,
		Capabilities:  caps,
	}

	selinuxActive := detectSELinux(prober)

	for _, file := range composeFiles {
		issues, err := analyzeComposeFile(file, caps, selinuxActive)
		if err != nil {
			return nil, fmt.Errorf("analyzing %s: %w", file, err)
		}
		report.Issues = append(report.Issues, issues...)
	}

	report.Summary = buildSummary(report.Issues)
	return report, nil
}

// findComposeFiles returns absolute paths to compose files found in projectRoot.
func findComposeFiles(projectRoot string) []string {
	var found []string
	for _, name := range ComposeFileNames {
		path := filepath.Join(projectRoot, name)
		if _, err := os.Stat(path); err == nil {
			found = append(found, path)
		}
	}
	return found
}

// detectSELinux returns true if SELinux enforcement is active.
func detectSELinux(prober Prober) bool {
	data, err := prober.ReadFile("/sys/fs/selinux/enforce")
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) == "1"
}

// analyzeComposeFile parses a single compose file and runs all checks.
func analyzeComposeFile(filePath string, caps *Capabilities, selinuxActive bool) ([]MigrationIssue, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	servicesRaw, ok := doc["services"]
	if !ok {
		return nil, nil
	}
	services, ok := servicesRaw.(map[string]any)
	if !ok {
		return nil, nil
	}

	// Extract top-level volume names for bind-mount vs named-volume detection.
	topLevelVolumes := extractTopLevelVolumes(doc)

	var issues []MigrationIssue
	for name, svcRaw := range services {
		svc, ok := svcRaw.(map[string]any)
		if !ok {
			continue
		}
		issues = append(issues, checkVolumePermissions(name, svc, filePath, topLevelVolumes)...)
		issues = append(issues, checkImageQualification(name, svc, filePath)...)
		issues = append(issues, checkPrivilegedPorts(name, svc, filePath, caps)...)
		issues = append(issues, checkPrivilegedMode(name, svc, filePath)...)
		issues = append(issues, checkDockerSocketMount(name, svc, filePath)...)
		issues = append(issues, checkSELinuxLabels(name, svc, filePath, selinuxActive, topLevelVolumes)...)
	}

	return issues, nil
}

// extractTopLevelVolumes returns the set of named volumes declared at the
// top-level "volumes" key of a compose file.
func extractTopLevelVolumes(doc map[string]any) map[string]bool {
	result := make(map[string]bool)
	volsRaw, ok := doc["volumes"]
	if !ok {
		return result
	}
	vols, ok := volsRaw.(map[string]any)
	if !ok {
		return result
	}
	for name := range vols {
		result[name] = true
	}
	return result
}

// isBindMount returns true if a volume string looks like a bind mount
// rather than a named volume.
func isBindMount(vol string, topLevelVolumes map[string]bool) bool {
	// Split off the container path and options.
	host, _, _ := strings.Cut(vol, ":")
	if strings.HasPrefix(host, ".") || strings.HasPrefix(host, "/") || strings.HasPrefix(host, "~") {
		return true
	}
	// If the host part matches a top-level volume name, it is a named volume.
	if topLevelVolumes[host] {
		return false
	}
	// If the host part doesn't start with path chars and isn't a known volume,
	// treat it as a named volume (Docker convention).
	return false
}

// checkVolumePermissions flags bind mounts that lack userns_mode: keep-id.
func checkVolumePermissions(serviceName string, svc map[string]any, filePath string, topLevelVolumes map[string]bool) []MigrationIssue {
	volumes := extractStringList(svc, "volumes")
	if len(volumes) == 0 {
		return nil
	}

	// If service already has userns_mode: keep-id, skip.
	if mode, ok := svc["userns_mode"]; ok {
		if modeStr, ok := mode.(string); ok && modeStr == "keep-id" {
			return nil
		}
	}

	hasBindMount := false
	for _, vol := range volumes {
		if isBindMount(vol, topLevelVolumes) {
			hasBindMount = true
			break
		}
	}
	if !hasBindMount {
		return nil
	}

	return []MigrationIssue{{
		Category:    CategoryVolumePerms,
		Severity:    SeverityWarning,
		File:        filePath,
		Service:     serviceName,
		Description: fmt.Sprintf("service %q has bind mounts without userns_mode: keep-id; files may be owned by root inside the container", serviceName),
		AutoFixable: true,
		Fix: &MigrationFix{
			Description: "add userns_mode: keep-id to the service",
			YAMLPath:    fmt.Sprintf("services.%s.userns_mode", serviceName),
			YAMLValue:   "keep-id",
		},
	}}
}

// checkImageQualification flags unqualified Docker Hub image names.
func checkImageQualification(serviceName string, svc map[string]any, filePath string) []MigrationIssue {
	imageRaw, ok := svc["image"]
	if !ok {
		return nil
	}
	image, ok := imageRaw.(string)
	if !ok {
		return nil
	}

	qualified := qualifyImageName(image)
	if qualified == image {
		return nil
	}

	return []MigrationIssue{{
		Category:    CategoryImageName,
		Severity:    SeverityInfo,
		File:        filePath,
		Service:     serviceName,
		Description: fmt.Sprintf("service %q uses unqualified image %q; Podman requires fully-qualified names", serviceName, image),
		AutoFixable: true,
		Fix: &MigrationFix{
			Description: fmt.Sprintf("qualify image as %q", qualified),
			YAMLPath:    fmt.Sprintf("services.%s.image", serviceName),
			YAMLValue:   qualified,
		},
	}}
}

// qualifyImageName adds the docker.io prefix when missing.
func qualifyImageName(image string) string {
	// Split off tag/digest first so dots in tags don't confuse domain detection.
	nameOnly := image
	suffix := ""
	if idx := strings.LastIndex(image, ":"); idx >= 0 {
		nameOnly = image[:idx]
		suffix = image[idx:]
	} else if idx := strings.LastIndex(image, "@"); idx >= 0 {
		nameOnly = image[:idx]
		suffix = image[idx:]
	}

	// Already qualified (name part contains a domain with a dot or localhost).
	if strings.Contains(nameOnly, ".") || strings.HasPrefix(nameOnly, "localhost/") {
		return image
	}

	if !strings.Contains(nameOnly, "/") {
		// Single name like "nginx" → docker.io/library/nginx
		return "docker.io/library/" + nameOnly + suffix
	}
	// org/name like "bitnami/redis" → docker.io/bitnami/redis
	return "docker.io/" + nameOnly + suffix
}

// checkPrivilegedPorts flags ports < 1024 bound on the host.
func checkPrivilegedPorts(serviceName string, svc map[string]any, filePath string, caps *Capabilities) []MigrationIssue {
	if caps != nil && caps.PrivilegedPorts {
		return nil
	}

	portsRaw, ok := svc["ports"]
	if !ok {
		return nil
	}
	portsList, ok := portsRaw.([]any)
	if !ok {
		return nil
	}

	var issues []MigrationIssue
	for _, portEntry := range portsList {
		hostPort := extractHostPort(portEntry)
		if hostPort > 0 && hostPort < 1024 {
			remapped := hostPort + portRemapOffset
			issues = append(issues, MigrationIssue{
				Category:    CategoryPrivPorts,
				Severity:    SeverityWarning,
				File:        filePath,
				Service:     serviceName,
				Description: fmt.Sprintf("service %q binds privileged host port %d; rootless containers cannot bind ports below 1024", serviceName, hostPort),
				AutoFixable: true,
				Fix: &MigrationFix{
					Description: fmt.Sprintf("remap host port %d to %d", hostPort, remapped),
					YAMLPath:    fmt.Sprintf("services.%s.ports", serviceName),
					YAMLValue:   fmt.Sprintf("%d (remapped from %d)", remapped, hostPort),
				},
			})
		}
	}
	return issues
}

// extractHostPort parses a port entry and returns the host port, or 0.
// Bare integers (e.g. `- 80`) are container-only ports in Docker Compose
// and do not map to a host port.
func extractHostPort(entry any) int {
	switch v := entry.(type) {
	case string:
		return parseHostPortFromString(v)
	case int, float64:
		return 0
	case map[string]any:
		if pub, ok := v["published"]; ok {
			switch p := pub.(type) {
			case string:
				n, _ := strconv.Atoi(p)
				return n
			case int:
				return p
			case float64:
				return int(p)
			}
		}
	}
	return 0
}

// parseHostPortFromString parses "80:80", "0.0.0.0:80:80", or "80" forms.
func parseHostPortFromString(s string) int {
	// Strip protocol suffix if present (e.g. "80:80/tcp").
	if idx := strings.Index(s, "/"); idx >= 0 {
		s = s[:idx]
	}

	parts := strings.Split(s, ":")
	switch len(parts) {
	case 1:
		// Just a container port; no host port mapping.
		return 0
	case 2:
		// host:container
		n, _ := strconv.Atoi(parts[0])
		return n
	case 3:
		// ip:host:container
		n, _ := strconv.Atoi(parts[1])
		return n
	}
	return 0
}

// checkPrivilegedMode flags services running with privileged: true.
func checkPrivilegedMode(serviceName string, svc map[string]any, filePath string) []MigrationIssue {
	priv, ok := svc["privileged"]
	if !ok {
		return nil
	}
	isPriv, ok := priv.(bool)
	if !ok || !isPriv {
		return nil
	}

	return []MigrationIssue{{
		Category:    CategoryPrivileged,
		Severity:    SeverityCritical,
		File:        filePath,
		Service:     serviceName,
		Description: fmt.Sprintf("service %q runs in privileged mode; this is incompatible with rootless Podman", serviceName),
		AutoFixable: false,
		Fix: &MigrationFix{
			Description: "evaluate whether privileged mode is truly required",
			ManualSteps: []string{
				"Review whether this service genuinely needs privileged mode. Consider using specific capabilities instead.",
			},
		},
	}}
}

// checkDockerSocketMount flags bind mounts of the Docker socket.
func checkDockerSocketMount(serviceName string, svc map[string]any, filePath string) []MigrationIssue {
	volumes := extractStringList(svc, "volumes")
	for _, vol := range volumes {
		if strings.Contains(vol, "/var/run/docker.sock") {
			return []MigrationIssue{{
				Category:    CategorySocketMount,
				Severity:    SeverityWarning,
				File:        filePath,
				Service:     serviceName,
				Description: fmt.Sprintf("service %q mounts the Docker socket; this must be replaced with the Podman socket path", serviceName),
				AutoFixable: true,
				Fix: &MigrationFix{
					Description: "replace Docker socket with Podman socket",
					YAMLPath:    fmt.Sprintf("services.%s.volumes", serviceName),
					YAMLValue:   "${XDG_RUNTIME_DIR}/podman/podman.sock",
					EnvVar:      "XDG_RUNTIME_DIR",
					EnvValue:    "/run/user/1000",
				},
			}}
		}
	}
	return nil
}

// checkSELinuxLabels flags bind mounts missing :z or :Z suffixes when
// SELinux is active.
func checkSELinuxLabels(serviceName string, svc map[string]any, filePath string, selinuxActive bool, topLevelVolumes map[string]bool) []MigrationIssue {
	if !selinuxActive {
		return nil
	}

	volumes := extractStringList(svc, "volumes")
	var issues []MigrationIssue
	for _, vol := range volumes {
		if !isBindMount(vol, topLevelVolumes) {
			continue
		}
		// Check if already has :z or :Z suffix.
		parts := strings.Split(vol, ":")
		if len(parts) >= 3 {
			opts := parts[len(parts)-1]
			if strings.Contains(opts, "z") || strings.Contains(opts, "Z") {
				continue
			}
		}
		issues = append(issues, MigrationIssue{
			Category:    CategorySELinux,
			Severity:    SeverityInfo,
			File:        filePath,
			Service:     serviceName,
			Description: fmt.Sprintf("service %q bind mount %q lacks SELinux label (:Z); containers may not be able to access the volume", serviceName, vol),
			AutoFixable: true,
			Fix: &MigrationFix{
				Description: fmt.Sprintf("add SELinux label to bind mount %q", vol),
				YAMLPath:    fmt.Sprintf("services.%s.volumes", serviceName),
				YAMLValue:   appendSELinuxOption(vol),
			},
		})
	}
	return issues
}

func appendSELinuxOption(vol string) string {
	parts := strings.Split(vol, ":")
	if len(parts) >= 3 {
		parts[len(parts)-1] = parts[len(parts)-1] + ",Z"
		return strings.Join(parts, ":")
	}
	return vol + ":Z"
}

// extractStringList extracts a []string from a map key that holds either
// a []any of strings or is absent.
func extractStringList(m map[string]any, key string) []string {
	raw, ok := m[key]
	if !ok {
		return nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	var result []string
	for _, item := range list {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// buildSummary computes counts from a slice of issues.
func buildSummary(issues []MigrationIssue) MigrationSummary {
	var s MigrationSummary
	s.Total = len(issues)
	for _, issue := range issues {
		switch issue.Severity {
		case SeverityCritical:
			s.Critical++
		case SeverityWarning:
			s.Warning++
		case SeverityInfo:
			s.Info++
		}
		if issue.AutoFixable {
			s.AutoFixable++
		} else {
			s.ManualOnly++
		}
	}
	return s
}
