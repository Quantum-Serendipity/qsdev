package devinit

// ecosystemGitignoreEntries maps ecosystem module names (EcosystemModule.Name(),
// the value stored in WizardAnswers.Languages[].Name) to their recommended
// .gitignore entries. These cover build artifacts, dependency directories, and
// secret files that should never be committed.
//
// Entries must never ignore files the ecosystem needs committed: Go's vendor/
// directory is deliberately absent (vendored modules are committed for
// reproducible -mod=vendor builds), and the Java build-wrapper jars are
// re-included after *.jar so ./gradlew and ./mvnw work on a fresh clone.
var ecosystemGitignoreEntries = map[string][]string{
	"javascript": {
		"node_modules/",
		"dist/",
		".env",
		".env.*",
		"*.pem",
		"*.key",
	},
	"python": {
		"__pycache__/",
		"*.py[cod]",
		"*.egg-info/",
		".venv/",
		"dist/",
		".env",
		".env.*",
		"*.pem",
		"*.key",
	},
	"go": {
		"*.exe",
		".env",
		".env.*",
		"*.pem",
		"*.key",
	},
	"rust": {
		"target/",
		".env",
		".env.*",
		"*.pem",
		"*.key",
	},
	"ruby": {
		"vendor/bundle/",
		".bundle/",
		"*.gem",
		".env",
		".env.*",
		"*.pem",
		"*.key",
	},
	"php": {
		"vendor/",
		".env",
		".env.*",
		"*.pem",
		"*.key",
	},
	"java": {
		"target/",
		"build/",
		"*.class",
		"*.jar",
		"!gradle/wrapper/gradle-wrapper.jar",
		"!.mvn/wrapper/maven-wrapper.jar",
		".env",
		".env.*",
		"*.pem",
		"*.key",
	},
	"dotnet": {
		"bin/",
		"obj/",
		"*.user",
		".env",
		".env.*",
		"*.pem",
		"*.key",
	},
	"elixir": {
		"_build/",
		"deps/",
		".env",
		".env.*",
		"*.pem",
		"*.key",
	},
	"dart": {
		".dart_tool/",
		"build/",
		".env",
		".env.*",
		"*.pem",
		"*.key",
	},
	"swift": {
		".build/",
		"Packages/",
		".env",
		".env.*",
		"*.pem",
		"*.key",
	},
	"haskell": {
		".stack-work/",
		"dist-newstyle/",
		".env",
		".env.*",
		"*.pem",
		"*.key",
	},
	// Terraform/OpenTofu: state and variable files hold plaintext secrets,
	// .terraform/ is the provider/module cache (with backend credentials in
	// .terraform/terraform.tfstate), and override files are local-only.
	// .terraform.lock.hcl is not matched and stays committed, and neither is
	// the .terraformrc the terraform module generates for the team.
	"terraform": {
		".terraform/",
		"*.tfstate",
		"*.tfstate.*",
		"*.tfvars",
		"*.tfvars.json",
		"crash.log",
		"crash.*.log",
		"override.tf",
		"override.tf.json",
		"*_override.tf",
		"*_override.tf.json",
		".env",
		".env.*",
		"*.pem",
		"*.key",
	},
	"zig": {
		"zig-cache/",
		"zig-out/",
		".env",
		".env.*",
		"*.pem",
		"*.key",
	},
}

// securityGitignoreEntries are always added regardless of ecosystem.
var securityGitignoreEntries = []string{
	".env",
	".env.*",
	"*.pem",
	"*.key",
}

// gitignoreEntriesForLanguages returns the combined .gitignore entries for
// the given language names. Security entries are always included. Duplicates
// are removed while preserving order.
func gitignoreEntriesForLanguages(languages []string) []string {
	seen := make(map[string]bool)
	var result []string

	add := func(entry string) {
		if !seen[entry] {
			seen[entry] = true
			result = append(result, entry)
		}
	}

	for _, lang := range languages {
		if entries, ok := ecosystemGitignoreEntries[lang]; ok {
			for _, e := range entries {
				add(e)
			}
		}
	}

	// Always include security entries even for unknown ecosystems.
	for _, e := range securityGitignoreEntries {
		add(e)
	}

	return result
}
