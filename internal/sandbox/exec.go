package sandbox

import (
	"errors"
	"os"
	"os/exec"
	"slices"
	"strings"
)

// ErrSetupFailed marks a RunHook failure where the sandbox itself could not be
// established, so the hook never ran. Callers must not mistake it for the
// hook's own verdict.
var ErrSetupFailed = errors.New("sandbox setup failed; the hook did not run")

// SourceEnvironment returns the environment a hook's variables are filtered
// from: cfg.Environment when the caller supplied one, otherwise the current
// process environment.
func SourceEnvironment(cfg *SandboxConfig) map[string]string {
	if cfg.Environment != nil {
		return cfg.Environment
	}
	env := make(map[string]string)
	for _, e := range os.Environ() {
		if k, v, ok := strings.Cut(e, "="); ok {
			env[k] = v
		}
	}
	return env
}

// EnvList renders env as a sorted KEY=VALUE list for exec.Cmd.Env. The result
// is never nil: os/exec treats a nil Env as "inherit the parent environment",
// so an empty environment must stay empty instead of leaking every variable of
// the qsdev process (credentials included) into the hook.
func EnvList(env map[string]string) []string {
	list := make([]string, 0, len(env))
	for k, v := range env {
		list = append(list, k+"="+v)
	}
	slices.Sort(list)
	return list
}

// Attach connects o to cmd. Stdin is forwarded to the child as is. Each output
// stream goes to its writer when one is set; a stream without a writer is left
// unset so RunCommand captures it (capped at MaxCapturedOutput) into the
// SandboxResult.
func (o ExecOpts) Attach(cmd *exec.Cmd) {
	cmd.Stdin = o.Stdin
	if o.Stdout != nil {
		cmd.Stdout = o.Stdout
	}
	if o.Stderr != nil {
		cmd.Stderr = o.Stderr
	}
}

// ExitCode interprets the error returned by exec.Cmd.Run. A command that ran
// and exited yields its exit code and a nil error (0 on success); an error that
// is not an exit status (e.g. the binary could not be started) is returned.
func ExitCode(err error) (int, error) {
	if err == nil {
		return 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), nil
	}
	return 0, err
}
