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
	DryRun      bool
}

// UpdateDevenvNix implements the devenv.nix update strategy.
func UpdateDevenvNix(opts NixUpdateOptions) (*NixUpdateResult, error) {
	if opts.NewMode == 0 {
		opts.NewMode = fileutil.ModeReadWrite
	}

	absPath := filepath.Join(opts.ProjectRoot, opts.FilePath)
	sidecarPath := absPath + ".new"

	// Clean up any stale sidecar.
	if err := CleanupSidecar(sidecarPath); err != nil {
		return nil, fmt.Errorf("cleanup stale sidecar: %w", err)
	}

	switch opts.Status {
	case types.Deleted:
		if !opts.Force {
			return &NixUpdateResult{
				Action:  NixSkipped,
				Message: "devenv.nix was deleted by user; skipping",
			}, nil
		}
		if err := fileutil.WriteFileAtomic(absPath, opts.NewContent, opts.NewMode); err != nil {
			return nil, fmt.Errorf("force write deleted devenv.nix: %w", err)
		}
		return &NixUpdateResult{
			Action:  NixForceOverwritten,
			Message: NixForceOverwriteWarning(),
		}, nil

	case types.Unmodified, types.New:
		if err := fileutil.WriteFileAtomic(absPath, opts.NewContent, opts.NewMode); err != nil {
			return nil, fmt.Errorf("regenerate devenv.nix: %w", err)
		}
		return &NixUpdateResult{
			Action:  NixRegenerated,
			Message: "devenv.nix regenerated",
		}, nil

	case types.Modified:
		if opts.Force {
			if err := fileutil.WriteFileAtomic(absPath, opts.NewContent, opts.NewMode); err != nil {
				return nil, fmt.Errorf("force overwrite devenv.nix: %w", err)
			}
			return &NixUpdateResult{
				Action:  NixForceOverwritten,
				Message: NixForceOverwriteWarning(),
			}, nil
		}

		oldContent, err := os.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("read current devenv.nix: %w", err)
		}

		diffOutput, err := ComputeUnifiedDiff(oldContent, opts.NewContent, opts.FilePath, opts.FilePath+".new")
		if err != nil {
			return nil, fmt.Errorf("compute diff: %w", err)
		}

		if !opts.DryRun {
			if err := fileutil.WriteFileAtomic(sidecarPath, opts.NewContent, opts.NewMode); err != nil {
				return nil, fmt.Errorf("write sidecar %s: %w", sidecarPath, err)
			}
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
			Message: "devenv.nix status unknown; skipping update",
		}, nil

	default:
		return &NixUpdateResult{
			Action:  NixSkipped,
			Message: fmt.Sprintf("devenv.nix: unrecognized status %v; skipping", opts.Status),
		}, nil
	}
}
