package main

import (
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/cmd"
	"fastcat.org/go/gdev/instance"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/addons/devinit"
	"github.com/Quantum-Serendipity/qsdev/internal/bugreport"
	_ "github.com/Quantum-Serendipity/qsdev/internal/extlog/providers"
	"github.com/Quantum-Serendipity/qsdev/internal/logcmd"
	"github.com/Quantum-Serendipity/qsdev/internal/logging"
	"github.com/Quantum-Serendipity/qsdev/internal/selfupdate"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
)

var logSession *logging.Session

func main() {
	instance.SetAppName("qsdev")
	instance.SetVersion(version.Info().Version)

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
		_ = os.Setenv("QSDEV_LOG", "debug")
	}
	return filtered
}

// initLogging is called by cobra.OnInitialize during Execute(), before any
// command's PreRun. It sets up the two-tier structured logging system.
func initLogging() {
	level := logging.LevelFromEnv()
	stderrToo := strings.EqualFold(os.Getenv("QSDEV_LOG"), "debug")

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
	parts := []string{"qsdev"}
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-") {
			break
		}
		parts = append(parts, arg)
	}
	return strings.Join(parts, " ")
}
