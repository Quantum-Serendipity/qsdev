package vsentinel

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

func LogVersionEvent(logPath string, event VersionEvent) error {
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshaling event: %w", err)
	}

	data = append(data, '\n')
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("writing event: %w", err)
	}

	return nil
}

func ReadVersionHistory(logPath string) ([]VersionEvent, error) {
	f, err := os.Open(logPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("opening log file: %w", err)
	}
	defer f.Close()

	var events []VersionEvent
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var event VersionEvent
		if err := json.Unmarshal(line, &event); err != nil {
			continue
		}
		events = append(events, event)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning log file: %w", err)
	}

	return events, nil
}
