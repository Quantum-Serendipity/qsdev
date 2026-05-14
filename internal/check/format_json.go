package check

import (
	"encoding/json"
	"io"
)

func formatJSON(report *CheckReport, w io.Writer) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = w.Write(data)
	return err
}
