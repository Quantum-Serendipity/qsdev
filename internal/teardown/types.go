package teardown

// Profile selects the scope of a teardown operation.
type Profile string

const (
	// ProfileQuick removes state directories only (.devinit/).
	// No files are removed or cleaned.
	ProfileQuick Profile = "quick"

	// ProfileDefault removes state directories, unmodified exclusive files,
	// and surgically cleans shared files. Modified files are preserved.
	ProfileDefault Profile = "default"

	// ProfileCompliance generates a final posture report and archives all
	// managed files before performing the default teardown.
	ProfileCompliance Profile = "compliance"
)

// TeardownOptions configures a teardown operation.
type TeardownOptions struct {
	Profile     Profile
	Force       bool
	Archive     bool
	DryRun      bool
	ProjectRoot string
}

// FileAction describes a single file operation in the teardown plan.
type FileAction struct {
	Path     string
	Reason   string
	Modified bool
}

// TeardownPlan describes the operations a teardown will perform.
type TeardownPlan struct {
	Remove   []FileAction // Exclusive files to delete entirely.
	Preserve []FileAction // Modified files to keep.
	Clean    []FileAction // Shared files to surgically clean.
	Dirs     []string     // Directories to remove.
	Profile  Profile
}

// TeardownResult describes what a teardown actually did.
type TeardownResult struct {
	Removed     []FileAction
	Preserved   []FileAction
	Cleaned     []FileAction
	DirsRemoved []string
	ArchivePath string
	ReportPath  string
	Errors      []error
}
