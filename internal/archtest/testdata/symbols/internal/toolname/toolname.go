package toolname

// Gate compares tool names to the literal "Bash" three ways; the string in
// this comment, the slice element and the plain assignment are not
// comparisons.
func Gate(tool string) bool {
	tools := []string{"Bash"}
	name := "Bash"
	if tool == "Bash" || "Bash" != tool {
		return true
	}
	switch tool {
	case "Read", "Bash":
		return true
	case "PowerShell":
		return false
	}
	return tool == "Monitor" && len(tools) > 0 && name != ""
}
