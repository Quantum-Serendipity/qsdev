package devinit

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/surgery"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// runEnable enables a tool: validates prerequisites, generates files, and
// updates persisted answers and state.
func runEnable(cmd *cobra.Command, toolName string, opts enableOptions) error {
	projectRoot, err := cmdutil.ProjectRoot()
	if err != nil {
		return err
	}

	answers, tool, err := loadToolForEnable(projectRoot, toolName)
	if err != nil {
		return err
	}

	// Already enabled — no-op.
	if answers.EnabledTools[toolName] {
		fmt.Fprintf(cmd.OutOrStdout(), "Tool %q is already enabled.\n", toolName)
		return nil
	}

	// Validate prerequisites and conflicts.
	if err := toolreg.ValidateEnable(toolreg.DefaultRegistry(), toolName, answers.EnabledTools); err != nil {
		return err
	}

	// Call the tool's enable function to update answers.
	if tool.EnableFunc != nil {
		tool.EnableFunc(&answers)
	}
	answers.EnabledTools[toolName] = true

	if opts.DryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would enable %q.\n", tool.DisplayName)
		printOwnedFiles(cmd, tool, "would write")
		return nil
	}

	writtenFiles, err := writeToolFiles(tool, toolName, projectRoot, answers)
	if err != nil {
		return err
	}

	stateFile := filepath.Join(projectRoot, stateFilePath())
	if err := saveEnableState(stateFile, toolName, writtenFiles); err != nil {
		return err
	}

	// Save updated answers.
	if err := saveAnswers(projectRoot, answers); err != nil {
		return fmt.Errorf("saving answers: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Enabled %q.\n", tool.DisplayName)
	if len(writtenFiles) > 0 {
		printWrittenFiles(cmd, writtenFiles, "wrote")
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "  No files generated (tool enabled but no output needed for current project configuration).\n")
	}
	return nil
}

// loadToolForEnable loads saved answers, infers enabled tools, and looks up
// the named tool in the registry.
func loadToolForEnable(projectRoot, toolName string) (types.WizardAnswers, *toolreg.Tool, error) {
	registry := toolreg.DefaultRegistry()

	// Load saved answers (empty if no prior init).
	answers, err := loadAnswersOrEmpty(projectRoot)
	if err != nil {
		return types.WizardAnswers{}, nil, fmt.Errorf("loading answers: %w", err)
	}
	answers.ProjectRoot = projectRoot

	// Migrate legacy projects that lack EnabledTools.
	toolreg.InferEnabledTools(&answers, registry)

	// Look up the tool.
	tool, ok := registry.ByName(toolName)
	if !ok {
		return types.WizardAnswers{}, nil, fmt.Errorf("unknown tool %q; use '%s list' to see available tools", toolName, branding.Get().AppName)
	}

	return answers, tool, nil
}

// writeToolFiles generates and writes both exclusive and shared files for a
// tool enable operation.
func writeToolFiles(tool *toolreg.Tool, toolName, projectRoot string, answers types.WizardAnswers) ([]types.GeneratedFile, error) {
	// Generate and write exclusive files.
	var writtenFiles []types.GeneratedFile
	if tool.GenerateFunc != nil {
		generated, err := tool.GenerateFunc(answers)
		if err != nil {
			return nil, fmt.Errorf("generating files for %q: %w", toolName, err)
		}
		for _, f := range generated {
			absPath := filepath.Join(projectRoot, f.Path)
			mode := f.Mode
			if mode == 0 {
				mode = fileutil.ModeReadWrite
			}
			if err := fileutil.WriteFileAtomic(absPath, f.Content, mode); err != nil {
				return nil, fmt.Errorf("writing %s: %w", f.Path, err)
			}
			f.Owner = toolName
			writtenFiles = append(writtenFiles, f)
		}
	}

	// Process shared files: insert sections.
	for _, sf := range tool.SharedFiles() {
		contentFunc, ok := tool.SharedContent[sf.SectionID]
		if !ok {
			continue
		}
		content, err := contentFunc(answers)
		if err != nil {
			return nil, fmt.Errorf("generating shared content for %s section %q: %w", sf.Path, sf.SectionID, err)
		}
		updated, err := applySurgery(projectRoot, sf.Path, sf.SectionID, content, true)
		if err != nil {
			return nil, fmt.Errorf("inserting section %q into %s: %w", sf.SectionID, sf.Path, err)
		}
		absPath := filepath.Join(projectRoot, sf.Path)
		if err := fileutil.WriteFileAtomic(absPath, updated, fileutil.ModeReadWrite); err != nil {
			return nil, fmt.Errorf("writing %s: %w", sf.Path, err)
		}
		writtenFiles = append(writtenFiles, types.GeneratedFile{
			Path:    sf.Path,
			Content: updated,
			Mode:    fileutil.ModeReadWrite,
			Owner:   toolName,
		})
	}

	return writtenFiles, nil
}

