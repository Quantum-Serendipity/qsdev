package cigeneration

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// WorkflowPin is one SHA-pinned action reference found in a workflow file.
type WorkflowPin struct {
	SHA  string
	Tag  string
	File string // base name of the first workflow that pins it
}

// usesRe matches a SHA-pinned action reference in a workflow line, e.g.
//
//	uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2
//
// The action path is captured whole, so subpath actions such as
// google/osv-scanner-action/osv-scanner-action compare correctly. The tag
// comment is optional in the pattern so a pin without one is reported rather
// than skipped.
var usesRe = regexp.MustCompile(`uses:\s+(\S+)@([0-9a-f]{40})(?:\s+#\s*(\S+))?`)

// ParseWorkflowPins collects every SHA-pinned action used by the .yml and
// .yaml workflow files in dir, keyed by action path (owner/repo[/subpath]).
//
// A pin without a "# tag" comment is an error: nothing could record which
// version it is. So is one action pinned to different commits in different
// files, since no single catalog entry could match both.
func ParseWorkflowPins(dir string) (map[string]WorkflowPin, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading workflow directory: %w", err)
	}

	pins := make(map[string]WorkflowPin)
	for _, e := range entries {
		if e.IsDir() || !isWorkflowFile(e.Name()) {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading workflow: %w", err)
		}
		if err := addFilePins(pins, e.Name(), b); err != nil {
			return nil, err
		}
	}
	return pins, nil
}

// maxWorkflowLine bounds one workflow line; long inline run scripts exceed
// bufio's 64 KiB default.
const maxWorkflowLine = 1 << 20

func isWorkflowFile(name string) bool {
	ext := filepath.Ext(name)
	return ext == ".yml" || ext == ".yaml"
}

// addFilePins adds the pins in one workflow file to pins.
func addFilePins(pins map[string]WorkflowPin, file string, content []byte) error {
	sc := bufio.NewScanner(bytes.NewReader(content))
	sc.Buffer(nil, maxWorkflowLine)
	for line := 1; sc.Scan(); line++ {
		m := usesRe.FindStringSubmatch(sc.Text())
		if m == nil {
			continue
		}
		path, pin := m[1], WorkflowPin{SHA: m[2], Tag: m[3], File: file}
		if pin.Tag == "" {
			return fmt.Errorf("%s:%d: %s@%s has no \"# tag\" comment", file, line, path, pin.SHA)
		}
		prev, seen := pins[path]
		if !seen {
			pins[path] = pin
			continue
		}
		if prev.SHA != pin.SHA || prev.Tag != pin.Tag {
			return fmt.Errorf("%s:%d: %s pinned @%s (%s) but %s pins @%s (%s)",
				file, line, path, pin.SHA, pin.Tag, prev.File, prev.SHA, prev.Tag)
		}
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("scanning %s: %w", file, err)
	}
	return nil
}
