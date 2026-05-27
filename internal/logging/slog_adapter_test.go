package logging

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func TestSlogAdapter_StepDoneIncludesElapsedAndSanitizedMeta(t *testing.T) {
	var entries []Entry
	timeline := []time.Time{
		time.Unix(100, 0),
		time.Unix(101, 500*int64(time.Millisecond)),
	}
	idx := 0
	logger := NewSlogAdapter(Options{
		Writer: io.Discard,
		Now: func() time.Time {
			if idx >= len(timeline) {
				return timeline[len(timeline)-1]
			}
			t := timeline[idx]
			idx++
			return t
		},
		Sink: func(entry Entry) {
			entries = append(entries, entry)
		},
	})

	start := logger.StepStart("plugin", "download", "download_asset", "GitHub", 2, map[string]string{
		"token": "abc",
		"path":  "/tmp/plugin.zip",
	})
	logger.StepDone(start, map[string]string{
		"target": "plugin.zip",
		"auth":   "Bearer secret-token",
	})

	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
	if entries[0].Source != "github" {
		t.Fatalf("source = %q, want github", entries[0].Source)
	}
	if entries[0].Meta["token"] != "***" {
		t.Fatalf("token meta should be masked, got %q", entries[0].Meta["token"])
	}
	if entries[1].ElapsedMS != 1500 {
		t.Fatalf("elapsed = %d, want 1500", entries[1].ElapsedMS)
	}
	if entries[1].Meta["auth"] != "***" {
		t.Fatalf("auth meta should be masked, got %q", entries[1].Meta["auth"])
	}
}

func TestSlogAdapter_StepFailCarriesErrorAndMasksCredentials(t *testing.T) {
	var entries []Entry
	timeline := []time.Time{time.Unix(200, 0), time.Unix(200, 700*int64(time.Millisecond))}
	idx := 0
	logger := NewSlogAdapter(Options{
		Writer: io.Discard,
		Now: func() time.Time {
			if idx >= len(timeline) {
				return timeline[len(timeline)-1]
			}
			t := timeline[idx]
			idx++
			return t
		},
		Sink: func(entry Entry) {
			entries = append(entries, entry)
		},
	})

	start := logger.StepStart("self_update", "download_and_apply", "start", "", 0, nil)
	logger.StepFail(start, errors.New("download failed token=abcdef"), nil)

	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
	failure := entries[1]
	if failure.Level != string(LevelError) {
		t.Fatalf("level = %q, want %q", failure.Level, LevelError)
	}
	if !strings.Contains(failure.Message, "步骤失败") {
		t.Fatalf("unexpected message: %q", failure.Message)
	}
	if strings.Contains(failure.Message, "abcdef") || strings.Contains(failure.Error, "abcdef") {
		t.Fatalf("sensitive token leaked: message=%q error=%q", failure.Message, failure.Error)
	}
	if failure.ElapsedMS != 700 {
		t.Fatalf("elapsed = %d, want 700", failure.ElapsedMS)
	}
}

func TestSlogAdapter_ReplaceAttrMasksSensitiveOutput(t *testing.T) {
	var output bytes.Buffer
	logger := NewSlogAdapter(Options{Writer: &output, Format: "json"})
	logger.Info("download", Fields{
		Component: "plugin",
		Meta: map[string]string{
			"download_url": "https://example.com/file?token=abc",
			"password":     "pw123",
		},
	})

	text := output.String()
	if strings.Contains(text, "abc") || strings.Contains(text, "pw123") {
		t.Fatalf("output leaked sensitive values: %s", text)
	}
	if !strings.Contains(text, "***") {
		t.Fatalf("output should contain masked marker: %s", text)
	}
}

func TestSlogAdapter_CopiesMetaBeforeSink(t *testing.T) {
	var captured Entry
	logger := NewSlogAdapter(Options{
		Writer: io.Discard,
		Sink: func(entry Entry) {
			captured = entry
		},
	})

	meta := map[string]string{"path": "/tmp/a.zip"}
	logger.Info("copy meta", Fields{Meta: meta})
	meta["path"] = "/tmp/b.zip"

	if captured.Meta["path"] != "/tmp/a.zip" {
		t.Fatalf("captured meta mutated: %q", captured.Meta["path"])
	}
}
