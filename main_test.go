package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// --- parseLogLevel tests ---

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"DEBUG", DEBUG},
		{"debug", DEBUG},
		{"INFO", INFO},
		{"info", INFO},
		{"WARN", WARN},
		{"warn", WARN},
		{"ERROR", ERROR},
		{"error", ERROR},
		{"", INFO},
		{"UNKNOWN", INFO},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseLogLevel(tt.input)
			if got != tt.expected {
				t.Errorf("parseLogLevel(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

// --- verifySignature tests ---

func computeSignature(payload []byte, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestVerifySignature(t *testing.T) {
	secret := []byte("test-secret")
	payload := []byte(`{"action":"opened"}`)

	t.Run("valid signature", func(t *testing.T) {
		webhookSecret = secret
		sig := computeSignature(payload, secret)
		if !verifySignature(payload, sig) {
			t.Error("expected valid signature to pass")
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		webhookSecret = secret
		if verifySignature(payload, "sha256=invalidsignature") {
			t.Error("expected invalid signature to fail")
		}
	})

	t.Run("missing sha256 prefix", func(t *testing.T) {
		webhookSecret = secret
		mac := hmac.New(sha256.New, secret)
		mac.Write(payload)
		rawSig := hex.EncodeToString(mac.Sum(nil))
		if verifySignature(payload, rawSig) {
			t.Error("expected signature without sha256= prefix to fail")
		}
	})

	t.Run("empty signature with secret set", func(t *testing.T) {
		webhookSecret = secret
		if verifySignature(payload, "") {
			t.Error("expected empty signature to fail when secret is set")
		}
	})

	t.Run("no secret configured skips verification", func(t *testing.T) {
		webhookSecret = nil
		if !verifySignature(payload, "") {
			t.Error("expected verification to pass when no secret is configured")
		}
	})

	// Restore for subsequent tests
	webhookSecret = nil
}

// --- loadEventConfig tests ---

func TestLoadEventConfig(t *testing.T) {
	t.Run("valid config file", func(t *testing.T) {
		cfg := []EventConfig{
			{EventType: "push", Channel: "chan-push"},
			{EventType: "pull_request", Channel: "chan-pr"},
		}
		data, _ := json.Marshal(cfg)

		f, err := os.CreateTemp("", "config-*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(f.Name())
		f.Write(data)
		f.Close()

		// Reset globals
		eventConfigs = nil
		eventChannelMap = nil

		if err := loadEventConfig(f.Name()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(eventConfigs) != 2 {
			t.Errorf("expected 2 event configs, got %d", len(eventConfigs))
		}
		if eventChannelMap["push"] != "chan-push" {
			t.Errorf("expected chan-push, got %q", eventChannelMap["push"])
		}
		if eventChannelMap["pull_request"] != "chan-pr" {
			t.Errorf("expected chan-pr, got %q", eventChannelMap["pull_request"])
		}
	})

	t.Run("missing file returns error", func(t *testing.T) {
		if err := loadEventConfig("/nonexistent/config.json"); err == nil {
			t.Error("expected error for missing file")
		}
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		f, err := os.CreateTemp("", "bad-config-*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(f.Name())
		f.WriteString("not valid json")
		f.Close()

		if err := loadEventConfig(f.Name()); err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

// --- webhookHandler tests ---

func setupWebhookHandler(t *testing.T, cfgEntries []EventConfig) {
	t.Helper()
	// Disable Redis for all handler tests
	redisClient = nil
	// Clear secret so signature verification is skipped
	webhookSecret = nil
	// Load event config map
	eventChannelMap = make(map[string]string)
	for _, e := range cfgEntries {
		eventChannelMap[e.EventType] = e.Channel
	}
}

func TestWebhookHandler_MethodNotAllowed(t *testing.T) {
	setupWebhookHandler(t, nil)

	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		req := httptest.NewRequest(method, "/webhook", nil)
		rec := httptest.NewRecorder()
		webhookHandler(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("method %s: expected 405, got %d", method, rec.Code)
		}
	}
}

func TestWebhookHandler_InvalidSignature(t *testing.T) {
	setupWebhookHandler(t, nil)
	webhookSecret = []byte("secret")
	defer func() { webhookSecret = nil }()

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBufferString(`{}`))
	req.Header.Set("X-Hub-Signature-256", "sha256=badsig")
	rec := httptest.NewRecorder()
	webhookHandler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestWebhookHandler_EventNotConfigured(t *testing.T) {
	setupWebhookHandler(t, []EventConfig{{EventType: "push", Channel: "chan-push"}})

	body := `{"ref":"refs/heads/main"}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBufferString(body))
	req.Header.Set("X-GitHub-Event", "issues") // not in config
	rec := httptest.NewRecorder()
	webhookHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "Webhook received but event type not configured" {
		t.Errorf("unexpected body: %q", rec.Body.String())
	}
}

func TestWebhookHandler_InvalidJSON(t *testing.T) {
	setupWebhookHandler(t, []EventConfig{{EventType: "push", Channel: "chan-push"}})

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBufferString("not json"))
	req.Header.Set("X-GitHub-Event", "push")
	rec := httptest.NewRecorder()
	webhookHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestWebhookHandler_Success(t *testing.T) {
	setupWebhookHandler(t, []EventConfig{{EventType: "push", Channel: "chan-push"}})

	body := `{"ref":"refs/heads/main"}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBufferString(body))
	req.Header.Set("X-GitHub-Event", "push")
	rec := httptest.NewRecorder()
	webhookHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "Webhook received" {
		t.Errorf("unexpected body: %q", rec.Body.String())
	}
}

func TestWebhookHandler_SuccessWithSignature(t *testing.T) {
	setupWebhookHandler(t, []EventConfig{{EventType: "push", Channel: "chan-push"}})
	secret := []byte("my-secret")
	webhookSecret = secret
	defer func() { webhookSecret = nil }()

	body := []byte(`{"ref":"refs/heads/main"}`)
	sig := computeSignature(body, secret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-Hub-Signature-256", sig)
	rec := httptest.NewRecorder()
	webhookHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "Webhook received" {
		t.Errorf("unexpected body: %q", rec.Body.String())
	}
}
