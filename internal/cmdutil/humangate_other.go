//go:build !linux && !darwin

package cmdutil

// parentPID reports the process ancestry as unknown: the gate relies on the
// terminal challenge here (Windows included).
func parentPID(int) (int, error) { return 0, errAncestryUnknown }

// processEnviron reports the process ancestry as unknown.
func processEnviron(int) ([]string, error) { return nil, errAncestryUnknown }

// processProgram reports the process ancestry as unknown.
func processProgram(int) (string, []string, error) { return "", nil, errAncestryUnknown }
