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
