package update

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// NixUpdateAction describes the outcome of the nix update strategy.
type NixUpdateAction int

const (
	NixRegenerated      NixUpdateAction = iota // unmodified: overwrote in place
	NixSidecarCreated                          // modified: wrote .devenv.nix.new
	NixSkipped                                 // deleted by user or error
	NixForceOverwritten                        // --force: overwrote despite modification
)

// NixUpdateResult describes what happened when updating devenv.nix.
type NixUpdateResult struct {
	Action      NixUpdateAction
	DiffOutput  string // unified diff (empty for Regenerated/Skipped)
	NewFilePath string // path to .devenv.nix.new (empty for Regenerated)
	Message     string // user-facing message
}

// NixUpdateOptions controls the nix update behavior.
type NixUpdateOptions struct {
	ProjectRoot string
	FilePath    string // relative path (e.g., "devenv.nix")
	NewContent  []byte
	NewMode     os.FileMode
	Status      types.ModificationStatus
	Force       bool
	DryRun      bool // report the action that would be taken without touching the filesystem
}

// UpdateDevenvNix implements the devenv.nix update strategy. With DryRun set it
// computes and reports the same result but performs no filesystem mutation (no
// sidecar cleanup, overwrite, or sidecar write).
func UpdateDevenvNix(opts NixUpdateOptions) (*NixUpdateResult, error) {
	if opts.NewMode == 0 {
		opts.NewMode = fileutil.ModeReadWrite
	}

	relPath := filepath.FromSlash(opts.FilePath)
	absPath := filepath.Join(opts.ProjectRoot, relPath)
	sidecarPath := absPath + ".new"

	// write performs every file write below, so DryRun is honored uniformly.
	// rel is relative to the project root, and the write is confined to it: a
	// symlinked file or parent directory resolving outside is refused.
	write := func(rel string) error {
		if opts.DryRun {
			return nil
		}
		return fileutil.WriteFileAtomicInRoot(opts.ProjectRoot, rel, opts.NewContent, opts.NewMode)
	}

	// Clean up any stale sidecar.
	if !opts.DryRun {
		if err := CleanupSidecar(sidecarPath); err != nil {
			return nil, fmt.Errorf("cleanup stale sidecar: %w", err)
		}
	}

	switch opts.Status {
	case types.Deleted:
		if !opts.Force {
			return &NixUpdateResult{
				Action:  NixSkipped,
				Message: fmt.Sprintf("%s was deleted by user; skipping", opts.FilePath),
			}, nil
		}
		if err := write(relPath); err != nil {
			return nil, fmt.Errorf("force write deleted %s: %w", opts.FilePath, err)
		}
		return &NixUpdateResult{
			Action:  NixForceOverwritten,
			Message: NixForceOverwriteWarning(),
		}, nil

	case types.Unmodified, types.New:
		if err := write(relPath); err != nil {
			return nil, fmt.Errorf("regenerate %s: %w", opts.FilePath, err)
		}
		return &NixUpdateResult{
			Action:  NixRegenerated,
			Message: opts.FilePath + " regenerated",
		}, nil

	case types.Modified:
		if opts.Force {
			if err := write(relPath); err != nil {
				return nil, fmt.Errorf("force overwrite %s: %w", opts.FilePath, err)
			}
			return &NixUpdateResult{
				Action:  NixForceOverwritten,
				Message: NixForceOverwriteWarning(),
			}, nil
		}

		oldContent, err := os.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("read current %s: %w", opts.FilePath, err)
		}

		diffOutput, err := ComputeUnifiedDiff(oldContent, opts.NewContent, opts.FilePath, opts.FilePath+".new")
		if err != nil {
			return nil, fmt.Errorf("compute diff: %w", err)
		}

		if err := write(relPath + ".new"); err != nil {
			return nil, fmt.Errorf("write sidecar %s: %w", sidecarPath, err)
		}

		return &NixUpdateResult{
			Action:      NixSidecarCreated,
			DiffOutput:  diffOutput,
			NewFilePath: sidecarPath,
			Message:     NixMergeInstructions(sidecarPath),
		}, nil

	case types.Unknown:
		return &NixUpdateResult{
			Action:  NixSkipped,
			Message: fmt.Sprintf("%s status unknown; skipping update", opts.FilePath),
		}, nil

	default:
		return &NixUpdateResult{
			Action:  NixSkipped,
			Message: fmt.Sprintf("%s: unrecognized status %v; skipping", opts.FilePath, opts.Status),
		}, nil
	}
}
