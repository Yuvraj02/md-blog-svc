package logger_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/marketing-digest/pkg/logger"
)

func TestNewEmitsJSONWithService(t *testing.T) {
	var buf bytes.Buffer
	log := logger.NewWithWriter("auth", &buf, slog.LevelInfo)
	log.Info("hello", "request_id", "abc")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("expected JSON log: %v", err)
	}
	if entry["service"] != "auth" {
		t.Fatalf("expected service=auth, got %v", entry["service"])
	}
	if entry["request_id"] != "abc" {
		t.Fatalf("expected request_id=abc, got %v", entry["request_id"])
	}
}
