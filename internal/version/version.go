package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "manual"
)

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
