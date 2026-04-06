package verify_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"utilivos/sdk/internal/manifest"
	"utilivos/sdk/internal/scaffold"
	"utilivos/sdk/internal/verify"
)

func TestVerifyFixtures(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		fixture   string
		wantError bool
	}{
		{name: "valid-minimal", fixture: "valid-minimal", wantError: false},
		{name: "premium-widget", fixture: "premium-widget", wantError: false},
		{name: "invalid-missing-icon", fixture: "invalid-missing-icon", wantError: true},
		{name: "invalid-missing-tags", fixture: "invalid-missing-tags", wantError: true},
		{name: "invalid-tier", fixture: "invalid-tier", wantError: true},
		{name: "invalid-category", fixture: "invalid-category", wantError: true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := verify.Run(fixturePath(tc.fixture))
			if err != nil {
				t.Fatalf("verify run: %v", err)
			}
			gotErr := result.Report.Errors > 0
			if gotErr != tc.wantError {
				t.Fatalf("expected errors=%t, got %t (errors=%d)", tc.wantError, gotErr, result.Report.Errors)
			}
		})
	}
}

func fixturePath(name string) string {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..")
	return filepath.Join(root, "tests", "fixtures", name)
}

func TestVerifyRejectsMissingDesignSystemImports(t *testing.T) {
	t.Parallel()
	root := createScaffoldFixture(t)
	entryPath := filepath.Join(root, "tool", "index.html")
	content, err := os.ReadFile(entryPath)
	if err != nil {
		t.Fatalf("read entry html: %v", err)
	}

	updated := strings.Replace(
		string(content),
		`    <link rel="stylesheet" href="./utilivos/components.css" />`+"\n",
		"",
		1,
	)
	if err := os.WriteFile(entryPath, []byte(updated), 0o644); err != nil {
		t.Fatalf("write entry html: %v", err)
	}

	result, err := verify.Run(root)
	if err != nil {
		t.Fatalf("verify run: %v", err)
	}
	if result.Report.Errors == 0 {
		t.Fatal("expected verify errors for missing components.css import")
	}
}

func TestVerifyRejectsTokenOverrides(t *testing.T) {
	t.Parallel()
	root := createScaffoldFixture(t)
	stylePath := filepath.Join(root, "tool", "style.css")
	if err := os.WriteFile(stylePath, []byte(":root { --uv-color-primary: #ffffff; }\n"), 0o644); err != nil {
		t.Fatalf("write style override: %v", err)
	}

	result, err := verify.Run(root)
	if err != nil {
		t.Fatalf("verify run: %v", err)
	}
	if result.Report.Errors == 0 {
		t.Fatal("expected verify errors for token override")
	}
}

func TestVerifyRejectsMissingLocaleFile(t *testing.T) {
	t.Parallel()
	root := createScaffoldFixture(t)
	manifestPath := filepath.Join(root, "manifest.json")
	parsed, err := manifest.Load(manifestPath)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	parsed.Locales = []string{"en", "fr"}
	if err := manifest.Save(manifestPath, parsed); err != nil {
		t.Fatalf("save manifest: %v", err)
	}

	result, err := verify.Run(root)
	if err != nil {
		t.Fatalf("verify run: %v", err)
	}
	if result.Report.Errors == 0 {
		t.Fatal("expected verify errors for missing locale file")
	}
}

func TestVerifyRejectsHardcodedLocalizedStrings(t *testing.T) {
	t.Parallel()
	root := createScaffoldFixture(t)
	mainPath := filepath.Join(root, "tool", "main.js")
	if err := os.WriteFile(mainPath, []byte(`(() => {
  const output = document.getElementById("output");
  if (!output) return;
  output.textContent = "Hello world";
})();
`), 0o644); err != nil {
		t.Fatalf("write main js: %v", err)
	}

	result, err := verify.Run(root)
	if err != nil {
		t.Fatalf("verify run: %v", err)
	}
	if result.Report.Errors == 0 {
		t.Fatal("expected verify errors for hardcoded localized strings")
	}
}

func TestVerifyAllowsMissingGeneratedDistDirectory(t *testing.T) {
	t.Parallel()
	root := createScaffoldFixture(t)
	if err := os.RemoveAll(filepath.Join(root, "dist")); err != nil {
		t.Fatalf("remove dist: %v", err)
	}

	result, err := verify.Run(root)
	if err != nil {
		t.Fatalf("verify run: %v", err)
	}

	for _, issue := range result.Report.Issues {
		if issue.Path == "dist" && issue.Severity == "error" {
			t.Fatalf("dist should not be treated as required source, got issue: %+v", issue)
		}
	}
}

func TestVerifyRejectsMissingRequiredSourceFile(t *testing.T) {
	t.Parallel()
	root := createScaffoldFixture(t)
	if err := os.Remove(filepath.Join(root, "tool", "main.js")); err != nil {
		t.Fatalf("remove main.js: %v", err)
	}

	result, err := verify.Run(root)
	if err != nil {
		t.Fatalf("verify run: %v", err)
	}
	if result.Report.Errors == 0 {
		t.Fatal("expected verify errors for missing required source file")
	}

	found := false
	for _, issue := range result.Report.Issues {
		if issue.Code == "structure.required_source" && issue.Path == "tool/main.js" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected structure.required_source issue for tool/main.js, got %+v", result.Report.Issues)
	}
}

func TestVerifyScaffoldIncludesSessionBridge(t *testing.T) {
	t.Parallel()
	root := createScaffoldFixture(t)
	result, err := verify.Run(root)
	if err != nil {
		t.Fatalf("verify run: %v", err)
	}
	for _, issue := range result.Report.Issues {
		if issue.Code == "session.persist_bridge" {
			t.Fatalf("unexpected session warning on default scaffold: %+v", issue)
		}
	}
}

func TestVerifyWarnsWhenSessionBridgeMissing(t *testing.T) {
	t.Parallel()
	root := createScaffoldFixture(t)
	mainPath := filepath.Join(root, "tool", "main.js")
	b, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("read main.js: %v", err)
	}
	updated := strings.ReplaceAll(string(b), "window.UtilivosRuntime.syncState", "window.UtilivoIgnored.syncState")
	if err := os.WriteFile(mainPath, []byte(updated), 0o644); err != nil {
		t.Fatalf("write main.js: %v", err)
	}

	result, err := verify.Run(root)
	if err != nil {
		t.Fatalf("verify run: %v", err)
	}
	found := false
	for _, issue := range result.Report.Issues {
		if issue.Code == "session.persist_bridge" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected session.persist_bridge warning, issues=%+v", result.Report.Issues)
	}
}

func TestVerifyRejectsExcludingRequiredPackagedFile(t *testing.T) {
	t.Parallel()
	root := createScaffoldFixture(t)
	if err := os.WriteFile(filepath.Join(root, "build.ignore"), []byte("tool/main.js\n"), 0o644); err != nil {
		t.Fatalf("write build.ignore: %v", err)
	}

	result, err := verify.Run(root)
	if err != nil {
		t.Fatalf("verify run: %v", err)
	}
	if result.Report.Errors == 0 {
		t.Fatal("expected verify errors for excluded required packaged file")
	}

	found := false
	for _, issue := range result.Report.Issues {
		if issue.Code == "archive.required_package" && issue.Path == "tool/main.js" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected archive.required_package issue for tool/main.js, got %+v", result.Report.Issues)
	}
}

func createScaffoldFixture(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "fixture-plugin")
	if _, err := scaffold.Create(scaffold.Options{TargetPath: root}); err != nil {
		t.Fatalf("create scaffold: %v", err)
	}
	return root
}
