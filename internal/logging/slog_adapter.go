package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

const envLogFormat = "CS2_LOG_FORMAT"

var (
	reSensitiveKeyValue = regexp.MustCompile(`(?i)\b(token|access_token|apikey|api_key|secret|password|passwd|signature|sign|auth|authorization|credential|private_key|key)\s*([=:])\s*([^\s&;,]+)`)
	reBearerToken       = regexp.MustCompile(`(?i)\b(Bearer\s+)[A-Za-z0-9\-\._~\+/=]+`)
	reAuthorizationHdr  = regexp.MustCompile(`(?i)\bAuthorization\s*:\s*[^\s]+(?:\s+[^\s]+)?`)
)

// Options controls SlogAdapter behavior.
type Options struct {
	Writer io.Writer
	Format string
	Now    func() time.Time
	Sink   func(Entry)
}

// SlogAdapter bridges slog output and envsetup structured log entries.
type SlogAdapter struct {
	logger *slog.Logger
	now    func() time.Time
	sink   func(Entry)
}

// NewSlogAdapter builds a structured logger backed by slog.
func NewSlogAdapter(opts Options) *SlogAdapter {
	writer := opts.Writer
	if writer == nil {
		writer = os.Stderr
	}
	format := normalizeFormat(opts.Format)
	replace := func(groups []string, attr slog.Attr) slog.Attr {
		return redactAttr(groups, attr)
	}
	handlerOptions := &slog.HandlerOptions{ReplaceAttr: replace}
	var handler slog.Handler
	if format == "text" {
		handler = slog.NewTextHandler(writer, handlerOptions)
	} else {
		handler = slog.NewJSONHandler(writer, handlerOptions)
	}

	now := opts.Now
	if now == nil {
		now = time.Now
	}
	sink := opts.Sink
	if sink == nil {
		sink = func(Entry) {}
	}
	return &SlogAdapter{
		logger: slog.New(handler),
		now:    now,
		sink:   sink,
	}
}

func normalizeFormat(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		value = strings.ToLower(strings.TrimSpace(os.Getenv(envLogFormat)))
	}
	if value == "text" {
		return "text"
	}
	return "json"
}

// Info emits an info-level structured entry.
func (s *SlogAdapter) Info(message string, fields Fields) {
	s.emit(LevelInfo, message, fields)
}

// Warn emits a warning-level structured entry.
func (s *SlogAdapter) Warn(message string, fields Fields) {
	s.emit(LevelWarn, message, fields)
}

// Error emits an error-level structured entry.
func (s *SlogAdapter) Error(message string, fields Fields) {
	s.emit(LevelError, message, fields)
}

// StepStart emits a standard step-start log and returns token for elapsed tracking.
func (s *SlogAdapter) StepStart(component, stage, action, source string, attempt int, meta map[string]string) StepToken {
	step := StepToken{
		Component: strings.TrimSpace(component),
		Stage:     strings.TrimSpace(stage),
		Action:    strings.TrimSpace(action),
		Source:    strings.TrimSpace(source),
		Attempt:   attempt,
		StartedAt: s.now(),
	}
	s.Info("步骤开始", Fields{
		Component: step.Component,
		Stage:     step.Stage,
		Action:    step.Action,
		Source:    step.Source,
		Attempt:   step.Attempt,
		Meta:      meta,
	})
	return step
}

// StepDone emits a standard step-done log using elapsed time.
func (s *SlogAdapter) StepDone(step StepToken, meta map[string]string) {
	s.Info("步骤完成", Fields{
		Component: step.Component,
		Stage:     step.Stage,
		Action:    step.Action,
		Source:    step.Source,
		Attempt:   step.Attempt,
		ElapsedMS: elapsedMS(step.StartedAt, s.now()),
		Meta:      meta,
	})
}

// StepFail emits a standard step-fail log using elapsed time and error details.
func (s *SlogAdapter) StepFail(step StepToken, err error, meta map[string]string) {
	message := "步骤失败"
	errText := ""
	if err != nil {
		errText = err.Error()
		message = fmt.Sprintf("步骤失败: %v", err)
	}
	s.Error(message, Fields{
		Component: step.Component,
		Stage:     step.Stage,
		Action:    step.Action,
		Source:    step.Source,
		Attempt:   step.Attempt,
		ElapsedMS: elapsedMS(step.StartedAt, s.now()),
		Error:     errText,
		Meta:      meta,
	})
}