// saveEnableState loads the current state file, records newly written files,
// marks the tool as enabled, and persists the updated state.
func saveEnableState(stateFile, toolName string, writtenFiles []types.GeneratedFile) error {
	existingState, err := state.LoadStateFromFile(stateFile)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}
	for _, f := range writtenFiles {
		fs := types.FileState{
			Hash:  state.ComputeHash(f.Content),
			Mode:  f.Mode,
			Owner: toolName,
		}
		existingState.Files[f.Path] = fs
	}
	if existingState.EnabledTools == nil {
		existingState.EnabledTools = make(map[string]bool)
	}
	existingState.EnabledTools[toolName] = true
	existingState.LastRun = time.Now().UTC()
	if err := state.SaveStateToFile(stateFile, existingState); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}
	return nil
}

// runDisable disables a tool: validates dependents, removes files, and
// updates persisted answers and state.
func runDisable(cmd *cobra.Command, toolName string, opts disableOptions) error {
	projectRoot, err := cmdutil.ProjectRoot()
	if err != nil {
		return err
	}

	registry := toolreg.DefaultRegistry()

	answers, err := loadAnswersOrEmpty(projectRoot)
	if err != nil {
		return fmt.Errorf("loading answers: %w", err)
	}
	answers.ProjectRoot = projectRoot

	toolreg.InferEnabledTools(&answers, registry)

	tool, ok := registry.ByName(toolName)
	if !ok {
		return fmt.Errorf("unknown tool %q; use '%s list' to see available tools", toolName, branding.Get().AppName)
	}

	// Already disabled — no-op.
	if !answers.EnabledTools[toolName] {
		fmt.Fprintf(cmd.OutOrStdout(), "Tool %q is already disabled.\n", toolName)
		return nil
	}

	// Validate that the tool can be disabled.
	if err := toolreg.ValidateDisable(registry, toolName, answers.EnabledTools); err != nil {
		var alwaysOnErr *toolreg.AlwaysOnError
		if errors.As(err, &alwaysOnErr) && opts.Force {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: disabling always-on tool %q.\n", toolName)
		} else {
			return err
		}
	}

	// Load state and check for user modifications on owned files.
	stateFile := filepath.Join(projectRoot, stateFilePath())
	existingState, err := state.LoadStateFromFile(stateFile)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	if !opts.Force {
		modStatus := state.CheckModified(existingState, projectRoot)
		var modified []string
		for _, ef := range tool.ExclusiveFiles() {
			if fs, ok := modStatus[ef.Path]; ok && fs.Status == types.Modified {
				modified = append(modified, ef.Path)
			}
		}
		if len(modified) > 0 {
			return fmt.Errorf(
				"the following files have been modified by the user:\n  %s\nUse --force to remove them anyway",
				strings.Join(modified, "\n  "),
			)
		}
	}

	// Call the tool's disable function to update answers.
	if tool.DisableFunc != nil {
		tool.DisableFunc(&answers)
	}
	answers.EnabledTools[toolName] = false

	if err := removeToolFiles(tool, projectRoot, existingState); err != nil {
		return err
	}

	if err := saveDisableState(stateFile, toolName, existingState); err != nil {
		return err
	}

	// Save updated answers.
	if err := saveAnswers(projectRoot, answers); err != nil {
		return fmt.Errorf("saving answers: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Disabled %q.\n", tool.DisplayName)
	return nil
}

// removeToolFiles removes exclusive files from disk and removes shared file
// sections owned by the tool. It mutates existingState in place to reflect
// the removals.
func removeToolFiles(tool *toolreg.Tool, projectRoot string, existingState types.GeneratedState) error {
	// Remove exclusive files.
	for _, ef := range tool.ExclusiveFiles() {
		absPath := filepath.Join(projectRoot, ef.Path)
		if err := os.Remove(absPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing %s: %w", ef.Path, err)
		}
		delete(existingState.Files, ef.Path)
	}

	// Process shared files: remove sections.
	for _, sf := range tool.SharedFiles() {
		updated, err := applySurgery(projectRoot, sf.Path, sf.SectionID, nil, false)
		if err != nil {
			return fmt.Errorf("removing section %q from %s: %w", sf.SectionID, sf.Path, err)
		}
		// applySurgery returns nil when the file doesn't exist — nothing to do.
		if updated == nil {
			continue
		}
		absPath := filepath.Join(projectRoot, sf.Path)
		if err := fileutil.WriteFileAtomic(absPath, updated, fileutil.ModeReadWrite); err != nil {
			return fmt.Errorf("writing %s: %w", sf.Path, err)
		}
		// Update the state hash for the shared file.
		existingState.Files[sf.Path] = types.FileState{
			Hash:  state.ComputeHash(updated),
			Mode:  fileutil.ModeReadWrite,
			Owner: existingState.Files[sf.Path].Owner,
		}
	}

	return nil
}

// saveDisableState marks the tool as disabled in the state, updates the
// timestamp, and persists the state to disk.
func saveDisableState(stateFile, toolName string, existingState types.GeneratedState) error {
	if existingState.EnabledTools == nil {
		existingState.EnabledTools = make(map[string]bool)
	}
	existingState.EnabledTools[toolName] = false
	existingState.LastRun = time.Now().UTC()
	if err := state.SaveStateToFile(stateFile, existingState); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}
	return nil
}

