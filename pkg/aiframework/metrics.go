package aiframework

import (
	"context"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/enumtext"
)

// MetricsProvider collects telemetry and health data for a framework.
type MetricsProvider interface {
	FrameworkID() FrameworkID
	EmitEvent(ctx context.Context, event MetricEvent) error
	CollectHealth(ctx context.Context) (*HealthReport, error)
	ContentRetention() ContentTier
}

// MetricEvent represents a single telemetry event emitted by a framework.
type MetricEvent struct {
	Timestamp   time.Time
	Action      string
	Category    string
	Severity    string
	ProjectRoot string
	FrameworkID FrameworkID
	Detail      map[string]any
}

// ContentTier controls how much detail is retained in event storage.
//
// The zero value is ContentUnknown, which never marshals; consumers must treat
// it as ContentMetadataOnly, so an unset tier never retains prompts or source.
type ContentTier int

const (
	ContentUnknown      ContentTier = iota // Unset; treat as metadata-only, never marshalled.
	ContentFull                            // All event data including prompts and source.
	ContentRedacted                        // Credential/secret values stripped.
	ContentMetadataOnly                    // Only timing, counts, and hashes.
)

var contentTierNames = [...]string{
	ContentUnknown:      "",
	ContentFull:         "full",
	ContentRedacted:     "redacted",
	ContentMetadataOnly: "metadata_only",
}

var contentTierText = enumtext.New[ContentTier]("ContentTier", "content tier", "unknown", contentTierNames[:])

func (c ContentTier) String() string { return contentTierText.String(c) }

func (c ContentTier) MarshalText() ([]byte, error) { return contentTierText.MarshalText(c) }

func (c *ContentTier) UnmarshalText(text []byte) error { return contentTierText.UnmarshalText(text, c) }

// HealthStatus summarises a framework's overall health.
//
// The zero value is StatusUnknown, which never marshals and must be treated
// as unhealthy, so a zero HealthReport never reads as healthy.
type HealthStatus int

const (
	StatusUnknown HealthStatus = iota // Unset; treat as unhealthy, never marshalled.
	StatusHealthy
	StatusDegraded
	StatusUnhealthy
)

var healthStatusNames = [...]string{
	StatusUnknown:   "",
	StatusHealthy:   "healthy",
	StatusDegraded:  "degraded",
	StatusUnhealthy: "unhealthy",
}

var healthStatusText = enumtext.New[HealthStatus]("HealthStatus", "health status", "unknown", healthStatusNames[:])

func (s HealthStatus) String() string { return healthStatusText.String(s) }

func (s HealthStatus) MarshalText() ([]byte, error) { return healthStatusText.MarshalText(s) }

func (s *HealthStatus) UnmarshalText(text []byte) error {
	return healthStatusText.UnmarshalText(text, s)
}

// CheckStatus represents the result of a single health check.
//
// The zero value is CheckUnknown, which never marshals and must be treated as
// a failure, so a zero HealthCheck never reads as passing.
type CheckStatus int

const (
	CheckUnknown CheckStatus = iota // Unset; treat as failed, never marshalled.
	CheckPass
	CheckFail
	CheckSkip
)

var checkStatusNames = [...]string{
	CheckUnknown: "",
	CheckPass:    "pass",
	CheckFail:    "fail",
	CheckSkip:    "skip",
}

var checkStatusText = enumtext.New[CheckStatus]("CheckStatus", "check status", "unknown", checkStatusNames[:])

func (c CheckStatus) String() string { return checkStatusText.String(c) }

func (c CheckStatus) MarshalText() ([]byte, error) { return checkStatusText.MarshalText(c) }

func (c *CheckStatus) UnmarshalText(text []byte) error { return checkStatusText.UnmarshalText(text, c) }

// HealthReport aggregates health checks for a framework.
type HealthReport struct {
	FrameworkID   FrameworkID
	Checks        []HealthCheck
	OverallStatus HealthStatus
}

// HealthCheck is a single diagnostic check within a HealthReport.
type HealthCheck struct {
	Name        string
	Status      CheckStatus
	Message     string
	Severity    string
	Remediation string
}
