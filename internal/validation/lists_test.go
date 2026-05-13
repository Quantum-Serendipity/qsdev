package validation

import (
	"testing"
)

func TestLanguagesReturnsAtLeast27(t *testing.T) {
	langs := Languages()
	if len(langs) < 27 {
		t.Errorf("Languages() returned %d entries, want >= 27", len(langs))
	}
}

func TestCoreLanguagesReturns8(t *testing.T) {
	core := CoreLanguages()
	if len(core) != 8 {
		t.Errorf("CoreLanguages() returned %d entries, want 8", len(core))
	}
}

func TestCoreLanguagesAreSubsetOfLanguages(t *testing.T) {
	for _, lang := range CoreLanguages() {
		if !IsValidLanguage(lang) {
			t.Errorf("core language %q is not in Languages()", lang)
		}
	}
}

func TestServicesReturns6(t *testing.T) {
	svcs := Services()
	if len(svcs) != 6 {
		t.Errorf("Services() returned %d entries, want 6", len(svcs))
	}
}

func TestPermissionPresetsReturns4(t *testing.T) {
	presets := PermissionPresets()
	if len(presets) != 4 {
		t.Errorf("PermissionPresets() returned %d entries, want 4", len(presets))
	}
}

func TestHookPresetsReturns4(t *testing.T) {
	presets := HookPresets()
	if len(presets) != 4 {
		t.Errorf("HookPresets() returned %d entries, want 4", len(presets))
	}
}

func TestNoDuplicates(t *testing.T) {
	cases := []struct {
		name string
		list []string
	}{
		{"Languages", Languages()},
		{"CoreLanguages", CoreLanguages()},
		{"Services", Services()},
		{"PermissionPresets", PermissionPresets()},
		{"HookPresets", HookPresets()},
		{"NodePackageManagers", NodePackageManagers()},
		{"PythonPackageManagers", PythonPackageManagers()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			seen := make(map[string]bool, len(tc.list))
			for _, v := range tc.list {
				if seen[v] {
					t.Errorf("duplicate entry %q in %s", v, tc.name)
				}
				seen[v] = true
			}
		})
	}
}

func TestIsValidLanguage(t *testing.T) {
	if !IsValidLanguage("go") {
		t.Error("IsValidLanguage(\"go\") = false, want true")
	}
	if !IsValidLanguage("powershell") {
		t.Error("IsValidLanguage(\"powershell\") = false, want true")
	}
	if IsValidLanguage("cobol") {
		t.Error("IsValidLanguage(\"cobol\") = true, want false")
	}
	if IsValidLanguage("") {
		t.Error("IsValidLanguage(\"\") = true, want false")
	}
}

func TestIsValidService(t *testing.T) {
	if !IsValidService("postgres") {
		t.Error("IsValidService(\"postgres\") = false, want true")
	}
	if IsValidService("sqlite") {
		t.Error("IsValidService(\"sqlite\") = true, want false")
	}
}

func TestIsValidPermissionPreset(t *testing.T) {
	if !IsValidPermissionPreset("minimal") {
		t.Error("IsValidPermissionPreset(\"minimal\") = false, want true")
	}
	if IsValidPermissionPreset("admin") {
		t.Error("IsValidPermissionPreset(\"admin\") = true, want false")
	}
}

func TestIsValidHookPreset(t *testing.T) {
	if !IsValidHookPreset("safety-block") {
		t.Error("IsValidHookPreset(\"safety-block\") = false, want true")
	}
	if IsValidHookPreset("unknown") {
		t.Error("IsValidHookPreset(\"unknown\") = true, want false")
	}
}

func TestIsValidNodePackageManager(t *testing.T) {
	for _, pm := range []string{"npm", "pnpm", "yarn", "bun"} {
		if !IsValidNodePackageManager(pm) {
			t.Errorf("IsValidNodePackageManager(%q) = false, want true", pm)
		}
	}
	if IsValidNodePackageManager("bower") {
		t.Error("IsValidNodePackageManager(\"bower\") = true, want false")
	}
}

func TestIsValidPythonPackageManager(t *testing.T) {
	for _, pm := range []string{"pip", "uv", "poetry"} {
		if !IsValidPythonPackageManager(pm) {
			t.Errorf("IsValidPythonPackageManager(%q) = false, want true", pm)
		}
	}
	if IsValidPythonPackageManager("conda") {
		t.Error("IsValidPythonPackageManager(\"conda\") = true, want false")
	}
}

func TestListsReturnCopies(t *testing.T) {
	// Mutating the returned slice must not affect subsequent calls.
	langs1 := Languages()
	langs1[0] = "MUTATED"
	langs2 := Languages()
	if langs2[0] == "MUTATED" {
		t.Error("Languages() returned a reference to the internal slice, not a copy")
	}
}

func TestIsValidCoreLanguage(t *testing.T) {
	if !IsValidCoreLanguage("go") {
		t.Error("IsValidCoreLanguage(\"go\") = false, want true")
	}
	if IsValidCoreLanguage("php") {
		t.Error("IsValidCoreLanguage(\"php\") = true, want false")
	}
}
