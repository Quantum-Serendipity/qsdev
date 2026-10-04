package devinit

import (
	"context"

	"github.com/Quantum-Serendipity/qsdev/internal/detect"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// hostProber answers the questions init, join, update and the lifecycle
// commands ask about the machine rather than the project: which container
// runtime and OS family it has (the probes detect.Detect runs, such as
// `docker --version`) and whether the devenv prerequisites are installed.
type hostProber struct {
	// detectOptions replace detect.Detect's host probes; none means the
	// real host.
	detectOptions []detect.Option
	// prerequisites reports which devenv prerequisites are installed.
	prerequisites func(context.Context) PrerequisiteResult
}

// host is the prober the commands use: the real machine. The package's tests
// replace it with deterministic fakes (see TestMain), so a test run does not
// start a host probe for every project it initializes.
var host = hostProber{prerequisites: CheckPrerequisites}

// detectProject runs project detection with the host prober's probes.
func (h hostProber) detectProject(ctx context.Context, projectRoot string) types.DetectedProject {
	return detect.Detect(ctx, projectRoot, h.detectOptions...)
}
