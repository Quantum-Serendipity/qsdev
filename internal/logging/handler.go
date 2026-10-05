package logging

import (
	"context"
	"log/slog"
)

// RedactingHandler wraps an slog.Handler, scrubbing secret values
// from every record before forwarding to the inner handler.
type RedactingHandler struct {
	inner    slog.Handler
	redactor *Redactor
	// denied is set once any enclosing WithGroup name is sensitive (e.g.
	// logger.WithGroup("secret")); every attribute beneath it is then redacted
	// whole, since the inner handler nests it under that group.
	denied bool
}

// NewRedactingHandler wraps inner with secret redaction.
func NewRedactingHandler(inner slog.Handler) *RedactingHandler {
	return &RedactingHandler{
		inner:    inner,
		redactor: NewRedactor(),
	}
}

func (h *RedactingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *RedactingHandler) Handle(ctx context.Context, r slog.Record) error {
	scrubbed := slog.NewRecord(r.Time, r.Level, h.redactor.RedactString(r.Message), r.PC)
	r.Attrs(func(a slog.Attr) bool {
		scrubbed.AddAttrs(h.redactAttr(a))
		return true
	})
	return h.inner.Handle(ctx, scrubbed)
}

func (h *RedactingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	redacted := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		redacted[i] = h.redactAttr(a)
	}
	return &RedactingHandler{
		inner:    h.inner.WithAttrs(redacted),
		redactor: h.redactor,
		denied:   h.denied,
	}
}

func (h *RedactingHandler) WithGroup(name string) slog.Handler {
	return &RedactingHandler{
		inner:    h.inner.WithGroup(name),
		redactor: h.redactor,
		denied:   h.denied || h.redactor.isKeyDenied(name),
	}
}

// redactAttr scrubs one attribute. Under a sensitive WithGroup scope every leaf
// value is replaced whole (keys, including those of nested groups, stay visible
// for diagnosability); otherwise the shared Redactor.RedactAttr rules apply.
func (h *RedactingHandler) redactAttr(a slog.Attr) slog.Attr {
	if !h.denied {
		return h.redactor.RedactAttr(a)
	}
	return denyAttr(a)
}

// TeeHandler fans out log records to multiple handlers.
type TeeHandler struct {
	handlers []slog.Handler
}

// NewTeeHandler creates a handler that writes to all provided handlers.
func NewTeeHandler(handlers ...slog.Handler) *TeeHandler {
	return &TeeHandler{handlers: handlers}
}

func (h *TeeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *TeeHandler) Handle(ctx context.Context, r slog.Record) error {
	var firstErr error
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, r.Level) {
			if err := handler.Handle(ctx, r); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (h *TeeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithAttrs(attrs)
	}
	return NewTeeHandler(handlers...)
}

func (h *TeeHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithGroup(name)
	}
	return NewTeeHandler(handlers...)
}
