package hardening

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type ProvenanceEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	Server      string    `json:"server"`
	Tool        string    `json:"tool"`
	Tier        int       `json:"tier"`
	ContentHash string    `json:"content_hash"`
	Detections  int       `json:"detections"`
}

func LogProvenance(path string, entry ProvenanceEntry) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating provenance log directory: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening provenance log: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling provenance entry: %w", err)
	}

	data = append(data, '\n')
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("writing provenance entry: %w", err)
	}

	return nil
}
