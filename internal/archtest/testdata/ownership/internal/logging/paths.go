package logging

func WalkUp(dir string, match func(string) bool) (string, bool) { return dir, false }

func Find(dir string) (string, bool) { return WalkUp(dir, nil) }
