package claudecode

import (
	"context"
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
)

func (a *Adapter) ChronicleAppend(_ context.Context, _ aiframework.ChronicleEntry) error {
	return nil
}

func (a *Adapter) ChronicleRead(_ context.Context, _ aiframework.ChronicleQuery) ([]aiframework.ChronicleEntry, error) {
	return nil, nil
}

func (a *Adapter) TaskCreate(_ context.Context, _ aiframework.TaskSpec) (string, error) {
	return "", fmt.Errorf("task management not yet implemented")
}

func (a *Adapter) TaskClaim(_ context.Context, _ string) error {
	return fmt.Errorf("task management not yet implemented")
}

func (a *Adapter) TaskUpdate(_ context.Context, _ string, _ aiframework.TaskStatus, _ string) error {
	return fmt.Errorf("task management not yet implemented")
}

func (a *Adapter) TaskComplete(_ context.Context, _ string, _ aiframework.TaskOutcome) error {
	return fmt.Errorf("task management not yet implemented")
}

func (a *Adapter) TaskList(_ context.Context, _ aiframework.TaskFilter) ([]aiframework.TaskInfo, error) {
	return nil, nil
}
