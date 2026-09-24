package postmortem

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// rawMessage is one line of a Claude Code session transcript. Every record
// carries the session ID and a timestamp; tool calls arrive in "assistant"
// records and their results in "user" records, both inside message.content.
type rawMessage struct {
	Type      string          `json:"type"`
	SessionID string          `json:"sessionId"`
	Timestamp string          `json:"timestamp"`
	Message   json.RawMessage `json:"message"`
}

// messageContent is a transcript message. Content is an array of blocks for
// tool traffic but a plain string for a typed user prompt, so it is decoded
// lazily by contentBlocks.
type messageContent struct {
	Content json.RawMessage `json:"content"`
}

type contentBlock struct {
	Type string `json:"type"`
	// tool_use fields.
	ID   string `json:"id"`
	Name string `json:"name"`
	// tool_result fields. Content is a string or an array of content blocks.
	ToolUseID string          `json:"tool_use_id"`
	IsError   bool            `json:"is_error"`
	Content   json.RawMessage `json:"content"`
}

type pendingToolUse struct {
	id   string
	name string
}

// ParseSessionJSONL analyzes one Claude Code session transcript: it counts tool
// calls, pairs each tool_result with its tool_use, and records failure
// sequences and the retries that recovered from them.
func ParseSessionJSONL(path string) (*SessionAnalysis, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening session file: %w", err)
	}
	defer f.Close()

	analysis := &SessionAnalysis{}
	p := sessionParser{analysis: analysis, lastFailureByTool: make(map[string]int)}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var raw rawMessage
		if err := json.Unmarshal(line, &raw); err != nil {
			continue
		}
		p.record(raw)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning session file: %w", err)
	}

	return analysis, nil
}

// sessionParser holds the cross-record state of one transcript parse.
type sessionParser struct {
	analysis     *SessionAnalysis
	pendingTools []pendingToolUse
	// lastFailureByTool maps a tool name to its latest entry in
	// FailureSequences, for retry and recovery detection.
	lastFailureByTool map[string]int
}

func (p *sessionParser) record(raw rawMessage) {
	if p.analysis.SessionID == "" && raw.SessionID != "" {
		p.analysis.SessionID = raw.SessionID
	}
	if p.analysis.StartTime.IsZero() && raw.Timestamp != "" {
		if ts, err := time.Parse(time.RFC3339Nano, raw.Timestamp); err == nil {
			p.analysis.StartTime = ts
		}
	}

	switch raw.Type {
	case "assistant":
		for _, block := range contentBlocks(raw.Message) {
			if block.Type == "tool_use" {
				p.analysis.ToolUseCount++
				p.pendingTools = append(p.pendingTools, pendingToolUse{id: block.ID, name: block.Name})
			}
		}
	case "user":
		for _, block := range contentBlocks(raw.Message) {
			if block.Type == "tool_result" {
				p.toolResult(block)
			}
		}
	}
}

func (p *sessionParser) toolResult(res contentBlock) {
	toolName := findAndRemovePending(&p.pendingTools, res.ToolUseID)
	if toolName == "" {
		return
	}
	a := p.analysis

	if !res.IsError {
		if prevIdx, ok := p.lastFailureByTool[toolName]; ok {
			prev := &a.FailureSequences[prevIdx]
			if !prev.Recovered {
				prev.Recovered = true
				prev.RecoveryAction = "retry"
				a.RecoveryPatterns = append(a.RecoveryPatterns, toolName+": retry")
			}
		}
		return
	}

	if prevIdx, ok := p.lastFailureByTool[toolName]; ok {
		prev := &a.FailureSequences[prevIdx]
		if !prev.Recovered {
			prev.RetryCount++
		}
	}
	p.lastFailureByTool[toolName] = len(a.FailureSequences)
	a.FailureSequences = append(a.FailureSequences, FailureSequence{
		ToolName:     toolName,
		ErrorMessage: resultText(res.Content),
	})
}

// contentBlocks decodes message.content as an array of blocks. A string
// content (a typed prompt) or an undecodable one yields no blocks.
func contentBlocks(message json.RawMessage) []contentBlock {
	if message == nil {
		return nil
	}
	var msg messageContent
	if err := json.Unmarshal(message, &msg); err != nil {
		return nil
	}
	var blocks []contentBlock
	if err := json.Unmarshal(msg.Content, &blocks); err != nil {
		return nil
	}
	return blocks
}

// resultText flattens a tool_result's content, which is either a string or an
// array of blocks whose text blocks carry the output.
func resultText(content json.RawMessage) string {
	if content == nil {
		return ""
	}
	var s string
	if err := json.Unmarshal(content, &s); err == nil {
		return s
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(content, &blocks); err != nil {
		return ""
	}
	var texts []string
	for _, b := range blocks {
		if b.Type == "text" && b.Text != "" {
			texts = append(texts, b.Text)
		}
	}
	return strings.Join(texts, "\n")
}

func findAndRemovePending(pending *[]pendingToolUse, id string) string {
	for i, p := range *pending {
		if p.id == id {
			name := p.name
			*pending = append((*pending)[:i], (*pending)[i+1:]...)
			return name
		}
	}
	return ""
}

func AggregateFailures(sessions []*SessionAnalysis) *FailureReport {
	report := &FailureReport{
		TotalSessions: len(sessions),
	}

	type patternKey struct {
		toolName string
		err      string
	}

	counts := make(map[patternKey]*PatternEntry)

	for _, s := range sessions {
		for _, f := range s.FailureSequences {
			key := patternKey{toolName: f.ToolName, err: f.ErrorMessage}
			entry, ok := counts[key]
			if !ok {
				entry = &PatternEntry{
					ToolName: f.ToolName,
					Error:    f.ErrorMessage,
				}
				counts[key] = entry
			}
			entry.Count++
			if f.Recovered {
				entry.Recovered++
			}
		}
	}

	for _, entry := range counts {
		report.Patterns = append(report.Patterns, *entry)
	}

	sort.Slice(report.Patterns, func(i, j int) bool {
		return report.Patterns[i].Count > report.Patterns[j].Count
	})

	return report
}

// SessionFileScan is the result of FindSessionFiles.
type SessionFileScan struct {
	// Paths are the session transcripts found, in walk order.
	Paths []string
	// Truncated reports that the walk stopped at the file limit.
	Truncated bool
	// Unreadable counts directories the walk could not read.
	Unreadable int
}

// FindSessionFiles collects the .jsonl session transcripts under root, stopping
// after limit files (limit <= 0 means no limit). Symlinks are not followed,
// so every returned path lies under root. Unreadable sub-directories are
// skipped and counted; an unreadable root is an error.
func FindSessionFiles(root string, limit int) (SessionFileScan, error) {
	var scan SessionFileScan
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == root {
				return err
			}
			scan.Unreadable++
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		// A symlinked transcript is skipped: following it would read a file
		// outside root, which callers use to confine what may be read.
		if d.IsDir() || d.Type()&fs.ModeSymlink != 0 || filepath.Ext(path) != ".jsonl" {
			return nil
		}
		if limit > 0 && len(scan.Paths) >= limit {
			scan.Truncated = true
			return fs.SkipAll
		}
		scan.Paths = append(scan.Paths, path)
		return nil
	})
	if err != nil && !errors.Is(err, fs.SkipAll) {
		return scan, fmt.Errorf("walking session directory: %w", err)
	}
	return scan, nil
}
