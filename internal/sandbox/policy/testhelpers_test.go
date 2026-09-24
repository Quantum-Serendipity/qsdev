package policy

// testProjectDir is the project directory the policy tests pass to
// ToSandboxConfig and ValidateMountDecl. It does not exist, so no symlink
// resolution applies to paths under it.
//
// It lives in an unconstrained file because both the Unix-only path
// validation tests (validate_test.go, defaults_test.go) and the portable
// ToSandboxConfig tests (tosandboxconfig_test.go), which also build on
// Windows, reference it.
const testProjectDir = "/srv/qsdev-policy-test/project"
