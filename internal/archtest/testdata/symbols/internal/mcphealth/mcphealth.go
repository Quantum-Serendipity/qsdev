package mcphealth

// ServerConfig is a stub.
type ServerConfig struct{}

// CheckServer is a stub probe entry point.
func CheckServer(cfg ServerConfig) {}

// CheckAll calls CheckServer unqualified inside the defining package.
func CheckAll(cfgs []ServerConfig) {
	for _, c := range cfgs {
		CheckServer(ProbeTarget(c))
	}
}

// ProbeTarget is a stub.
func ProbeTarget(cfg ServerConfig) ServerConfig { return cfg }
