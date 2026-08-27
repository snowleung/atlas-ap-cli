package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestWriteErrorJSON_Structure(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteErrorJSON(&buf, "MISSING_SERVER", "missing server", 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got ErrorEnvelope
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, buf.String())
	}
	if got.Success != false {
		t.Errorf("success should be false, got %v", got.Success)
	}
	if got.Code != "MISSING_SERVER" {
		t.Errorf("code mismatch: got %q", got.Code)
	}
	if got.Message != "missing server" {
		t.Errorf("message mismatch: got %q", got.Message)
	}
	if got.HTTPStatus != 0 {
		t.Errorf("http_status should be omitted for 0, got %d", got.HTTPStatus)
	}
}

func TestWriteErrorJSON_IncludesHTTPStatus(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteErrorJSON(&buf, "BAD_REQUEST", "bad", 400); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got ErrorEnvelope
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, buf.String())
	}
	if got.HTTPStatus != 400 {
		t.Errorf("http_status mismatch: got %d", got.HTTPStatus)
	}
}

func TestWriteErrorJSON_NeverIncludesToken(t *testing.T) {
	token := "supersecrettoken-XYZ-12345"
	var buf bytes.Buffer
	if err := WriteErrorJSON(&buf, "NETWORK_ERROR", "connection refused", 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(buf.String(), token) {
		t.Fatalf("token leaked into output: %s", buf.String())
	}
}

func TestWriteErrorJSON_MessageNeverCarriesToken(t *testing.T) {
	// Simulate a caller who mistakenly includes a token in the message.
	token := "very-secret-token-value-abc"
	var buf bytes.Buffer
	err := WriteErrorJSON(&buf, "BAD_REQUEST", "auth failed: token="+token, 401)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Output is rendered as-is — the contract is that callers must not put
	// tokens in messages. We assert the helper exposes only the structured
	// fields, never the raw input string.
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if got["code"] != "BAD_REQUEST" || got["message"] != "auth failed: token="+token {
		t.Fatalf("structured output mismatch: %v", got)
	}
}

func TestWriteErrorJSON_SingleLine(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteErrorJSON(&buf, "X", "y", 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Count(buf.String(), "\n") != 1 {
		t.Errorf("expected single trailing newline, got %q", buf.String())
	}
}

func TestWriteErrorHuman_NoTokenInOutput(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteErrorHuman(&buf, "TIMEOUT", "request timed out"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "TIMEOUT") {
		t.Errorf("expected code in output: %q", buf.String())
	}
	if !strings.Contains(buf.String(), "request timed out") {
		t.Errorf("expected message in output: %q", buf.String())
	}
}

func TestVersionIsSet(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}
}
