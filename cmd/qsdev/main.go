package main

import (
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
	"github.com/Quantum-Serendipity/qsdev/internal/logcmd"
	"github.com/Quantum-Serendipity/qsdev/internal/logging"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserver"
	"github.com/Quantum-Serendipity/qsdev/internal/selfupdate"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

var logSession *logging.Session

func main() {
	instance.SetBranding(branding.Default())
	// Framework adapters, external-log providers and the release version are
	// wired through the instance package so downstream tools (including
	// scaffold-instance output) get exactly the same runtime as qsdev.
	instance.RegisterFrameworkAdapters()
	instance.ApplyBuildVersion()

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
	class := classifyInvocation(os.Args[1:])
	if class == logging.ClassUnlogged {
		return
	}

	level := logging.LevelFromEnv()
	stderrToo := strings.EqualFold(os.Getenv(branding.Get().EnvLogVar), "debug")

	projectRoot := logging.DetectProjectRoot()
	commandPath := detectCommandFromArgs()

	var err error
	logSession, err = logging.Init(logging.Config{
		Level:         level,
		StderrToo:     stderrToo,
		ProjectRoot:   projectRoot,
		ProjectScoped: class != logging.ClassGlobal && projectRoot != "",
		Automated:     class == logging.ClassAutomated,
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

// classifyInvocation extends logging.ClassifyInvocation with the embedded MCP
// servers ("mcp <provider>"), which are launched by the agent for every session
// and are therefore automated. They are derived from the provider registry so
// a newly registered server is classified without further wiring.
func classifyInvocation(args []string) logging.CommandClass {
	class := logging.ClassifyInvocation(args)
	if class == logging.ClassProject && len(args) >= 2 && args[0] == "mcp" {
		if _, ok := mcpserver.DefaultRegistry().Get(args[1]); ok {
			return logging.ClassAutomated
		}
	}
	return class
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
