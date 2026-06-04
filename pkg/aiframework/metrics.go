package aiframework

import (
	"context"
	"fmt"
	"time"
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
type ContentTier int

const (
	ContentFull         ContentTier = iota // All event data including prompts and source.
	ContentRedacted                        // Credential/secret values stripped.
	ContentMetadataOnly                    // Only timing, counts, and hashes.
)

var contentTierNames = [...]string{
	ContentFull:         "full",
	ContentRedacted:     "redacted",
	ContentMetadataOnly: "metadata_only",
}

func (c ContentTier) String() string {
	if int(c) >= 0 && int(c) < len(contentTierNames) {
		return contentTierNames[c]
	}
	return "unknown"
}

func (c ContentTier) MarshalText() ([]byte, error) {
	s := c.String()
	if s == "unknown" {
		return nil, fmt.Errorf("cannot marshal unknown ContentTier value %d", int(c))
	}
	return []byte(s), nil
}

func (c *ContentTier) UnmarshalText(text []byte) error {
	for i, name := range contentTierNames {
		if name == string(text) {
			*c = ContentTier(i)
			return nil
		}
	}
	return fmt.Errorf("unknown content tier: %q", string(text))
}

// HealthStatus summarises a framework's overall health.
type HealthStatus int

const (
	StatusHealthy HealthStatus = iota
	StatusDegraded
	StatusUnhealthy
)

var healthStatusNames = [...]string{
	StatusHealthy:   "healthy",
	StatusDegraded:  "degraded",
	StatusUnhealthy: "unhealthy",
}

func (s HealthStatus) String() string {
	if int(s) >= 0 && int(s) < len(healthStatusNames) {
		return healthStatusNames[s]
	}
	return "unknown"
}

func (s HealthStatus) MarshalText() ([]byte, error) {
	str := s.String()
	if str == "unknown" {
		return nil, fmt.Errorf("cannot marshal unknown HealthStatus value %d", int(s))
	}
	return []byte(str), nil
}

func (s *HealthStatus) UnmarshalText(text []byte) error {
	for i, name := range healthStatusNames {
		if name == string(text) {
			*s = HealthStatus(i)
			return nil
		}
	}
	return fmt.Errorf("unknown health status: %q", string(text))
}

// CheckStatus represents the result of a single health check.
type CheckStatus int

const (
	CheckPass CheckStatus = iota
	CheckFail
	CheckSkip
)

var checkStatusNames = [...]string{
	CheckPass: "pass",
	CheckFail: "fail",
	CheckSkip: "skip",
}

func (c CheckStatus) String() string {
	if int(c) >= 0 && int(c) < len(checkStatusNames) {
		return checkStatusNames[c]
	}
	return "unknown"
}

func (c CheckStatus) MarshalText() ([]byte, error) {
	s := c.String()
	if s == "unknown" {
		return nil, fmt.Errorf("cannot marshal unknown CheckStatus value %d", int(c))
	}
	return []byte(s), nil
}

func (c *CheckStatus) UnmarshalText(text []byte) error {
	for i, name := range checkStatusNames {
		if name == string(text) {
			*c = CheckStatus(i)
			return nil
		}
	}
	return fmt.Errorf("unknown check status: %q", string(text))
}

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