// runList prints all registered tools grouped by category.
func runList(cmd *cobra.Command, opts listOptions) error {
	registry := toolreg.DefaultRegistry()

	// Load project state for enabled/disabled display.
	projectRoot, _ := cmdutil.ProjectRoot()
	var enabledTools map[string]bool
	if projectRoot != "" {
		if ans, err := loadAnswersOrEmpty(projectRoot); err == nil {
			ans.ProjectRoot = projectRoot
			toolreg.InferEnabledTools(&ans, registry)
			enabledTools = ans.EnabledTools
		}
	}

	var tools []*toolreg.Tool
	if opts.Category != "" {
		cat := toolreg.ToolCategory(opts.Category)
		tools = registry.ByCategory(cat)
		if len(tools) == 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "No tools found in category %q.\n", opts.Category)
			return nil
		}
		printToolGroup(cmd, cat, tools, enabledTools)
		return nil
	}

	tools = registry.All()
	if len(tools) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No tools registered.")
		return nil
	}

	// Group by category, preserving sort order from registry.All().
	groups := groupByCategory(tools)
	for _, g := range groups {
		printToolGroup(cmd, g.category, g.tools, enabledTools)
		fmt.Fprintln(cmd.OutOrStdout())
	}
	return nil
}

type categoryGroup struct {
	category toolreg.ToolCategory
	tools    []*toolreg.Tool
}

// groupByCategory groups a pre-sorted tool slice into category groups,
// preserving the input order.
func groupByCategory(tools []*toolreg.Tool) []categoryGroup {
	var groups []categoryGroup
	seen := make(map[toolreg.ToolCategory]int) // category -> index in groups

	for _, t := range tools {
		idx, ok := seen[t.Category]
		if !ok {
			idx = len(groups)
			seen[t.Category] = idx
			groups = append(groups, categoryGroup{category: t.Category})
		}
		groups[idx].tools = append(groups[idx].tools, t)
	}
	return groups
}

