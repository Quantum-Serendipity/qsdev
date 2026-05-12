package claudecode

// VersionDiff describes what changed between stored and current versions.
type VersionDiff struct {
	TemplateChanged     bool
	SkillLibraryChanged bool
	StoredTemplateVer   string
	CurrentTemplateVer  string
	StoredSkillLibVer   string
	CurrentSkillLibVer  string
}

// CompareVersions compares stored GeneratedState versions against the
// current embedded template versions.
func CompareVersions(storedTemplateVer, storedSkillLibVer string) VersionDiff {
	currentTemplate := ComputeTemplateVersion()
	currentSkillLib := ComputeSkillLibraryVersion()

	return VersionDiff{
		TemplateChanged:     storedTemplateVer != currentTemplate,
		SkillLibraryChanged: storedSkillLibVer != currentSkillLib,
		StoredTemplateVer:   storedTemplateVer,
		CurrentTemplateVer:  currentTemplate,
		StoredSkillLibVer:   storedSkillLibVer,
		CurrentSkillLibVer:  currentSkillLib,
	}
}

// NeedsUpdate returns true if any version changed.
func (d VersionDiff) NeedsUpdate() bool {
	return d.TemplateChanged || d.SkillLibraryChanged
}

// IsLibrarySkill returns true if the skill name is in the embedded manifest.
func IsLibrarySkill(name string) bool {
	manifest, err := loadManifest()
	if err != nil {
		return false
	}
	for _, s := range manifest.Skills {
		if s.Name == name {
			return true
		}
	}
	return false
}

// IsLibraryRule returns true if the rule filename matches a known library rule.
func IsLibraryRule(filename string) bool {
	if filename == "security-rules.md" {
		return true
	}
	for _, rules := range languageToRules {
		for _, r := range rules {
			if r == filename {
				return true
			}
		}
	}
	return false
}
