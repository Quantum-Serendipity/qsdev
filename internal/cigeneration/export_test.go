package cigeneration

// CheckWorkflowActions exposes checkWorkflowActions to the external test
// package, which checks generated workflows (internal/profile imports this
// package, so the check cannot run from an internal test).
var CheckWorkflowActions = checkWorkflowActions
