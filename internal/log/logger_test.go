package log

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  Level
	}{
		{"debug", LevelDebug},
		{"info", LevelInfo},
		{"warn", LevelWarn},
		{"error", LevelError},
		{"", LevelInfo},        // default
		{"unknown", LevelInfo}, // default
	}
	for _, tt := range tests {
		got := ParseLevel(tt.input)
		if got != tt.want {
			t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestLevel_String(t *testing.T) {
	if LevelDebug.String() != "debug" {
		t.Fail()
	}
	if LevelInfo.String() != "info" {
		t.Fail()
	}
	if LevelWarn.String() != "warn" {
		t.Fail()
	}
	if LevelError.String() != "error" {
		t.Fail()
	}
}

// TestLevel_String_Unknown verifies the default "unknown" case for unrecognised Level values.
func TestLevel_String_Unknown(t *testing.T) {
	got := Level(99).String()
	if got != "unknown" {
		t.Errorf("Level(99).String() = %q, want %q", got, "unknown")
	}
}

func TestLogger_TextFormat(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, LevelDebug, FormatText)

	log.Info("hello world", "key", "val")

	out := buf.String()
	if !strings.Contains(out, "hello world") {
		t.Fatalf("message missing from output: %q", out)
	}
	if !strings.Contains(out, "key=val") {
		t.Fatalf("key=val missing from output: %q", out)
	}
	if !strings.Contains(out, "info") {
		t.Fatalf("level missing from output: %q", out)
	}
}

func TestLogger_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, LevelDebug, FormatJSON)

	log.Error("something broke", "code", 42)

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("output not valid JSON: %v — output: %q", err, buf.String())
	}
	if entry["msg"] != "something broke" {
		t.Fatalf("msg: %v", entry["msg"])
	}
	if entry["level"] != "error" {
		t.Fatalf("level: %v", entry["level"])
	}
}

func TestLogger_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, LevelWarn, FormatText) // only warn and above

	log.Debug("this should not appear")
	log.Info("this should not appear either")
	log.Warn("this should appear")

	out := buf.String()
	if strings.Contains(out, "should not appear") {
		t.Fatal("debug/info messages leaked through warn filter")
	}
	if !strings.Contains(out, "this should appear") {
		t.Fatal("warn message missing from output")
	}
}

func TestLogger_NilWriter(t *testing.T) {
	// nil writer should not panic (writes to stderr or is discarded).
	log := New(nil, LevelInfo, FormatText)
	log.Info("no panic please") // must not panic
}

func TestLogger_With(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, LevelDebug, FormatText)
	child := log.With("component", "test")

	child.Info("child log")

	out := buf.String()
	if !strings.Contains(out, "component=test") {
		t.Fatalf("With() fields missing: %q", out)
	}
}

func TestLogger_MultipleFields(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, LevelDebug, FormatText)

	log.Debug("multi", "a", 1, "b", "two", "c", true)

	out := buf.String()
	if !strings.Contains(out, "a=1") {
		t.Fatalf("field a missing: %q", out)
	}
	if !strings.Contains(out, "b=two") {
		t.Fatalf("field b missing: %q", out)
	}
}

// TestPackageLevel_SetLevel verifies SetLevel updates the Default logger's level.
func TestPackageLevel_SetLevel(t *testing.T) {
	// Restore original level after test.
	orig := Default.level
	t.Cleanup(func() { Default.level = orig })

	SetLevel(LevelError)
	if Default.level != LevelError {
		t.Errorf("SetLevel: want LevelError, got %v", Default.level)
	}
	SetLevel(LevelDebug)
	if Default.level != LevelDebug {
		t.Errorf("SetLevel: want LevelDebug, got %v", Default.level)
	}
}

// TestPackageLevel_SetFormat verifies SetFormat updates the Default logger's format.
func TestPackageLevel_SetFormat(t *testing.T) {
	orig := Default.format
	t.Cleanup(func() { Default.format = orig })

	SetFormat(FormatJSON)
	if Default.format != FormatJSON {
		t.Errorf("SetFormat: want FormatJSON, got %v", Default.format)
	}
	SetFormat(FormatText)
	if Default.format != FormatText {
		t.Errorf("SetFormat: want FormatText, got %v", Default.format)
	}
}

// TestPackageLevel_LogFunctions verifies the package-level Debug/Info/Warn/Error
// functions route through the Default logger without panicking.
func TestPackageLevel_LogFunctions(t *testing.T) {
	var buf bytes.Buffer
	orig := Default.out
	t.Cleanup(func() { Default.out = orig })
	Default.out = &buf

	SetLevel(LevelDebug)
	Debug("debug msg", "k", "v")
	Info("info msg")
	Warn("warn msg")
	Error("error msg")

	out := buf.String()
	if !strings.Contains(out, "debug msg") {
		t.Errorf("Debug output missing: %q", out)
	}
	if !strings.Contains(out, "info msg") {
		t.Errorf("Info output missing: %q", out)
	}
	if !strings.Contains(out, "warn msg") {
		t.Errorf("Warn output missing: %q", out)
	}
	if !strings.Contains(out, "error msg") {
		t.Errorf("Error output missing: %q", out)
	}
}
