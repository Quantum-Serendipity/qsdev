package consumer

import (
	"path/filepath"

	"example.com/own/internal/logging"
)

// ConsumerSeverity is a second severity enum. Mentioning "settings.json" or
// go.sum in a comment is not a violation.
type ConsumerSeverity int

// The following are not severity enums: structs, aliases, unexported types.
type (
	CVSSSeverity  struct{ Score float64 }
	AliasSeverity = int
	levelSeverity string
	osvSeverity   struct {
		Type string `json:"settings.json"`
	}
)

func Paths(root string) []string {
	_, _ = logging.WalkUp(root, nil)
	return []string{
		filepath.Join(root, "go.sum"),
		"fixture.lock",
		"untracked.lock",
		"fixture",
		"%s/.claude/settings.json",
		"parsing settings.json: %w",
		"settings.local.json",
		".qsdev.yaml",
		"~/.qsdev/bin",
		".qsdevx",
		`go.sum.bak`,
	}
}