func (s *SlogAdapter) emit(level Level, message string, fields Fields) {
	clean := sanitizeFields(fields)
	cleanMessage := sanitizeText(strings.TrimSpace(message))
	now := s.now()
	s.logger.LogAttrs(context.Background(), levelToSlog(level), cleanMessage, fieldsToAttrs(clean)...)

	entry := Entry{
		Level:     string(level),
		Message:   cleanMessage,
		Time:      now.Format(time.RFC3339),
		Component: clean.Component,
		Stage:     clean.Stage,
		Action:    clean.Action,
		Source:    clean.Source,
		Attempt:   clean.Attempt,
		ElapsedMS: clean.ElapsedMS,
		Error:     clean.Error,
	}
	if len(clean.Meta) > 0 {
		entry.Meta = clean.Meta
	}
	s.sink(entry)
}

func levelToSlog(level Level) slog.Level {
	switch level {
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func elapsedMS(startedAt time.Time, now time.Time) int64 {
	if startedAt.IsZero() {
		return 0
	}
	elapsed := now.Sub(startedAt).Milliseconds()
	if elapsed < 0 {
		return 0
	}
	return elapsed
}

func sanitizeFields(fields Fields) Fields {
	clean := Fields{
		Component: strings.TrimSpace(fields.Component),
		Stage:     strings.TrimSpace(fields.Stage),
		Action:    strings.TrimSpace(fields.Action),
		Source:    strings.ToLower(strings.TrimSpace(fields.Source)),
		Attempt:   fields.Attempt,
		ElapsedMS: fields.ElapsedMS,
		Error:     sanitizeText(strings.TrimSpace(fields.Error)),
	}
	if clean.ElapsedMS < 0 {
		clean.ElapsedMS = 0
	}
	if len(fields.Meta) > 0 {
		meta := make(map[string]string, len(fields.Meta))
		for key, value := range fields.Meta {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			meta[key] = sanitizeValueByKey(key, value)
		}
		if len(meta) > 0 {
			clean.Meta = meta
		}
	}
	return clean
}

func fieldsToAttrs(fields Fields) []slog.Attr {
	attrs := make([]slog.Attr, 0, 8)
	if fields.Component != "" {
		attrs = append(attrs, slog.String("component", fields.Component))
	}
	if fields.Stage != "" {
		attrs = append(attrs, slog.String("stage", fields.Stage))
	}
	if fields.Action != "" {
		attrs = append(attrs, slog.String("action", fields.Action))
	}
	if fields.Source != "" {
		attrs = append(attrs, slog.String("source", fields.Source))
	}
	if fields.Attempt > 0 {
		attrs = append(attrs, slog.Int("attempt", fields.Attempt))
	}
	if fields.ElapsedMS > 0 {
		attrs = append(attrs, slog.Int64("elapsed_ms", fields.ElapsedMS))
	}
	if fields.Error != "" {
		attrs = append(attrs, slog.String("error", fields.Error))
	}
	if len(fields.Meta) > 0 {
		keys := make([]string, 0, len(fields.Meta))
		for key := range fields.Meta {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		metaAttrs := make([]slog.Attr, 0, len(keys))
		for _, key := range keys {
			metaAttrs = append(metaAttrs, slog.String(key, fields.Meta[key]))
		}
		attrs = append(attrs, slog.Attr{Key: "meta", Value: slog.GroupValue(metaAttrs...)})
	}
	return attrs
}

func redactAttr(groups []string, attr slog.Attr) slog.Attr {
	if attr.Key == "" {
		return attr
	}
	fullKey := strings.Join(append(append([]string(nil), groups...), attr.Key), ".")
	if isSensitiveKey(fullKey) {
		attr.Value = slog.StringValue("***")
		return attr
	}
	if attr.Value.Kind() == slog.KindString {
		attr.Value = slog.StringValue(sanitizeValueByKey(attr.Key, attr.Value.String()))
	}
	return attr
}

func sanitizeValueByKey(key string, value string) string {
	if isSensitiveKey(key) {
		return "***"
	}
	return sanitizeText(strings.TrimSpace(value))
}

func sanitizeText(value string) string {
	if value == "" {
		return ""
	}
	value = reAuthorizationHdr.ReplaceAllString(value, "Authorization:***")
	value = reBearerToken.ReplaceAllString(value, "${1}***")
	value = reSensitiveKeyValue.ReplaceAllString(value, "$1$2***")
	return value
}

func isSensitiveKey(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	if key == "" {
		return false
	}
	switch {
	case strings.Contains(key, "token"),
		strings.Contains(key, "secret"),
		strings.Contains(key, "pass"),
		strings.Contains(key, "auth"),
		strings.Contains(key, "sign"),
		strings.Contains(key, "credential"),
		strings.Contains(key, "private"),
		strings.Contains(key, "cookie"),
		strings.HasSuffix(key, "_key"),
		strings.HasPrefix(key, "key_"),
		key == "key":
		return true
	default:
		return false
	}
}
