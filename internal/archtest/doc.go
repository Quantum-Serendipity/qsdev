// Package archtest machine-checks the architecture rules that still have
// violations in the tree, against a checked-in baseline that may only shrink.
//
// The engine is stdlib-only: it parses every .go file in the module with
// go/parser, whatever its build constraints, so the result is the union over
// all GOOS values and the same baseline holds on linux, darwin and windows.
// Rules report violations keyed by rule, subject (an import edge or a
// package directory) and a per-package count; line numbers never appear, so
// moving code within a package does not churn baseline.txt.
//
// TestArchitecture requires the computed set to equal baseline.txt exactly.
// TestBaselineMonotone, run in CI with ARCHTEST_BASE pointing at the base
// branch's baseline, rejects any new entry or raised count. Regenerate the
// file after fixing sites with:
//
//	GOWORK=off go test ./internal/archtest -run TestArchitecture -update
package archtest
