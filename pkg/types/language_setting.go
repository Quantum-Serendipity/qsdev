package types

import (
	"slices"
	"strings"
)

// Setting keys that name a dedicated LanguageChoice field rather than an
// extra. Every other key names an entry of LanguageChoice.Extras, which
// reaches the ecosystem module as ModuleConfig.Extras[key].
const (
	// SettingVersion names LanguageChoice.Version (ModuleConfig.Version).
	SettingVersion = "version"
	// SettingPackageManager names LanguageChoice.PackageManager
	// (ModuleConfig.PackageManager).
	SettingPackageManager = "package_manager"
)

// Setting returns the value lc holds for key and whether it is set. The
// SettingVersion and SettingPackageManager keys read their dedicated fields;
// any other key reads the extra of that name, where a bare "key" flag means
// "true" and, as in ModuleConfig.Extra, an empty value counts as unset. When
// an extra appears more than once the last entry wins, matching how the
// extras are turned into a map.
func (lc LanguageChoice) Setting(key string) (string, bool) {
	switch key {
	case SettingVersion:
		return lc.Version, lc.Version != ""
	case SettingPackageManager:
		return lc.PackageManager, lc.PackageManager != ""
	}
	value := ""
	for _, e := range lc.Extras {
		k, v, hasValue := strings.Cut(e, "=")
		if k != key {
			continue
		}
		if !hasValue {
			v = "true"
		}
		value = v
	}
	return value, value != ""
}

// WithSetting returns lc with key set to value: the dedicated field for
// SettingVersion and SettingPackageManager, otherwise a "key=value" extra
// that replaces every existing entry (bare or "key=value") for key, in the
// position of the first one. lc's Extras slice is never modified in place.
func (lc LanguageChoice) WithSetting(key, value string) LanguageChoice {
	switch key {
	case SettingVersion:
		lc.Version = value
		return lc
	case SettingPackageManager:
		lc.PackageManager = value
		return lc
	}
	entry := key + "=" + value
	extras := make([]string, 0, len(lc.Extras)+1)
	replaced := false
	for _, e := range lc.Extras {
		if extraKey(e) != key {
			extras = append(extras, e)
			continue
		}
		if !replaced {
			extras = append(extras, entry)
			replaced = true
		}
	}
	if !replaced {
		extras = append(extras, entry)
	}
	lc.Extras = slices.Clip(extras)
	return lc
}
