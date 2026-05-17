package fileutil

import "os"

const (
	ModeReadWrite  os.FileMode = 0o644
	ModeExecutable os.FileMode = 0o755
	ModePrivate    os.FileMode = 0o600
	ModeDirDefault os.FileMode = 0o755
)
