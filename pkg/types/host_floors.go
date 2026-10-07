package types

// Host tool floors: the oldest versions of the host programs the generated
// environment and its hooks run on. They are the single source for the
// generator (devenv.yaml require_version), `qsdev devenv doctor` and the
// installers, so a host doctor passes can run what init generates.
const (
	// MinDevenv is the oldest devenv ("major.minor") the generated
	// devenv.yaml accepts (its require_version).
	MinDevenv = "2.1"

	// MinNix is the oldest Nix with `nix profile` and flakes, which the
	// devenv bootstrap and the generated environment rely on.
	MinNix = "2.4"

	// MinHookPython is the oldest python3 ("major.minor") the generated
	// Python hooks support (D20). Each hook script blocks (exit 2) below it;
	// the scripts' in-file `_MIN_PYTHON` tuples are pinned to it by tests.
	MinHookPython = "3.9"
)
