package hookio

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func TestRunWithDeadline(t *testing.T) {
	t.Parallel()

	errDeny := errors.New("deny")
	tests := []struct {
		name         string
		eval         func(ctx context.Context, w io.Writer) error
		wantTimedOut bool
		wantErr      bool
		wantOutput   string // exact output, or "" to check wantContains
		wantContains []string
		wantAbsent   string
	}{
		{
			name: "blocking evaluator times out and its late output is dropped",
			eval: func(_ context.Context, w io.Writer) error {
				time.Sleep(300 * time.Millisecond)
				_, _ = io.WriteString(w, "LATE ALLOW OUTPUT")
				return nil
			},
			wantTimedOut: true,
			wantContains: []string{"SP-TIMEOUT", EvalDeadline.String(), "fail closed"},
			wantAbsent:   "LATE ALLOW OUTPUT",
		},
		{
			name: "fast evaluator output is copied through unchanged",
			eval: func(_ context.Context, w io.Writer) error {
				WriteDeny(w, "SP-001", "protected")
				return errDeny
			},
			wantErr:    true,
			wantOutput: "qsdev-selfprotect: SP-001 — protected\n",
		},
		{
			name:       "fast allowing evaluator writes nothing",
			eval:       func(context.Context, io.Writer) error { return nil },
			wantOutput: "",
		},
		{
			name: "panicking evaluator becomes an error",
			eval: func(context.Context, io.Writer) error {
				panic("boom")
			},
			wantErr:      true,
			wantContains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()

			var out bytes.Buffer
			start := time.Now()
			timedOut, err := RunWithDeadline(ctx, tt.eval, &out)
			if elapsed := time.Since(start); elapsed > time.Second {
				t.Errorf("RunWithDeadline returned after %v", elapsed)
			}
			if timedOut != tt.wantTimedOut {
				t.Errorf("timedOut = %v, want %v", timedOut, tt.wantTimedOut)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, want error %v", err, tt.wantErr)
			}
			got := out.String()
			if tt.wantContains == nil && got != tt.wantOutput {
				t.Errorf("output = %q, want %q", got, tt.wantOutput)
			}
			for _, s := range tt.wantContains {
				if !strings.Contains(got, s) {
					t.Errorf("output %q does not contain %q", got, s)
				}
			}
			if tt.wantAbsent != "" {
				// Give the abandoned evaluator time to finish its late write.
				time.Sleep(400 * time.Millisecond)
				if strings.Contains(out.String(), tt.wantAbsent) {
					t.Errorf("output %q contains the evaluator's late output", out.String())
				}
			}
		})
	}
}

func TestRunWithDeadline_PanicErrorNamesValue(t *testing.T) {
	t.Parallel()
	_, err := RunWithDeadline(context.Background(), func(context.Context, io.Writer) error {
		panic("boom")
	}, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("err = %v, want it to name the panic value", err)
	}
}
