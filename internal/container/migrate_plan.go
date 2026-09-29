package container

import (
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// This file holds the decisions the migration analyzer and the auto-fixer must
// agree on: which privileged ports can be remapped (and to what), and which
// bind mounts may be relabelled for SELinux (and how). Both sides derive them
// from the same compose file with the functions below.

// SELinux relabel options for bind mounts: "Z" gives the content a label
// private to one container, "z" a label shared by all containers.
const (
	selinuxPrivate = "Z"
	selinuxShared  = "z"
)

// publishedPort is a host port together with its protocol; the same number may
// be published once per protocol.
type publishedPort struct {
	port  int
	proto string
}

// parsePortSpec parses a short-syntax port entry ("80:80", "0.0.0.0:80:80/udp")
// into its host port. A container-only entry ("80") publishes nothing.
func parsePortSpec(spec string) (publishedPort, bool) {
	proto := "tcp"
	if base, p, found := strings.Cut(spec, "/"); found {
		spec, proto = base, p
	}
	port := parseHostPortFromString(spec)
	if port <= 0 {
		return publishedPort{}, false
	}
	return publishedPort{port: port, proto: proto}, true
}

// longPortSpec builds a publishedPort from a long-syntax port entry's
// "published" and "protocol" values.
func longPortSpec(published, protocol string) (publishedPort, bool) {
	port, err := strconv.Atoi(published)
	if err != nil || port <= 0 {
		return publishedPort{}, false
	}
	if protocol == "" {
		protocol = "tcp"
	}
	return publishedPort{port: port, proto: protocol}, true
}

// portRemapTarget returns the unprivileged host port a privileged port is
// remapped to (port+portRemapOffset). It reports false when that port is
// already published in the compose file, where remapping would make the fixed
// file fail with "address already in use"; such ports need a manual choice.
func portRemapTarget(p publishedPort, published map[publishedPort]bool) (int, bool) {
	target := publishedPort{port: p.port + portRemapOffset, proto: p.proto}
	if published[target] {
		return 0, false
	}
	return target.port, true
}

// projectBindHost returns the cleaned project-relative host path of a bind
// mount, and false when the host path is absolute, home-relative or escapes
// the project directory.
func projectBindHost(host string) (string, bool) {
	if !strings.HasPrefix(host, ".") {
		return "", false
	}
	cleaned := path.Clean(host)
	if cleaned != "." && !filepath.IsLocal(cleaned) {
		return "", false
	}
	return cleaned, true
}

// bindHostOf returns the host part of a short-syntax volume entry.
func bindHostOf(volume string) string {
	host, _, _ := strings.Cut(volume, ":")
	return host
}

// countBindHosts counts, per project-relative host path, how many services of
// a compose file bind-mount it.
func countBindHosts(volumesByService map[string][]string) map[string]int {
	counts := make(map[string]int)
	for _, volumes := range volumesByService {
		seen := make(map[string]bool)
		for _, vol := range volumes {
			host, ok := projectBindHost(bindHostOf(vol))
			if !ok || seen[host] {
				continue
			}
			seen[host] = true
			counts[host]++
		}
	}
	return counts
}

// selinuxRelabelOption returns the SELinux option that can be added to a bind
// mount automatically, and false when relabelling must be left to the user.
// Only paths inside the project qualify: relabelling system paths, the home
// directory or sockets changes their label for every other process on the host.
// A path several services mount gets the shared "z" label, because a private
// "Z" label would leave it readable only by the last container started.
func selinuxRelabelOption(host string, bindHosts map[string]int) (string, bool) {
	cleaned, ok := projectBindHost(host)
	if !ok {
		return "", false
	}
	if bindHosts[cleaned] > 1 {
		return selinuxShared, true
	}
	return selinuxPrivate, true
}

// hasSELinuxOption reports whether a short-syntax volume entry already carries
// a z or Z option.
func hasSELinuxOption(volume string) bool {
	parts := strings.Split(volume, ":")
	if len(parts) < 3 {
		return false
	}
	for opt := range strings.SplitSeq(parts[len(parts)-1], ",") {
		if opt == selinuxShared || opt == selinuxPrivate {
			return true
		}
	}
	return false
}

// appendSELinuxOption adds the relabel option to a short-syntax volume entry,
// comma-separated after any existing options.
func appendSELinuxOption(volume, option string) string {
	parts := strings.Split(volume, ":")
	if len(parts) >= 3 {
		parts[len(parts)-1] += "," + option
		return strings.Join(parts, ":")
	}
	return volume + ":" + option
}
