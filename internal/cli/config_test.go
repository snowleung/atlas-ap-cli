package cli

import (
	"errors"
	"strings"
	"testing"
)

func TestResolveConfig_FlagPrecedence(t *testing.T) {
	environ := []string{
		"ATLAS_REMOTE_URL=https://env.example.com",
		"ATLAS_REMOTE_TOKEN=env-token",
	}

	cfg, err := ResolveConfig("https://flag.example.com/", "flag-token", environ)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server != "https://flag.example.com" {
		t.Errorf("server flag ignored: got %q", cfg.Server)
	}
	if cfg.Token != "flag-token" {
		t.Errorf("token flag ignored: got %q", cfg.Token)
	}
}

func TestResolveConfig_EnvFallback(t *testing.T) {
	environ := []string{
		"ATLAS_REMOTE_URL=https://env.example.com",
		"ATLAS_REMOTE_TOKEN=env-token",
		"PATH=/usr/bin", // unrelated
	}

	cfg, err := ResolveConfig("", "", environ)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server != "https://env.example.com" {
		t.Errorf("env URL not used: got %q", cfg.Server)
	}
	if cfg.Token != "env-token" {
		t.Errorf("env token not used: got %q", cfg.Token)
	}
}

func TestResolveConfig_PartialEnvFallback(t *testing.T) {
	environ := []string{
		"ATLAS_REMOTE_URL=https://env.example.com",
	}

	cfg, err := ResolveConfig("", "flag-token", environ)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server != "https://env.example.com" {
		t.Errorf("env URL not used: got %q", cfg.Server)
	}
	if cfg.Token != "flag-token" {
		t.Errorf("token flag should win: got %q", cfg.Token)
	}
}

func TestResolveConfig_MissingServer(t *testing.T) {
	environ := []string{
		"ATLAS_REMOTE_TOKEN=env-token",
	}

	_, err := ResolveConfig("", "", environ)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrMissingServer) {
		t.Errorf("expected ErrMissingServer, got %v", err)
	}
	if !strings.Contains(err.Error(), "ATLAS_REMOTE_URL") {
		t.Errorf("error should mention env var: %q", err.Error())
	}
}

func TestResolveConfig_TrimsTrailingSlash(t *testing.T) {
	cfg, err := ResolveConfig("https://api.example.com/", "", []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server != "https://api.example.com" {
		t.Errorf("trailing slash not trimmed: got %q", cfg.Server)
	}
}

func TestResolveConfig_OnlyPathHasSlash(t *testing.T) {
	// Server with path should still trim only trailing slash
	cfg, err := ResolveConfig("https://api.example.com/v1/", "", []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server != "https://api.example.com/v1" {
		t.Errorf("unexpected server: got %q", cfg.Server)
	}
}

func TestResolveConfig_TokenWhitespaceTrimmed(t *testing.T) {
	cfg, err := ResolveConfig("https://api.example.com", "  token-123  ", []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Token != "token-123" {
		t.Errorf("token not trimmed: got %q", cfg.Token)
	}
}

func TestResolveConfig_TokenInternalWhitespacePreserved(t *testing.T) {
	cfg, err := ResolveConfig("https://api.example.com", "ab cd", []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Token != "ab cd" {
		t.Errorf("internal whitespace stripped: got %q", cfg.Token)
	}
}

func TestResolveConfig_EmptyTokenOK(t *testing.T) {
	cfg, err := ResolveConfig("https://api.example.com", "", []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Token != "" {
		t.Errorf("expected empty token, got %q", cfg.Token)
	}
}