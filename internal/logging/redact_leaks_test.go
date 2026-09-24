package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"testing"
)

// secretLogValuer is a slog.LogValuer whose resolved value carries a secret.
type secretLogValuer struct{}

func (secretLogValuer) LogValue() slog.Value {
	return slog.StringValue("DATABASE_PASSWORD=fromvaluer")
}

type cloneConfig struct {
	Remote   string `json:"remote"`
	APIToken string `json:"api_token"`
}

// TestRedactingHandler_ScrubsNonStringAttrs is the regression guard for the
// KindAny leak: errors, argv slices, maps, structs and LogValuers were written
// verbatim because RedactAttr only scrubbed KindString values.
func TestRedactingHandler_ScrubsNonStringAttrs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		attr    any
		secrets []string
	}{
		{
			name:    "error with tokenized remote",
			attr:    errors.New("git push https://x-access-token:s3cr3tvalue@github.com/o/r.git failed"),
			secrets: []string{"s3cr3tvalue", "x-access-token"},
		},
		{
			name:    "wrapped error with named secret",
			attr:    fmt.Errorf("wrap: %w", errors.New("token=opensesame")),
			secrets: []string{"opensesame"},
		},
		{
			name:    "argv slice",
			attr:    []string{"--password", "hunter2", "DATABASE_PASSWORD=zz", "--api-key", "k3y"},
			secrets: []string{"hunter2", "=zz", "k3y"},
		},
		{
			name:    "map with sensitive key",
			attr:    map[string]any{"api_token": "plaintext", "region": "eu"},
			secrets: []string{"plaintext"},
		},
		{
			name:    "struct with sensitive json field",
			attr:    cloneConfig{Remote: "origin", APIToken: "structsecret"},
			secrets: []string{"structsecret"},
		},
		{
			name:    "log valuer",
			attr:    secretLogValuer{},
			secrets: []string{"fromvaluer"},
		},
		{
			name:    "url with userinfo",
			attr:    &url.URL{Scheme: "https", User: url.UserPassword("bob", "urlpass"), Host: "example.com"},
			secrets: []string{"urlpass"},
		},
	}

	handlers := map[string]func(*bytes.Buffer) slog.Handler{
		"json": func(b *bytes.Buffer) slog.Handler { return slog.NewJSONHandler(b, nil) },
		"text": func(b *bytes.Buffer) slog.Handler { return slog.NewTextHandler(b, nil) },
	}

	for _, tt := range tests {
		for hname, mk := range handlers {
			t.Run(tt.name+"/"+hname, func(t *testing.T) {
				t.Parallel()
				var buf bytes.Buffer
				logger := slog.New(NewRedactingHandler(mk(&buf)))
				logger.Warn("operation failed", "detail", tt.attr)

				out := buf.String()
				for _, s := range tt.secrets {
					if strings.Contains(out, s) {
						t.Errorf("handler leaked %q: %s", s, out)
					}
				}
				if !strings.Contains(out, redacted) {
					t.Errorf("expected %s marker in output: %s", redacted, out)
				}
			})
		}
	}
}

// TestRedactingHandler_KeepsBenignAnyValues proves non-secret KindAny values
// are still rendered faithfully after the scrub.
func TestRedactingHandler_KeepsBenignAnyValues(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(NewRedactingHandler(slog.NewJSONHandler(&buf, nil)))
	var nilErr *url.Error
	logger.Info("ok",
		"args", []string{"build", "--verbose", "./..."},
		"counts", map[string]int{"files": 3},
		"error", errors.New("exit status 1"),
		"nil_err", nilErr,
	)

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("unmarshal %s: %v", buf.String(), err)
	}
	if got := fmt.Sprint(rec["args"]); got != "[build --verbose ./...]" {
		t.Errorf("args = %s, want unchanged", got)
	}
	if got := fmt.Sprint(rec["counts"]); got != "map[files:3]" {
		t.Errorf("counts = %s, want unchanged", got)
	}
	if rec["error"] != "exit status 1" {
		t.Errorf("error = %v, want unchanged", rec["error"])
	}
}

