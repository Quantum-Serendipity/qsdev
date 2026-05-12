package claudecode

import (
	"fmt"
	"strings"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/state"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
)

// UpdateSummary categorizes what changed during an update.
type UpdateSummary struct {
	SkillsUpdated    int
	SkillsAdded      int
	RulesUpdated     int
	RulesAdded       int
	TemplatesUpdated int
	VersionBump      bool
}

// BuildUpdateSummary compares previous state against newly generated files.
func BuildUpdateSummary(prevState types.GeneratedState, newFiles []types.GeneratedFile, vDiff VersionDiff) UpdateSummary {
	s := UpdateSummary{
		VersionBump: vDiff.NeedsUpdate(),
	}

	for _, f := range newFiles {
		hash := state.ComputeHash(f.Content)

		prev, existed := prevState.Files[f.Path]

		isSkill := strings.HasPrefix(f.Path, ".claude/skills/")
		isRule := strings.HasPrefix(f.Path, ".claude/rules/")

		if !existed {
			switch {
			case isSkill:
				s.SkillsAdded++
			case isRule:
				s.RulesAdded++
			default:
				s.TemplatesUpdated++
			}
			continue
		}

		if prev.Hash != hash {
			switch {
			case isSkill:
				s.SkillsUpdated++
			case isRule:
				s.RulesUpdated++
			default:
				s.TemplatesUpdated++
			}
		}
	}

	return s
}

// String returns the summary message.
func (s UpdateSummary) String() string {
	if s.SkillsUpdated == 0 && s.SkillsAdded == 0 &&
		s.RulesUpdated == 0 && s.RulesAdded == 0 &&
		s.TemplatesUpdated == 0 {
		return "All files up to date."
	}

	var parts []string

	if s.SkillsUpdated > 0 {
		parts = append(parts, fmt.Sprintf("updated %d skill(s)", s.SkillsUpdated))
	}
	if s.SkillsAdded > 0 {
		parts = append(parts, fmt.Sprintf("added %d skill(s)", s.SkillsAdded))
	}
	if s.RulesUpdated > 0 {
		parts = append(parts, fmt.Sprintf("updated %d rule(s)", s.RulesUpdated))
	}
	if s.RulesAdded > 0 {
		parts = append(parts, fmt.Sprintf("added %d rule(s)", s.RulesAdded))
	}
	if s.TemplatesUpdated > 0 {
		parts = append(parts, fmt.Sprintf("refreshed %d template(s)", s.TemplatesUpdated))
	}

	msg := strings.Join(parts, ", ")
	// Capitalize first letter.
	if len(msg) > 0 {
		msg = strings.ToUpper(msg[:1]) + msg[1:]
	}
	return msg + "."
}
