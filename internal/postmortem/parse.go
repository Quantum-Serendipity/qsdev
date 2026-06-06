package postmortem

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type rawMessage struct {
	Type      string          `json:"type"`
	SessionID string          `json:"sessionId"`
	Message   json.RawMessage `json:"message"`
	Result    json.RawMessage `json:"result"`
}

type messageContent struct {
	Content []contentBlock `json:"content"`
}

type contentBlock struct {
	Type  string          `json:"type"`
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

type resultPayload struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	IsError   bool   `json:"is_error"`
	Content   string `json:"content"`
}

type pendingToolUse struct {
	id   string
	name string
}

func ParseSessionJSONL(path string) (*SessionAnalysis, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening session file: %w", err)
	}
	defer f.Close()

	analysis := &SessionAnalysis{
		StartTime: time.Now(),
	}

	var pendingTools []pendingToolUse
	// Track failures by tool name for recovery detection.
	// Key: tool name, Value: index into FailureSequences.
	lastFailureByTool := make(map[string]int)

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

		switch raw.Type {
		case "summary":
			analysis.SessionID = raw.SessionID

		case "assistant":
			if raw.Message == nil {
				continue
			}
			var msg messageContent
			if err := json.Unmarshal(raw.Message, &msg); err != nil {
				continue
			}
			for _, block := range msg.Content {
				if block.Type == "tool_use" {
					analysis.ToolUseCount++
					pendingTools = append(pendingTools, pendingToolUse{
						id:   block.ID,
						name: block.Name,
					})
				}
			}

		case "result":
			if raw.Result == nil {
				continue
			}
			var res resultPayload
			if err := json.Unmarshal(raw.Result, &res); err != nil {
				continue
			}
			if res.Type != "tool_result" {
				continue
			}

			toolName := findAndRemovePending(&pendingTools, res.ToolUseID)
			if toolName == "" {
				continue
			}

			if res.IsError {
				failIdx := len(analysis.FailureSequences)
				seq := FailureSequence{
					ToolName:     toolName,
					ErrorMessage: res.Content,
					RetryCount:   0,
					Recovered:    false,
				}

				if prevIdx, ok := lastFailureByTool[toolName]; ok {
					prev := &analysis.FailureSequences[prevIdx]
					if !prev.Recovered {
						prev.RetryCount++
					}
				}

				analysis.FailureSequences = append(analysis.FailureSequences, seq)
				lastFailureByTool[toolName] = failIdx
			} else {
				if prevIdx, ok := lastFailureByTool[toolName]; ok {
					prev := &analysis.FailureSequences[prevIdx]
					if !prev.Recovered {
						prev.Recovered = true
						prev.RecoveryAction = "retry"
						analysis.RecoveryPatterns = append(analysis.RecoveryPatterns, toolName+": retry")
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning session file: %w", err)
	}

	return analysis, nil
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

func FindSessionFiles(root string) ([]string, error) {
	var paths []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && filepath.Ext(path) == ".jsonl" {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking session directory: %w", err)
	}
	return paths, nil
}
