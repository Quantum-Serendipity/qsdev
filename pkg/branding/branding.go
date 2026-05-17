package branding

import (
	"sync"
	"sync/atomic"
)

type Config struct {
	AppName       string
	ConfigFile    string
	LocalConfig   string
	StateDir      string
	EnvLogVar     string
	EnvLogDirVar  string
	EnvNoUpdate   string
	EnvPrefix     string
	LogFilePrefix string
	TempPrefix    string
	GitHubOwner   string
	GitHubRepo    string
}

func Default() Config {
	return Config{
		AppName:       "qsdev",
		ConfigFile:    ".qsdev.yaml",
		LocalConfig:   ".qsdev.local.yaml",
		StateDir:      ".devinit",
		EnvLogVar:     "QSDEV_LOG",
		EnvLogDirVar:  "QSDEV_LOG_DIR",
		EnvNoUpdate:   "QSDEV_NO_UPDATE_CHECK",
		EnvPrefix:     "QSDEV_",
		LogFilePrefix: "qsdev-",
		TempPrefix:    ".qsdev-tmp-",
		GitHubOwner:   "Quantum-Serendipity",
		GitHubRepo:    "qsdev",
	}
}

var (
	mu     sync.Mutex
	active = Default()
	sealed atomic.Bool
)

func Set(cfg Config) {
	mu.Lock()
	defer mu.Unlock()
	if sealed.Load() {
		panic("branding: Set called after Get; branding must be configured before first use")
	}
	if cfg.AppName != "" {
		active.AppName = cfg.AppName
	}
	if cfg.ConfigFile != "" {
		active.ConfigFile = cfg.ConfigFile
	}
	if cfg.LocalConfig != "" {
		active.LocalConfig = cfg.LocalConfig
	}
	if cfg.StateDir != "" {
		active.StateDir = cfg.StateDir
	}
	if cfg.EnvLogVar != "" {
		active.EnvLogVar = cfg.EnvLogVar
	}
	if cfg.EnvLogDirVar != "" {
		active.EnvLogDirVar = cfg.EnvLogDirVar
	}
	if cfg.EnvNoUpdate != "" {
		active.EnvNoUpdate = cfg.EnvNoUpdate
	}
	if cfg.EnvPrefix != "" {
		active.EnvPrefix = cfg.EnvPrefix
	}
	if cfg.LogFilePrefix != "" {
		active.LogFilePrefix = cfg.LogFilePrefix
	}
	if cfg.TempPrefix != "" {
		active.TempPrefix = cfg.TempPrefix
	}
	if cfg.GitHubOwner != "" {
		active.GitHubOwner = cfg.GitHubOwner
	}
	if cfg.GitHubRepo != "" {
		active.GitHubRepo = cfg.GitHubRepo
	}
}

func Get() Config {
	sealed.Store(true)
	return active
}
