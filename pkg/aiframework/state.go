package aiframework

import (
	"context"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/enumtext"
)

// StateBackend persists chronicle entries and task state for multi-agent coordination.
//
// Experimental: no production backend exists yet (the Claude Code adapter's
// chronicle methods are no-ops and its task methods return "not yet
// implemented") and the task lifecycle is
// incomplete; see the package documentation. The interface may change without
// notice when multi-agent coordination is implemented.
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
//
// Experimental: only the non-terminal states exist. A task's outcome is
// reported through StateBackend.TaskComplete, but TaskInfo.Status has no
// completed, failed or cancelled value to reflect it, and no transition rules
// are defined. Both will be settled with the first real StateBackend.
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

var taskStatusText = enumtext.New[TaskStatus]("TaskStatus", "task status", "unknown", taskStatusNames[:])

func (s TaskStatus) String() string { return taskStatusText.String(s) }

func (s TaskStatus) MarshalText() ([]byte, error) { return taskStatusText.MarshalText(s) }

func (s *TaskStatus) UnmarshalText(text []byte) error { return taskStatusText.UnmarshalText(text, s) }

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

var taskOutcomeText = enumtext.New[TaskOutcome]("TaskOutcome", "task outcome", "unknown", taskOutcomeNames[:])

func (o TaskOutcome) String() string { return taskOutcomeText.String(o) }

func (o TaskOutcome) MarshalText() ([]byte, error) { return taskOutcomeText.MarshalText(o) }

func (o *TaskOutcome) UnmarshalText(text []byte) error { return taskOutcomeText.UnmarshalText(text, o) }

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
