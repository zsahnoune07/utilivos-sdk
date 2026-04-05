package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"utilivo/sdk/internal/cli"
	"utilivo/sdk/internal/util"
)

func TestVersionCommand(t *testing.T) {
	t.Parallel()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code, err := cli.Run([]string{"utilivo", "version"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run version: %v", err)
	}
	if code != util.ExitOK {
		t.Fatalf("expected exit code %d, got %d", util.ExitOK, code)
	}
	if got := stdout.String(); got != util.Version+"\n" {
		t.Fatalf("unexpected version output: %q", got)
	}
}

func TestVersionFlag(t *testing.T) {
	t.Parallel()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code, err := cli.Run([]string{"utilivo", "--version"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run --version: %v", err)
	}
	if code != util.ExitOK {
		t.Fatalf("expected exit code %d, got %d", util.ExitOK, code)
	}
	if got := stdout.String(); got != util.Version+"\n" {
		t.Fatalf("unexpected version output: %q", got)
	}
}

func TestVersionJSON(t *testing.T) {
	t.Parallel()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code, err := cli.Run([]string{"utilivo", "version", "--json"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run version --json: %v", err)
	}
	if code != util.ExitOK {
		t.Fatalf("expected exit code %d, got %d", util.ExitOK, code)
	}
	var payload map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("parse json output: %v", err)
	}
	if payload["version"] != util.Version {
		t.Fatalf("expected version %q, got %q", util.Version, payload["version"])
	}
}

func TestHelpIncludesRunCommand(t *testing.T) {
	t.Parallel()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code, err := cli.Run([]string{"utilivo", "--help"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run --help: %v", err)
	}
	if code != util.ExitOK {
		t.Fatalf("expected exit code %d, got %d", util.ExitOK, code)
	}
	if !strings.Contains(stdout.String(), "run        Start local browser-ready dev runtime") {
		t.Fatalf("help output missing run command: %s", stdout.String())
	}
}

func TestRunAliasRejectsInvalidTheme(t *testing.T) {
	t.Parallel()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code, err := cli.Run([]string{"run", "--theme", "invalid-theme"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected invalid theme error")
	}
	if code != util.ExitUsage {
		t.Fatalf("expected exit code %d, got %d", util.ExitUsage, code)
	}
	if !strings.Contains(err.Error(), "invalid --theme value") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestManifestUpdateDryRun(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	raw := `{
  "id": "com.utilivo.plugin.sample-plugin",
  "name": "Sample Plugin",
  "slug": "sample-plugin",
  "version": "1.0.0",
  "description": "Test",
  "author": "T",
  "entry": "tool/index.html",
  "icon": "tool/assets/icon.png",
  "thumb": "tool/assets/thumb.png",
  "category": "utilities",
  "tags": ["a", "b", "c"],
  "tier": "free",
  "permissions": ["storage"],
  "ui_type": "page",
  "api_version": "1",
  "screenshots": ["tool/assets/screenshots/home.png"],
  "min_app_version": "1.0.0"
}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code, err := cli.Run([]string{"utilivo", "manifest", "update", "--dir", dir, "--dry-run"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if code != util.ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "manifest_schema_version") && !strings.Contains(out, "design_system_version") {
		t.Fatalf("expected schema-related fields in output: %q", out)
	}
}

func TestManifestSyncDryRun(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	raw := `{
  "id": "com.utilivo.plugin.sample-plugin",
  "name": "Sample Plugin",
  "slug": "sample-plugin",
  "version": "1.0.0",
  "description": "Test",
  "author": "T",
  "entry": "tool/index.html",
  "icon": "tool/assets/icon.png",
  "thumb": "tool/assets/thumb.png",
  "category": "utilities",
  "tags": ["a", "b", "c"],
  "tier": "free",
  "permissions": ["storage"],
  "ui_type": "page",
  "api_version": "1",
  "screenshots": ["tool/assets/screenshots/home.png"],
  "min_app_version": "1.0.0"
}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code, err := cli.Run([]string{"utilivo", "manifest", "sync", "--dir", dir, "--dry-run"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if code != util.ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if stdout.Len() == 0 {
		t.Fatal("expected output")
	}
}

func TestRunAliasDispatchesAndFailsOnMissingProjectDir(t *testing.T) {
	t.Parallel()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	missingRoot := filepath.Join(t.TempDir(), "does-not-exist")
	if _, err := os.Stat(missingRoot); !os.IsNotExist(err) {
		t.Fatalf("expected missing directory for test setup, got err=%v", err)
	}

	code, err := cli.Run([]string{"run", "--dir", missingRoot, "--no-open"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected run failure for missing project directory")
	}
	if code != util.ExitBuildFailure {
		t.Fatalf("expected exit code %d, got %d", util.ExitBuildFailure, code)
	}
	if !strings.Contains(err.Error(), "project directory does not exist") {
		t.Fatalf("unexpected error: %v", err)
	}
}
