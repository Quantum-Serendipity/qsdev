package doctor

import health "example.com/s/internal/mcphealth"

// Probe bypasses mcpregistry's trust gate.
func Probe() {
	cfg := health.ProbeTarget(health.ServerConfig{})
	health.CheckServer(cfg)
	health.CheckAll(nil)
	run := health.CheckServer // a method value is a reference too
	run(cfg)
}
