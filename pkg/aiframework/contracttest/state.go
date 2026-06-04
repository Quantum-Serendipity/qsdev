package contracttest

import (
	"context"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
)

func TestStateBackend(t *testing.T, backend aiframework.StateBackend, fixtures ContractFixtures) {
	t.Helper()

	t.Run("ChronicleAppendNoError", func(t *testing.T) {
		entry := aiframework.ChronicleEntry{
			Timestamp: time.Now(),
			AgentID:   "test-agent",
			Verb:      aiframework.VerbTaskStarted,
			Target:    "test-target",
		}
		if err := backend.ChronicleAppend(context.Background(), entry); err != nil {
			t.Errorf("ChronicleAppend() error: %v", err)
		}
	})

	t.Run("ChronicleReadNoError", func(t *testing.T) {
		_, err := backend.ChronicleRead(context.Background(), aiframework.ChronicleQuery{Limit: 10})
		if err != nil {
			t.Errorf("ChronicleRead() error: %v", err)
		}
	})
}
