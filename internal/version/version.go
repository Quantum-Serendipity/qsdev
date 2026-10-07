package version

import (
	"fmt"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "manual"

	// gatewayImageDigest is the index digest of the gateway image the
	// release workflow published for this version, stamped by GoReleaser.
	// Empty for development and snapshot builds, and for the image's own
	// binary, which cannot contain its own digest.
	gatewayImageDigest = ""
)

// imageDigestRe is an OCI content digest as the release workflow's docker
// manifest push prints it.
var imageDigestRe = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// IsImageDigest reports whether d is a well-formed sha256 OCI content digest
// ("sha256:" and 64 lower-case hex digits).
func IsImageDigest(d string) bool {
	return imageDigestRe.MatchString(d)
}

// GatewayImageDigest returns the gateway image index digest stamped into
// this build, or "" when none was stamped or the stamped value is not a
// well-formed digest (a malformed stamp is ignored rather than producing an
// unpullable image reference).
func GatewayImageDigest() string {
	d := strings.TrimSpace(gatewayImageDigest)
	if !IsImageDigest(d) {
		return ""
	}
	return d
}

type BuildInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	BuiltBy   string `json:"builtBy"`
	GoVersion string `json:"goVersion"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

func Info() BuildInfo {
	info := BuildInfo{
		Version:   version,
		Commit:    commit,
		Date:      date,
		BuiltBy:   builtBy,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}

	if info.Commit == "none" || info.Version == "dev" {
		if bi, ok := debug.ReadBuildInfo(); ok {
			for _, s := range bi.Settings {
				switch s.Key {
				case "vcs.revision":
					if info.Commit == "none" && s.Value != "" {
						info.Commit = s.Value
					}
				case "vcs.time":
					if info.Date == "unknown" && s.Value != "" {
						info.Date = s.Value
					}
				case "vcs.modified":
					if s.Value == "true" && info.Commit != "none" {
						info.Commit += "-dirty"
					}
				}
			}
			if info.Version == "dev" && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
				info.Version = bi.Main.Version
			}
		}
	}

	if len(info.Commit) > 12 && !strings.HasSuffix(info.Commit, "-dirty") {
		info.Commit = info.Commit[:12]
	}
	if strings.HasSuffix(info.Commit, "-dirty") && len(info.Commit) > 18 {
		info.Commit = info.Commit[:12] + "-dirty"
	}

	return info
}

func (b BuildInfo) String() string {
	return fmt.Sprintf("%s (%s, built %s by %s, %s %s/%s)",
		b.Version, b.Commit, b.Date, b.BuiltBy, b.GoVersion, b.OS, b.Arch)
}
