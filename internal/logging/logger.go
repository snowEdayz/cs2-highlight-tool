package logging

import "time"

// Level represents the normalized log level used by startup logging.
type Level string

const (
	LevelInfo  Level = "info"
	LevelWarn  Level = "warning"
	LevelError Level = "error"
)

// Fields contains structured attributes for a single log entry.
type Fields struct {
	Component string
	Stage     string
	Action    string
	Source    string
	Attempt   int
	ElapsedMS int64
	Error     string
	Meta      map[string]string
}

// Entry is the normalized structured log payload consumed by envsetup.
type Entry struct {
	Level     string
	Message   string
	Time      string
	Component string
	Stage     string
	Action    string
	Source    string
	Attempt   int
	ElapsedMS int64
	Error     string
	Meta      map[string]string
}

// StepToken carries timing/context between StepStart and StepDone/StepFail.
type StepToken struct {
	Component string
	Stage     string
	Action    string
	Source    string
	Attempt   int
	StartedAt time.Time
}

// Logger is the internal structured logging contract for startup runtime.
type Logger interface {
	Info(message string, fields Fields)
	Warn(message string, fields Fields)
	Error(message string, fields Fields)
	StepStart(component, stage, action, source string, attempt int, meta map[string]string) StepToken
	StepDone(step StepToken, meta map[string]string)
	StepFail(step StepToken, err error, meta map[string]string)
}
