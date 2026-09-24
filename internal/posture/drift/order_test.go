package drift

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestDetect_DeterministicOrder guards against map iteration leaking into the
// report: several detectors range over maps, so without sorting two runs over
// an unchanged project produce findings in different orders.
func TestDetect_DeterministicOrder(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	files := make(map[string]types.FileState)
	for i := range 20 {
		name := fmt.Sprintf("file-%02d.txt", i)
		writeFile(t, filepath.Join(dir, name), "edited\n")
		files[name] = types.FileState{Hash: "stale", Strategy: types.Overwrite}
	}
	writeFile(t, filepath.Join(dir, "CLAUDE.md"),
		"<!-- qsdev:b -->\n<!-- qsdev:a -->\n<!-- qsdev:c -->\n<!-- /qsdev:d -->\n")
	genState := types.GeneratedState{QsdevVersion: "dev", Files: files}

	first := Detect(dir, genState, nil)
	for i := range 10 {
		again := Detect(dir, genState, nil)
		if !reflect.DeepEqual(first.Categories, again.Categories) {
			t.Fatalf("run %d produced a different finding order", i+1)
		}
	}

	for _, cat := range first.Categories {
		for i := 1; i < len(cat.Findings); i++ {
			if cat.Findings[i-1].Subject > cat.Findings[i].Subject {
				t.Errorf("%s: %q ordered before %q", cat.Name, cat.Findings[i-1].Subject, cat.Findings[i].Subject)
			}
		}
	}
}
