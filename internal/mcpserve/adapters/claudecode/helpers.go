package claudecode

// optionalStringSchema builds an object schema with a single optional string
// property of the given name and description.
func optionalStringSchema(name, desc string) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			name: map[string]any{"type": "string", "description": desc},
		},
	}
}

// optionalBoolSchema builds an object schema with a single optional boolean
// property of the given name and description.
func optionalBoolSchema(name, desc string) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			name: map[string]any{"type": "boolean", "description": desc},
		},
	}
}

// stringArg extracts an optional string argument, defaulting to "" when absent
// or of the wrong type.
func stringArg(args map[string]any, name string) string {
	v, _ := args[name].(string)
	return v
}

// boolArg extracts an optional boolean argument, defaulting to false when absent
// or of the wrong type.
func boolArg(args map[string]any, name string) bool {
	v, _ := args[name].(bool)
	return v
}