func printToolGroup(cmd *cobra.Command, cat toolreg.ToolCategory, tools []*toolreg.Tool, enabledTools map[string]bool) {
	fmt.Fprintf(cmd.OutOrStdout(), "%s:\n", cat.DisplayName())

	// Sort by name within category for consistent output.
	sorted := make([]*toolreg.Tool, len(tools))
	copy(sorted, tools)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	for _, t := range sorted {
		defaultStr := t.Default.String()
		stateStr := ""
		if enabledTools != nil {
			if enabledTools[t.Name] {
				stateStr = "[enabled]  "
			} else {
				stateStr = "[disabled] "
			}
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  %-25s  %-15s  %s%s\n", t.Name, "("+defaultStr+")", stateStr, t.Description)
	}
}

// applySurgery reads a file from disk and applies the appropriate insert or
// remove operation based on the file extension/name, then returns the updated
// content. The caller is responsible for writing the result back to disk.
func applySurgery(projectRoot, relPath, sectionID string, content []byte, insert bool) ([]byte, error) {
	absPath := filepath.Join(projectRoot, relPath)
	existing, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) && insert {
			// For insert into a non-existent file, we need a minimal scaffold.
			existing = scaffoldForPath(relPath)
		} else if os.IsNotExist(err) && !insert {
			// Nothing to remove from a file that doesn't exist.
			return nil, nil
		} else {
			return nil, fmt.Errorf("reading %s: %w", relPath, err)
		}
	}

	base := filepath.Base(relPath)
	ext := filepath.Ext(relPath)

	switch {
	case base == ".mcp.json":
		// MCP JSON uses server-name based add/remove, not section markers.
		// The sectionID is used as the server name.
		if insert {
			return surgery.JSONAddMCPServer(existing, sectionID, content)
		}
		return surgery.JSONRemoveMCPServer(existing, sectionID)

	case strings.HasSuffix(base, "settings.json"):
		// settings.json uses SettingsAdditions/SettingsRemovals.
		// For the lifecycle system, shared content for settings.json is the
		// raw JSON of the full file. This path is typically not used because
		// settings.json tools provide their own EnableFunc/DisableFunc logic.
		// Fall through to the generic case.
		if insert {
			return surgery.MarkdownInsertSection(existing, sectionID, content)
		}
		return surgery.MarkdownRemoveSection(existing, sectionID)

	case ext == ".md":
		if insert {
			return surgery.MarkdownInsertSection(existing, sectionID, content)
		}
		return surgery.MarkdownRemoveSection(existing, sectionID)

	case ext == ".nix":
		if insert {
			return surgery.NixInsertSection(existing, sectionID, content)
		}
		return surgery.NixRemoveSection(existing, sectionID)

	default:
		// For unrecognized file types, use markdown-style HTML comment markers.
		// This works for most text files and is the safest default.
		if insert {
			return surgery.MarkdownInsertSection(existing, sectionID, content)
		}
		return surgery.MarkdownRemoveSection(existing, sectionID)
	}
}

// scaffoldForPath returns minimal file content for a new shared file so that
// surgery insert operations have valid insertion points.
func scaffoldForPath(relPath string) []byte {
	ext := filepath.Ext(relPath)
	base := filepath.Base(relPath)

	switch {
	case base == ".mcp.json":
		return []byte("{}\n")
	case strings.HasSuffix(base, "settings.json"):
		return []byte("{}\n")
	case ext == ".md":
		return []byte("<!-- END GENERATED SECTION -->\n")
	case ext == ".nix":
		return []byte("{\n}\n")
	default:
		return []byte("<!-- END GENERATED SECTION -->\n")
	}
}

// printOwnedFiles prints the list of files a tool owns.
func printOwnedFiles(cmd *cobra.Command, tool *toolreg.Tool, verb string) {
	if len(tool.OwnedFiles) == 0 {
		return
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Files %s:\n", verb)
	for _, f := range tool.OwnedFiles {
		ownerType := f.Ownership.String()
		fmt.Fprintf(cmd.OutOrStdout(), "  %s (%s)\n", f.Path, ownerType)
	}
}

// printWrittenFiles prints the list of files that were actually written during
// an enable operation.
func printWrittenFiles(cmd *cobra.Command, files []types.GeneratedFile, verb string) {
	if len(files) == 0 {
		return
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Files %s:\n", verb)
	for _, f := range files {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", f.Path)
	}
}
