package sys

// FallbackLogFileEnv is an environment variable that can be set to provide
// a fallback log file path for daemons started without systemd support.
//
// This will not be passed to the actual daemon.
const FallbackLogFileEnv = "__FALLBACK_LOG_FILE"
