// Package projectctx is the single place qsdev decides which project a
// command acts on and where per-user files live.
//
// Resolve finds the project root with a bounded, trusted walk: only the
// project config file and the generated-state directory are markers, the walk
// never leaves the git toplevel or the device it started on, and markers an
// untrusted user could have planted are skipped. The resolved Context is
// carried on a context.Context (WithContext, FromContext) instead of being
// recomputed from the working directory. ProbeBoundary uses the same walk to
// bound version probes by the enclosing repository.
//
// WorkingDir and HomeDir are the only readers of the process working directory
// and home directory in the tree; UserDirs derives the per-user state and
// cache directories from them and the XDG environment.
//
// The package is a foundation leaf: it imports only the standard library and
// pkg/branding.
package projectctx
