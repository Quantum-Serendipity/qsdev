package mcpregistry

import "example.com/s/internal/mcphealth"

// ProbeAll is the owner: it may start and dial configured servers.
func ProbeAll(cfgs []mcphealth.ServerConfig) {
	_ = mcphealth.ProbeTarget(mcphealth.ServerConfig{})
	mcphealth.CheckAll(cfgs)
}
