package outdated

var ecosystemCommands = []EcosystemCommand{
	{Ecosystem: "javascript", Binary: "npm", Args: []string{"outdated"}, OutdatedOnExit1: true},
	{Ecosystem: "javascript", Binary: "pnpm", Args: []string{"outdated"}, OutdatedOnExit1: true},
	{Ecosystem: "javascript", Binary: "yarn", Args: []string{"outdated"}, OutdatedOnExit1: true},
	{Ecosystem: "python", Binary: "pip", Args: []string{"list", "--outdated"}, OutdatedOnExit1: false},
	{Ecosystem: "python", Binary: "uv", Args: []string{"pip", "list", "--outdated"}, OutdatedOnExit1: false},
	{Ecosystem: "go", Binary: "go", Args: []string{"list", "-m", "-u", "all"}, OutdatedOnExit1: false},
	{Ecosystem: "rust", Binary: "cargo", Args: []string{"outdated"}, OutdatedOnExit1: true},
	{Ecosystem: "dotnet", Binary: "dotnet", Args: []string{"list", "package", "--outdated"}, OutdatedOnExit1: false},
	{Ecosystem: "ruby", Binary: "bundle", Args: []string{"outdated"}, OutdatedOnExit1: true},
	{Ecosystem: "php", Binary: "composer", Args: []string{"outdated"}, OutdatedOnExit1: true},
	{Ecosystem: "elixir", Binary: "mix", Args: []string{"hex.outdated"}, OutdatedOnExit1: false},
	{Ecosystem: "java", Binary: "mvn", Args: []string{"versions:display-dependency-updates"}, OutdatedOnExit1: false},
	{Ecosystem: "java", Binary: "gradle", Args: []string{"dependencyUpdates"}, OutdatedOnExit1: false},
}

// CommandsForEcosystem returns matching commands for the given ecosystem.
// For ecosystems with multiple package managers (e.g., javascript has npm/pnpm/yarn),
// returns all — caller should pick the first whose binary is available.
func CommandsForEcosystem(eco string) []EcosystemCommand {
	var result []EcosystemCommand
	for _, cmd := range ecosystemCommands {
		if cmd.Ecosystem == eco {
			result = append(result, cmd)
		}
	}
	return result
}

// SupportedEcosystems returns the list of ecosystem names that have outdated commands.
func SupportedEcosystems() []string {
	seen := make(map[string]bool)
	var result []string
	for _, cmd := range ecosystemCommands {
		if !seen[cmd.Ecosystem] {
			seen[cmd.Ecosystem] = true
			result = append(result, cmd.Ecosystem)
		}
	}
	return result
}
