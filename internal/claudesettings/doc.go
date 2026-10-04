// Package claudesettings reads the Claude Code settings files of a project.
// It is the only home of their paths, of the key names qsdev inspects and of
// the exact-key parser, so every consumer (check, posture) judges the same
// view the agent runs with. It imports only the standard library.
package claudesettings
