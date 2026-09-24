package bwrap

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// disableUserNSMinVersion is the first bubblewrap release that accepts
// --disable-userns (0.8.0). Older releases reject the unknown option and would
// fail every hook, so the flag is only passed when the binary is new enough.
var disableUserNSMinVersion = [3]int{0, 8, 0} //nolint:gochecknoglobals // constant table

// disableUserNSCache memoises the per-binary probe so a hook run pays for at
// most one `bwrap --version` per process.
var disableUserNSCache sync.Map //nolint:gochecknoglobals // process-wide probe cache

// supportsDisableUserNS reports whether the bwrap binary at bwrapBin accepts
// --disable-userns. With it, the sandboxed process cannot create nested user
// namespaces by any route (unshare, clone(CLONE_NEWUSER), clone3), which closes
// the user-namespace-gated kernel attack surface at the kernel level instead of
// relying on per-syscall seccomp rules. Any probe failure reports false.
func supportsDisableUserNS(ctx context.Context, bwrapBin string) bool {
	if v, ok := disableUserNSCache.Load(bwrapBin); ok {
		return v.(bool)
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bwrapBin, "--version").Output() //nolint:gosec // bwrapBin is the probed sandbox binary
	supported := err == nil && versionAtLeast(parseBwrapVersion(string(out)), disableUserNSMinVersion)
	// A probe cut short by the caller's context says nothing about the
	// binary, so it is not memoised.
	if ctx.Err() == nil {
		disableUserNSCache.Store(bwrapBin, supported)
	}
	return supported
}

// parseBwrapVersion extracts the numeric version from `bwrap --version` output
// ("bubblewrap 0.11.2"). It returns nil when no version can be parsed.
func parseBwrapVersion(out string) []int {
	fields := strings.Fields(out)
	if len(fields) < 2 || fields[0] != "bubblewrap" {
		return nil
	}
	var version []int
	for part := range strings.SplitSeq(fields[1], ".") {
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil
		}
		version = append(version, n)
	}
	return version
}

// versionAtLeast reports whether version >= minimum, treating missing trailing
// components as zero. A nil version is never at least anything.
func versionAtLeast(version []int, minimum [3]int) bool {
	if len(version) == 0 {
		return false
	}
	for i, want := range minimum {
		got := 0
		if i < len(version) {
			got = version[i]
		}
		if got != want {
			return got > want
		}
	}
	return true
}
