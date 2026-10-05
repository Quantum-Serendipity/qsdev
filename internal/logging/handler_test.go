package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

// TestRedactingHandler_RedactsMessage proves the wrapping handler scrubs a secret
// embedded in a record's MESSAGE (not just its attributes) before forwarding to
// the inner handler — the third credential-leak path (handler copied r.Message
// verbatim before this fix).
func TestRedactingHandler_RedactsMessage(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(NewRedactingHandler(inner))

	logger.Info("connecting with PGPASSWORD=supersecret to db")

	out := buf.String()
	if strings.Contains(out, "supersecret") {
		t.Errorf("handler leaked secret embedded in message: %s", out)
	}
	if !strings.Contains(out, redacted) {
		t.Errorf("expected %s marker in message output: %s", redacted, out)
	}
}

// TestRedactingHandler_SensitiveGroup proves a sensitive GROUP name redacts every
// value beneath it — whether the group is an inline slog.Group attr, a
// logger.WithGroup scope (for both record attrs and WithAttrs), or a map value
// under a sensitive key — while keys stay visible for diagnosability and a
// benign group passes its values through unchanged.
func TestRedactingHandler_SensitiveGroup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		log      func(l *slog.Logger)
		leak     string
		wantKeys []string
	}{
		{
			name:     "inline group attr",
			log:      func(l *slog.Logger) { l.Info("m", slog.Group("password", "value", "x1")) },
			leak:     "x1",
			wantKeys: []string{`"password"`, `"value"`},
		},
		{
			name:     "WithGroup record attr",
			log:      func(l *slog.Logger) { l.WithGroup("secret").Info("m", "value", "x2") },
			leak:     "x2",
			wantKeys: []string{`"secret"`, `"value"`},
		},
		{
			name:     "WithGroup then With",
			log:      func(l *slog.Logger) { l.WithGroup("secret").With("value", "x4").Info("m") },
			leak:     "x4",
			wantKeys: []string{`"secret"`, `"value"`},
		},
		{
			name:     "nested WithGroup under sensitive group",
			log:      func(l *slog.Logger) { l.WithGroup("secret").WithGroup("inner").Info("m", "value", "x5") },
			leak:     "x5",
			wantKeys: []string{`"secret"`, `"inner"`, `"value"`},
		},
		{
			name:     "inline group under sensitive WithGroup",
			log:      func(l *slog.Logger) { l.WithGroup("secret").Info("m", slog.Group("g", "v", "x6")) },
			leak:     "x6",
			wantKeys: []string{`"secret"`, `"g"`, `"v"`},
		},
		{
			name:     "map under plural sensitive key",
			log:      func(l *slog.Logger) { l.Info("m", "credentials", map[string]string{"a": "x3"}) },
			leak:     "x3",
			wantKeys: []string{`"credentials"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			tt.log(slog.New(NewRedactingHandler(slog.NewJSONHandler(&buf, nil))))
			out := buf.String()
			if strings.Contains(out, tt.leak) {
				t.Errorf("sensitive group leaked %q: %s", tt.leak, out)
			}
			for _, k := range tt.wantKeys {
				if !strings.Contains(out, k) {
					t.Errorf("key %s should stay visible: %s", k, out)
				}
			}
		})
	}

	t.Run("empty attr under sensitive WithGroup stays dropped", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		l := slog.New(NewRedactingHandler(slog.NewJSONHandler(&buf, nil)))
		l.WithGroup("secret").Info("m", slog.Attr{})
		l.Info("m", slog.Group("password", slog.Attr{}, "value", "x7"))
		out := buf.String()
		if strings.Contains(out, `"":`) {
			t.Errorf("empty attr rendered as an empty-key member: %s", out)
		}
		if strings.Contains(out, "x7") {
			t.Errorf("sensitive inline group leaked: %s", out)
		}
	})

	t.Run("benign group passes through", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		l := slog.New(NewRedactingHandler(slog.NewJSONHandler(&buf, nil)))
		l.WithGroup("req").With("path", "/v1").Info("m", "value", "ok-value")
		out := buf.String()
		for _, want := range []string{`"req":{`, `"path":"/v1"`, `"value":"ok-value"`} {
			if !strings.Contains(out, want) {
				t.Errorf("benign group output missing %s: %s", want, out)
			}
		}
	})
}
