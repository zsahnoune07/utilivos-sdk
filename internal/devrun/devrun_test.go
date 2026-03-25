package devrun_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/utilivo/utilivo-sdk/internal/devrun"
	"github.com/utilivo/utilivo-sdk/internal/scaffold"
)

func TestServerStartsAndServesDevRuntime(t *testing.T) {
	t.Parallel()
	project := scaffoldProject(t)

	server, err := devrun.Start(devrun.Options{Root: project, Host: "127.0.0.1", Port: 0, Theme: "light"})
	if err != nil {
		t.Fatalf("start dev runtime: %v", err)
	}
	defer func() {
		_ = server.Close()
	}()

	client := &http.Client{Timeout: 5 * time.Second}

	shellBody := mustGET(t, client, server.URL()+"/")
	if !strings.Contains(shellBody, "utilivo run") {
		t.Fatalf("shell response missing utilivo run marker: %s", shellBody)
	}
	if !strings.Contains(shellBody, server.EntryPath()) {
		t.Fatalf("shell response missing entry path %q", server.EntryPath())
	}

	entryBody := mustGET(t, client, server.URL()+"/"+server.EntryPath())
	for _, expected := range []string{"UtilivoHost", "__uv-dev-error-overlay", "/__utilivo__/live"} {
		if !strings.Contains(entryBody, expected) {
			t.Fatalf("entry response missing %q", expected)
		}
	}

	manifestBody := mustGET(t, client, server.URL()+"/__utilivo__/manifest")
	var payload struct {
		Manifest struct {
			Slug string `json:"slug"`
		} `json:"manifest"`
		Runtime struct {
			Mode string `json:"mode"`
		} `json:"runtime"`
	}
	if err := json.Unmarshal([]byte(manifestBody), &payload); err != nil {
		t.Fatalf("parse manifest payload: %v", err)
	}
	if payload.Manifest.Slug == "" {
		t.Fatal("expected manifest slug in runtime payload")
	}
	if payload.Runtime.Mode != "dev" {
		t.Fatalf("expected runtime mode dev, got %q", payload.Runtime.Mode)
	}
}

func TestLiveEndpointDetectsRevisionChanges(t *testing.T) {
	t.Parallel()
	project := scaffoldProject(t)

	server, err := devrun.Start(devrun.Options{Root: project, Host: "127.0.0.1", Port: 0, Theme: "system"})
	if err != nil {
		t.Fatalf("start dev runtime: %v", err)
	}
	defer func() {
		_ = server.Close()
	}()

	initial := server.Revision()
	if initial == "" {
		t.Fatal("expected initial revision")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	endpoint := server.URL() + "/__utilivo__/live?since=" + url.QueryEscape(initial)

	type result struct {
		payload devrun.LiveResponse
		err     error
	}
	resultCh := make(chan result, 1)
	go func() {
		body := mustGET(t, client, endpoint)
		var live devrun.LiveResponse
		err := json.Unmarshal([]byte(body), &live)
		resultCh <- result{payload: live, err: err}
	}()

	mainPath := filepath.Join(project, "tool", "main.js")
	original, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("read main.js: %v", err)
	}
	updated := string(original) + "\n// test live revision touch\n"
	if err := os.WriteFile(mainPath, []byte(updated), 0o644); err != nil {
		t.Fatalf("write main.js: %v", err)
	}

	select {
	case outcome := <-resultCh:
		if outcome.err != nil {
			t.Fatalf("decode live response: %v", outcome.err)
		}
		if !outcome.payload.Changed {
			t.Fatalf("expected changed=true, got %+v", outcome.payload)
		}
		if outcome.payload.Revision == initial {
			t.Fatalf("expected revision to change from %s", initial)
		}
	case <-time.After(12 * time.Second):
		t.Fatal("timed out waiting for live endpoint to detect file change")
	}
}

func TestPreludeInjectsHostMetadataAndOverlayBootstrap(t *testing.T) {
	t.Parallel()
	project := scaffoldProject(t)

	server, err := devrun.Start(devrun.Options{Root: project, Host: "127.0.0.1", Port: 0, Theme: "dark"})
	if err != nil {
		t.Fatalf("start dev runtime: %v", err)
	}
	defer func() {
		_ = server.Close()
	}()

	client := &http.Client{Timeout: 5 * time.Second}
	entryBody := mustGET(t, client, server.URL()+"/"+server.EntryPath())

	for _, expected := range []string{
		`Object.defineProperty(window, "UtilivoHost"`,
		"deepFreeze",
		`mode: "dev"`,
		"__uv-dev-error-overlay",
		"Fix the error and save. utilivo run reloads the tool automatically.",
		"/__utilivo__/live",
	} {
		if !strings.Contains(entryBody, expected) {
			t.Fatalf("entry response missing %q", expected)
		}
	}
}

func scaffoldProject(t *testing.T) string {
	t.Helper()
	tempRoot := t.TempDir()
	project := filepath.Join(tempRoot, "run-plugin")
	if _, err := scaffold.Create(scaffold.Options{TargetPath: project}); err != nil {
		t.Fatalf("create scaffold: %v", err)
	}
	return project
}

func mustGET(t *testing.T, client *http.Client, target string) string {
	t.Helper()
	resp, err := client.Get(target)
	if err != nil {
		t.Fatalf("GET %s: %v", target, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET %s status %d: %s", target, resp.StatusCode, string(body))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read GET %s body: %v", target, err)
	}
	return string(body)
}
