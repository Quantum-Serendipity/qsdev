package aiframework

import (
	"context"
	"fmt"
	"time"
)

// StateBackend persists chronicle entries and task state for multi-agent coordination.
type StateBackend interface {
	ChronicleAppend(ctx context.Context, entry ChronicleEntry) error
	ChronicleRead(ctx context.Context, opts ChronicleQuery) ([]ChronicleEntry, error)
	TaskCreate(ctx context.Context, task TaskSpec) (string, error)
	TaskClaim(ctx context.Context, taskID string) error
	TaskUpdate(ctx context.Context, taskID string, status TaskStatus, note string) error
	TaskComplete(ctx context.Context, taskID string, outcome TaskOutcome) error
	TaskList(ctx context.Context, filter TaskFilter) ([]TaskInfo, error)
}

// ChronicleVerb identifies the kind of action recorded in a chronicle entry.
type ChronicleVerb string

const (
	VerbTaskStarted     ChronicleVerb = "task_started"
	VerbTaskCompleted   ChronicleVerb = "task_completed"
	VerbTaskFailed      ChronicleVerb = "task_failed"
	VerbTaskBlocked     ChronicleVerb = "task_blocked"
	VerbWorktreeCreated ChronicleVerb = "worktree_created"
	VerbWorktreeCleaned ChronicleVerb = "worktree_cleaned"
	VerbFileCreated     ChronicleVerb = "file_created"
	VerbFileModified    ChronicleVerb = "file_modified"
	VerbTestPassed      ChronicleVerb = "test_passed"
	VerbTestFailed      ChronicleVerb = "test_failed"
	VerbReviewRequested ChronicleVerb = "review_requested"
	VerbReviewCompleted ChronicleVerb = "review_completed"
	VerbLessonFiled     ChronicleVerb = "lesson_filed"
)

var validVerbs = map[ChronicleVerb]bool{
	VerbTaskStarted: true, VerbTaskCompleted: true, VerbTaskFailed: true,
	VerbTaskBlocked: true, VerbWorktreeCreated: true, VerbWorktreeCleaned: true,
	VerbFileCreated: true, VerbFileModified: true, VerbTestPassed: true,
	VerbTestFailed: true, VerbReviewRequested: true, VerbReviewCompleted: true,
	VerbLessonFiled: true,
}

// ValidChronicleVerb reports whether v is a recognized chronicle verb.
func ValidChronicleVerb(v ChronicleVerb) bool {
	return validVerbs[v]
}

// ChronicleEntry is a single timestamped record in the chronicle log.
type ChronicleEntry struct {
	Timestamp time.Time
	AgentID   string
	Verb      ChronicleVerb
	Target    string
	Note      string
}

// ChronicleQuery constrains which chronicle entries are returned.
type ChronicleQuery struct {
	Since      time.Time
	VerbFilter []ChronicleVerb
	Limit      int
}

// TaskStatus tracks a task's lifecycle state.
type TaskStatus int

const (
	TaskOpen TaskStatus = iota
	TaskAssigned
	TaskInProgress
	TaskBlocked
)

var taskStatusNames = [...]string{
	TaskOpen:       "open",
	TaskAssigned:   "assigned",
	TaskInProgress: "in_progress",
	TaskBlocked:    "blocked",
}

func (s TaskStatus) String() string {
	if int(s) >= 0 && int(s) < len(taskStatusNames) {
		return taskStatusNames[s]
	}
	return "unknown"
}

func (s TaskStatus) MarshalText() ([]byte, error) {
	str := s.String()
	if str == "unknown" {
		return nil, fmt.Errorf("cannot marshal unknown TaskStatus value %d", int(s))
	}
	return []byte(str), nil
}

func (s *TaskStatus) UnmarshalText(text []byte) error {
	for i, name := range taskStatusNames {
		if name == string(text) {
			*s = TaskStatus(i)
			return nil
		}
	}
	return fmt.Errorf("unknown task status: %q", string(text))
}

// TaskOutcome records how a completed task finished.
type TaskOutcome int

const (
	OutcomeSuccess TaskOutcome = iota
	OutcomePartial
	OutcomeFailure
)

var taskOutcomeNames = [...]string{
	OutcomeSuccess: "success",
	OutcomePartial: "partial",
	OutcomeFailure: "failure",
}

func (o TaskOutcome) String() string {
	if int(o) >= 0 && int(o) < len(taskOutcomeNames) {
		return taskOutcomeNames[o]
	}
	return "unknown"
}

func (o TaskOutcome) MarshalText() ([]byte, error) {
	s := o.String()
	if s == "unknown" {
		return nil, fmt.Errorf("cannot marshal unknown TaskOutcome value %d", int(o))
	}
	return []byte(s), nil
}

func (o *TaskOutcome) UnmarshalText(text []byte) error {
	for i, name := range taskOutcomeNames {
		if name == string(text) {
			*o = TaskOutcome(i)
			return nil
		}
	}
	return fmt.Errorf("unknown task outcome: %q", string(text))
}

// ValidTaskTransition reports whether moving from one status to another is allowed.
// Valid transitions: open->assigned, assigned->in_progress, in_progress->blocked,
// blocked->in_progress.
func ValidTaskTransition(from, to TaskStatus) bool {
	switch {
	case from == TaskOpen && to == TaskAssigned:
		return true
	case from == TaskAssigned && to == TaskInProgress:
		return true
	case from == TaskInProgress && to == TaskBlocked:
		return true
	case from == TaskBlocked && to == TaskInProgress:
		return true
	default:
		return false
	}
}

// TaskSpec describes a new task to create.
type TaskSpec struct {
	Title       string
	Description string
	Priority    int
	Labels      []string
}

// TaskInfo is the full state of a task, including metadata.
type TaskInfo struct {
	ID         string
	Spec       TaskSpec
	Status     TaskStatus
	AssignedTo string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Notes      []string
}

// TaskFilter constrains which tasks are returned by TaskList.
type TaskFilter struct {
	Status     *TaskStatus
	AssignedTo string
	Limit      int
}
