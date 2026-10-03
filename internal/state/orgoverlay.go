package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// OrgOverlayFile returns the path, relative to the project root, of the file
// that records which org overlay a human approved for the project (see
// catalog.OrgConfigPin). It lives in the state directory, which is local to
// the checkout and which the self-protection hook keeps the agent from
// changing.
func OrgOverlayFile() string {
	b := branding.Get()
	return b.StateDir + "/." + b.AppName + "-org-overlay.yaml"
}

// orgOverlayRecord is the content of OrgOverlayFile.
type orgOverlayRecord struct {
	// Path is the overlay path the CLI resolved when the record was made, or
	// "" when it resolved none.
	Path string `yaml:"path"`
}

// SaveOrgOverlay records path as the project's approved org overlay
// ("" for none), replacing any earlier record.
func SaveOrgOverlay(projectRoot, path string) error {
	data, err := yaml.Marshal(orgOverlayRecord{Path: path})
	if err != nil {
		return fmt.Errorf("rendering %s: %w", OrgOverlayFile(), err)
	}
	if err := fileutil.WriteFileAtomicInRoot(projectRoot, OrgOverlayFile(), data, fileutil.ModeReadWrite); err != nil {
		return fmt.Errorf("writing %s: %w", OrgOverlayFile(), err)
	}
	return nil
}

// LoadOrgOverlay returns the org overlay path recorded for the project at
// projectRoot (see SaveOrgOverlay) and true, or false when none is recorded.
func LoadOrgOverlay(projectRoot string) (string, bool, error) {
	data, err := os.ReadFile(filepath.Join(projectRoot, filepath.FromSlash(OrgOverlayFile())))
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("reading %s: %w", OrgOverlayFile(), err)
	}
	var rec orgOverlayRecord
	if err := yaml.Unmarshal(data, &rec); err != nil {
		return "", false, fmt.Errorf("parsing %s: %w", OrgOverlayFile(), err)
	}
	return rec.Path, true, nil
}
