package toolcheck

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
)

// Info holds the result of detecting a tool on the system.
type Info struct {
	Found   bool
	Path    string
	Version string
}

// Detect checks whether the named tool exists on PATH and, if so,
// runs it with versionArg to capture a version string.
func Detect(ctx context.Context, name, versionArg string) Info {
	path, err := exec.LookPath(name)
	if err != nil {
		return Info{}
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, versionArg)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return Info{Found: true, Path: path}
	}

	version := strings.TrimSpace(strings.SplitN(out.String(), "\n", 2)[0])
	return Info{Found: true, Path: path, Version: version}
}
