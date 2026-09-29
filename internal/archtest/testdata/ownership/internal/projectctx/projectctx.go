package projectctx

import "example.com/own/internal/logging"

var markers = []string{".qsdev.yaml", ".qsdev"}

func Find(dir string) (string, bool) { return logging.WalkUp(dir, nil) }
