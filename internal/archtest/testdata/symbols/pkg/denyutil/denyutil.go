package denyutil

// bashPattern compares a permission-rule tool prefix, not a hook tool name.
func bashPattern(tool string) bool { return tool == "Bash" }
