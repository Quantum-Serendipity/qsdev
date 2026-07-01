package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/cmd"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/addons/devinit"
	"github.com/Quantum-Serendipity/qsdev/instance"
	"github.com/Quantum-Serendipity/qsdev/internal/bugreport"
	_ "github.com/Quantum-Serendipity/qsdev/internal/extlog/providers"
	"github.com/Quantum-Serendipity/qsdev/internal/logcmd"
	"github.com/Quantum-Serendipity/qsdev/internal/logging"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/selfupdate"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"

	// Universal MCP server framework adapters. Concrete adapters under
	// internal/mcpserve/adapters/* delegate to addon packages (e.g.
	// addons/claudecode) and so MUST NOT be imported by the mcpserve server
	// package itself — that would create an import cycle. They are wired in
	// explicitly from this entry point (see registerFrameworkAdapters) rather than
	// self-registering from init(), so registration order is visible here. The
	// aliases avoid colliding with the addon packages of the same name.
	claudecodeadapter "github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/claudecode"
	clineadapter "github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/cline"
	codexadapter "github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/codex"
	cursoradapter "github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/cursor"
	windsurfadapter "github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/windsurf"
)

var logSession *logging.Session

// registerFrameworkAdapters explicitly wires the universal MCP server's framework
// adapters into spi.DefaultRegistry(). Explicit registration here (rather than a
// per-package init()) keeps the set and its order visible at the entry point, per
// the project's Go conventions and DefaultRegistry's own guidance. A duplicate-id
// error can only mean an adapter was listed twice — a build wiring mistake worth
// surfacing loudly.
func registerFrameworkAdapters() {
	reg := spi.DefaultRegistry()
	for _, a := range []spi.FrameworkAdapter{
		claudecodeadapter.New(),
		clineadapter.New(),
		codexadapter.New(),
		cursoradapter.New(),
		windsurfadapter.New(),
	} {
		if err := reg.Register(a); err != nil {
			panic(fmt.Sprintf("registering framework adapter %q: %v", a.ID(), err))
		}
	}
}

func main() {
	instance.SetBranding(branding.Default())
	registerFrameworkAdapters()

	if vi := version.Info(); vi.Version != "dev" && vi.Version != "(devel)" {
		instance.SetVersionOverride(vi.Version, vi.Commit)
	}

	bootstrap.Configure(
		bootstrap.WithSteps(
			devenv.InstallDevenvStep(),
			devenv.InstallDirenvStep(),
			claudecode.InstallClaudeStep(),
		),
	)

	devenv.Configure(
		devenv.WithDefaultLanguages("go"),
		devenv.WithDirenv(true),
	)
	claudecode.Configure(
		claudecode.WithDefaultPermissions(claudecode.PermissionPresetStandard),
	)
	devinit.Configure(
		devinit.WithDetectProjectType(true),
		devinit.WithPlanPreview(true),
	)

	instance.AddCommands(selfupdate.Command())
	instance.AddCommands(logcmd.Command())
	instance.AddCommands(bugreport.Command())

	// Pre-parse --debug from args before cobra processes them.
	// Sets QSDEV_LOG=debug so the OnInitialize callback picks it up.
	// The flag is consumed here and removed from os.Args so cobra
	// doesn't reject it as unknown.
	os.Args = extractDebugFlag(os.Args)

	cobra.OnInitialize(initLogging)

	updateCh := selfupdate.BackgroundCheck(version.Info().Version)
	cmd.Main()
	if logSession != nil {
		logSession.Close()
	}
	selfupdate.PrintNotice(updateCh)
}

// extractDebugFlag scans args for --debug, sets QSDEV_LOG=debug if found,
// and returns args with --debug removed.
func extractDebugFlag(args []string) []string {
	var filtered []string
	found := false
	for _, arg := range args {
		if arg == "--debug" {
			found = true
			continue
		}
		filtered = append(filtered, arg)
	}
	if found {
		_ = os.Setenv(branding.Get().EnvLogVar, "debug")
	}
	return filtered
}

// initLogging is called by cobra.OnInitialize during Execute(), before any
// command's PreRun. It sets up the two-tier structured logging system.
func initLogging() {
	level := logging.LevelFromEnv()
	stderrToo := strings.EqualFold(os.Getenv(branding.Get().EnvLogVar), "debug")

	projectRoot := logging.DetectProjectRoot()
	commandPath := detectCommandFromArgs()
	isProjectCmd := logging.IsProjectScopedCommand(commandPath)

	var err error
	logSession, err = logging.Init(logging.Config{
		Level:         level,
		StderrToo:     stderrToo,
		ProjectRoot:   projectRoot,
		ProjectScoped: isProjectCmd && projectRoot != "",
	})
	if err != nil {
		slog.Warn("logging init failed", "error", err)
		return
	}
	if logSession != nil {
		logSession.Command = commandPath
		slog.Info("command starting", "command", commandPath)
	}
}

// detectCommandFromArgs builds a rough command path from os.Args for the
// opening log record. This runs before cobra parses, so it uses simple
// heuristics: take args until one starts with "-".
func detectCommandFromArgs() string {
	parts := []string{branding.Get().AppName}
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-") {
			break
		}
		parts = append(parts, arg)
	}
	return strings.Join(parts, " ")
}
