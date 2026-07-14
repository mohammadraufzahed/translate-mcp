package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load(\"\") failed: %v", err)
	}
	if cfg.Server.Transport != "stdio" {
		t.Errorf("expected default transport stdio, got %s", cfg.Server.Transport)
	}
	if cfg.Server.Port != 8787 {
		t.Errorf("expected default port 8787, got %d", cfg.Server.Port)
	}
	if cfg.Cache.DefaultTTL != "24h" {
		t.Errorf("expected default cache TTL 24h, got %s", cfg.Cache.DefaultTTL)
	}
	if cfg.Translation.DefaultProvider != "openai" {
		t.Errorf("expected default provider openai, got %s", cfg.Translation.DefaultProvider)
	}
}

func TestLoadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
server:
  transport: http
  port: 9999
providers:
  openai:
    api_key: secret
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Server.Transport != "http" || cfg.Server.Port != 9999 {
		t.Errorf("server settings not loaded: %+v", cfg.Server)
	}
	if cfg.Providers["openai"].String("api_key") != "secret" {
		t.Errorf("provider config not loaded: %v", cfg.Providers["openai"].String("api_key"))
	}
}

func TestExpandEnv(t *testing.T) {
	t.Setenv("TRANSLATE_TEST_KEY", "value123")
	input := "key=${TRANSLATE_TEST_KEY}, missing=${MISSING_VAR:-fallback}"
	got := expandEnv(input)
	want := "key=value123, missing=fallback"
	if got != want {
		t.Errorf("expandEnv(%q) = %q, want %q", input, got, want)
	}
}

func TestValidateInvalidTransport(t *testing.T) {
	cfg := defaults()
	cfg.Server.Transport = "ftp"
	if err := validate(cfg); err == nil {
		t.Error("expected error for invalid transport")
	}
}

func TestProviderConfigHelpers(t *testing.T) {
	pc := ProviderConfig{
		"api_key": "abc",
		"enabled": true,
		"retries": 3,
	}
	if pc.String("api_key") != "abc" {
		t.Errorf("String failed: %q", pc.String("api_key"))
	}
	if !pc.Bool("enabled") {
		t.Error("Bool failed")
	}
	if pc.Int("retries", 0) != 3 {
		t.Errorf("Int failed: %d", pc.Int("retries", 0))
	}
	if pc.String("missing") != "" {
		t.Errorf("String default failed: %q", pc.String("missing"))
	}
}
