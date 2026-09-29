package pkgmanager

// Install command lines for the Windows package managers. They live in an
// untagged file so the non-Windows stubs describe the same commands the real
// Windows implementations run (e.g. for doctor recommendations).

// InstallArgs returns the command Install runs for one package; winget
// installs one package per invocation, so pass a single package.
func (w *Winget) InstallArgs(packages ...string) (string, []string) {
	args := append([]string{"install", "--id"}, packages...)
	return "winget", append(args, "-e", "--accept-source-agreements", "--accept-package-agreements")
}

// InstallArgs returns the command Install runs.
func (s *Scoop) InstallArgs(packages ...string) (string, []string) {
	return "scoop", append([]string{"install"}, packages...)
}

// InstallArgs returns the command Install runs.
func (c *Choco) InstallArgs(packages ...string) (string, []string) {
	return "choco", append([]string{"install", "-y"}, packages...)
}
