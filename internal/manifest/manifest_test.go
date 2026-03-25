package manifest_test

import (
	"strings"
	"testing"

	"github.com/utilivo/utilivo-sdk/internal/manifest"
)

func TestValidate_DefaultManifestIsValid(t *testing.T) {
	t.Parallel()
	m := manifest.Default("Sample", "sample-plugin", "Tester")
	r := manifest.Validate(m)
	if r.Errors != 0 {
		t.Fatalf("expected zero errors, got %d", r.Errors)
	}
}

func TestValidate_InvalidManifestFails(t *testing.T) {
	t.Parallel()
	m := manifest.Default("Sample", "sample-plugin", "Tester")
	m.Slug = "Not Allowed"
	m.Category = "unknown"
	m.Tier = "enterprise"
	m.Tags = []string{}
	r := manifest.Validate(m)
	if r.Errors == 0 {
		t.Fatal("expected validation errors")
	}
}

func TestBumpVersion(t *testing.T) {
	t.Parallel()
	v, err := manifest.BumpVersion("1.2.3", "minor")
	if err != nil {
		t.Fatalf("bump version: %v", err)
	}
	if v != "1.3.0" {
		t.Fatalf("expected 1.3.0, got %s", v)
	}
}

func TestValidate_DuplicateTagsFail(t *testing.T) {
	t.Parallel()
	m := manifest.Default("Sample", "sample-plugin", "Tester")
	m.Tags = []string{"starter", "starter", "utilivo"}
	r := manifest.Validate(m)
	if r.Errors == 0 {
		t.Fatal("expected duplicate tag validation error")
	}
}

func TestValidate_DarkModeAndDesignSystemRequired(t *testing.T) {
	t.Parallel()
	m := manifest.Default("Sample", "sample-plugin", "Tester")
	m.DesignSystemVersion = 0
	m.SupportsDarkMode = false
	r := manifest.Validate(m)
	if r.Errors == 0 {
		t.Fatal("expected design system and dark mode validation errors")
	}
}

func TestValidate_LocalesMustDefaultToEnglish(t *testing.T) {
	t.Parallel()
	m := manifest.Default("Sample", "sample-plugin", "Tester")
	m.Locales = []string{"fr"}
	m.DefaultLocale = "fr"
	r := manifest.Validate(m)
	if r.Errors == 0 {
		t.Fatal("expected locale validation errors")
	}
}

func TestParseJSON_OptionalFieldsAreAccepted(t *testing.T) {
	t.Parallel()
	raw := `{
  "id": "com.utilivo.plugin.sample-plugin",
  "name": "Sample Plugin",
  "slug": "sample-plugin",
  "version": "1.0.0",
  "description": "Sample",
  "author": "Tester",
  "entry": "tool/index.html",
  "icon": "tool/assets/icon.png",
  "thumb": "tool/assets/thumb.png",
  "category": "utilities",
  "tags": ["starter", "tool", "utilivo"],
  "tier": "free",
  "permissions": ["storage"],
  "ui_type": "page",
  "api_version": "1",
  "screenshots": ["tool/assets/screenshots/home.png"],
  "min_app_version": "1.0.0",
  "homepage": "https://example.com",
  "repository": "https://example.com/repo",
  "license": "MIT",
  "keywords": ["tool", "plugin"],
  "locales": ["en"],
  "default_locale": "en",
  "content_security_policy": {
    "script_src": ["'self'"],
    "connect_src": ["'self'"],
    "img_src": ["'self'", "data:"]
  },
  "design_system_version": 1,
  "supports_dark_mode": true
}`
	m, err := manifest.ParseJSON([]byte(raw))
	if err != nil {
		t.Fatalf("parse manifest with optional fields: %v", err)
	}
	if m.DefaultLocale != "en" || !strings.Contains(m.Repository, "example.com") {
		t.Fatalf("unexpected parsed optional fields: %+v", m)
	}
}