// TestRedactString_JSONKeys covers sensitive names written as JSON members,
// which the bare NAME=value pattern missed because a quote sits between the
// NAME and its separator.
func TestRedactString_JSONKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "compact",
			input: `{"DATABASE_PASSWORD":"hunter2"}`,
			want:  `{"DATABASE_PASSWORD":"[REDACTED]"}`,
		},
		{
			name:  "spaced",
			input: `{"DATABASE_PASSWORD": "hunter2", "x": 1}`,
			want:  `{"DATABASE_PASSWORD": "[REDACTED]", "x": 1}`,
		},
		{
			name:  "later members preserved",
			input: `{"region":"eu","password":"hunter2","port":5432}`,
			want:  `{"region":"eu","password":"[REDACTED]","port":5432}`,
		},
		{
			name:  "escaped quote inside value",
			input: `{"password":"a\"b c","x":1}`,
			want:  `{"password":"[REDACTED]","x":1}`,
		},
		{
			name:  "nested object value",
			input: `{"cfg":{"api_token":"plaintext","n":1},"ok":true}`,
			want:  `{"cfg":{"api_token":"[REDACTED]","n":1},"ok":true}`,
		},
		{
			name:  "object under sensitive key",
			input: `{"auth":{"user":"u","pass":"p"},"ok":true}`,
			want:  `{"auth":"[REDACTED]","ok":true}`,
		},
		{
			name:  "numeric value",
			input: `{"pin_secret":123456,"ok":true}`,
			want:  `{"pin_secret":"[REDACTED]","ok":true}`,
		},
		{
			name:  "escaped json in a json string",
			input: `{"msg":"body {\"PGPASSWORD\":\"s3cr3t\",\"db\":\"x\"}"}`,
			want:  `{"msg":"body {\"PGPASSWORD\":\"[REDACTED]\",\"db\":\"x\"}"}`,
		},
		{
			name:  "unterminated value fails closed",
			input: `{"password":"hunter2`,
			want:  `{"password":"[REDACTED]`,
		},
		{
			name:  "non-sensitive json unchanged",
			input: `{"path":"/usr/bin","keyboard":"us"}`,
			want:  `{"path":"/usr/bin","keyboard":"us"}`,
		},
		{
			name:  "pair after non-sensitive pair is not swallowed",
			input: "user=alice,password=hunter2",
			want:  "user=alice,password=[REDACTED]",
		},
		{
			// The pair is the value of a preceding "NAME:" (an error prefix);
			// matching that prefix must not consume the pair's first letter.
			name:  "pair as the value of a prefix key",
			input: "clone failed: token=abc123",
			want:  "clone failed: token=[REDACTED]",
		},
		{
			name:  "pair directly after a prefix key",
			input: "wrap:PASSWORD=hunter2 retry=1",
			want:  "wrap:PASSWORD=[REDACTED] retry=1",
		},
	}
	r := NewRedactor()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := r.RedactString(tt.input); got != tt.want {
				t.Errorf("RedactString(%s)\n got  %s\n want %s", tt.input, got, tt.want)
			}
		})
	}
}

// TestRedactString_JSONLineStaysValid proves a redacted JSONL record is still
// valid JSON, so display-time re-scrubbing never corrupts log lines.
func TestRedactString_JSONLineStaysValid(t *testing.T) {
	t.Parallel()

	line := `{"time":"t","msg":"m","cfg":{"api_token":"plaintext"},"args":["--token","x"],"secret":42}`
	got := NewRedactor().RedactString(line)
	if strings.Contains(got, "plaintext") {
		t.Errorf("leaked secret: %s", got)
	}
	if !json.Valid([]byte(got)) {
		t.Errorf("redacted line is not valid JSON: %s", got)
	}
}

