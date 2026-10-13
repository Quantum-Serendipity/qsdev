package doctor

import "example.com/s/internal/mcphealth"

// Integration tests may drive the probe directly.
func probeInTest() { mcphealth.CheckServer(mcphealth.ServerConfig{}) }