// TestRedactString_PrivateKeyBlock covers the whole-block PEM redaction: the
// old pattern replaced only the BEGIN header and leaked the key body.
func TestRedactString_PrivateKeyBlock(t *testing.T) {
	t.Parallel()

	const body = "MIIEowIBAAKCAQEAsecretbodyline"
	tests := []struct {
		name  string
		input string
		keep  []string
	}{
		{
			name:  "multi-line rsa",
			input: "before\n-----BEGIN RSA PRIVATE KEY-----\n" + body + "\n-----END RSA PRIVATE KEY-----\nafter",
			keep:  []string{"before\n", "\nafter"},
		},
		{
			name:  "json-escaped newlines",
			input: `{"key":"-----BEGIN PRIVATE KEY-----\n` + body + `\n-----END PRIVATE KEY-----","n":1}`,
			keep:  []string{`"n":1`},
		},
		{
			name:  "encrypted",
			input: "-----BEGIN ENCRYPTED PRIVATE KEY-----\n" + body + "\n-----END ENCRYPTED PRIVATE KEY-----",
		},
		{
			name:  "pgp",
			input: "-----BEGIN PGP PRIVATE KEY BLOCK-----\n" + body + "\n-----END PGP PRIVATE KEY BLOCK-----",
		},
		{
			name:  "openssh",
			input: "-----BEGIN OPENSSH PRIVATE KEY-----\n" + body + "\n-----END OPENSSH PRIVATE KEY-----",
		},
		{
			name:  "missing end marker fails closed",
			input: "-----BEGIN EC PRIVATE KEY-----\n" + body,
		},
	}
	r := NewRedactor()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := r.RedactString(tt.input)
			if strings.Contains(got, body) || strings.Contains(got, "PRIVATE KEY") {
				t.Errorf("key material survived: %q", got)
			}
			for _, k := range tt.keep {
				if !strings.Contains(got, k) {
					t.Errorf("surrounding text %q lost: %q", k, got)
				}
			}
		})
	}
}

// TestPrivateKeyLineFilter proves a key block split across the lines of a
// line-oriented stream is fully suppressed.
func TestPrivateKeyLineFilter(t *testing.T) {
	t.Parallel()

	lines := []string{
		"building",
		"key: -----BEGIN RSA PRIVATE KEY-----",
		"MIIEowIBAAKCAQEAsecretbodyline",
		"c2Vjb25kc2VjcmV0bGluZQ==",
		"-----END RSA PRIVATE KEY----- trailing",
		"done",
	}
	var f PrivateKeyLineFilter
	r := NewRedactor()
	var got []string
	for _, l := range lines {
		got = append(got, r.RedactString(f.Filter(l)))
	}
	joined := strings.Join(got, "\n")
	for _, secret := range []string{"secretbodyline", "c2Vjb25kc2VjcmV0bGluZQ", "PRIVATE KEY"} {
		if strings.Contains(joined, secret) {
			t.Errorf("line filter leaked %q:\n%s", secret, joined)
		}
	}
	if got[0] != "building" || got[5] != "done" {
		t.Errorf("lines outside the block changed: %q", got)
	}
	if !strings.HasSuffix(got[4], " trailing") {
		t.Errorf("text after END marker lost: %q", got[4])
	}
}

// TestRedactString_URLUserinfo covers token-only userinfo, which urlCredRe
// (user:pass@ only) missed.
func TestRedactString_URLUserinfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		secret string
		want   string
	}{
		{
			name:   "token only",
			input:  "cloning https://mytoken1234567890@gitlab.example.com/repo.git",
			secret: "mytoken1234567890",
			want:   "cloning https://[REDACTED]@gitlab.example.com/repo.git",
		},
		{
			name:   "user and password",
			input:  "https://admin:s3cret@registry.example.com/v2/",
			secret: "s3cret",
		},
		{
			name:  "scp form untouched",
			input: "git@github.com:owner/repo.git",
			want:  "git@github.com:owner/repo.git",
		},
		{
			name:  "scoped package path untouched",
			input: "https://registry.npmjs.org/@scope/pkg",
			want:  "https://registry.npmjs.org/@scope/pkg",
		},
	}
	r := NewRedactor()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := r.RedactString(tt.input)
			if tt.secret != "" && strings.Contains(got, tt.secret) {
				t.Errorf("RedactString(%q) = %q leaks %q", tt.input, got, tt.secret)
			}
			if tt.want != "" && got != tt.want {
				t.Errorf("RedactString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
