package scaffold_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"utilivo/sdk/internal/manifest"
	"utilivo/sdk/internal/scaffold"
	"utilivo/sdk/internal/util"
)

func TestCreateScaffold(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	target := filepath.Join(dir, "my-plugin")

	created, err := scaffold.Create(scaffold.Options{TargetPath: target})
	if err != nil {
		t.Fatalf("create scaffold: %v", err)
	}
	if created != target {
		t.Fatalf("expected created path %s, got %s", target, created)
	}

	required := []string{
		"manifest.json",
		"AGENTS.md",
		"build.ignore",
		"README.md",
		"PLAN.md",
		".env.example",
		"tool/index.html",
		"tool/main.js",
		"tool/style.css",
		"tool/locales/en.json",
		"tool/utilivo/base.css",
		"tool/utilivo/components.css",
		"tool/utilivo/layout.css",
		"tool/utilivo/utilities.css",
		"tool/utilivo/patterns.css",
		"tool/utilivo/tool.js",
		"tool/assets/icon.png",
		"tool/assets/thumb.png",
		"tool/assets/screenshots/home.png",
		"tests",
		"dist",
		".utilivo/project.json",
	}
	for _, rel := range required {
		if !util.Exists(filepath.Join(created, rel)) {
			t.Fatalf("missing required scaffold path: %s", rel)
		}
	}

	m, err := manifest.Load(filepath.Join(created, "manifest.json"))
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	r := manifest.Validate(m)
	if r.Errors != 0 {
		t.Fatalf("expected valid manifest, got %d errors", r.Errors)
	}

	agents, err := os.ReadFile(filepath.Join(created, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read agents: %v", err)
	}
	agentsText := string(agents)
	for _, expected := range []string{
		"operating contract",
		"Implement and validate; defer packaging",
		"manifest-driven behavior",
		"manifest.json",
		"PLUGIN UI RULES",
		"cannot hardcode product colors or redefine --uv-* tokens",
		"Lead with the primary workflow",
		"must not run utilivo build",
	} {
		if !strings.Contains(agentsText, expected) {
			t.Fatalf("agents template missing expected text: %q", expected)
		}
	}

	buildIgnore, err := os.ReadFile(filepath.Join(created, "build.ignore"))
	if err != nil {
		t.Fatalf("read build.ignore: %v", err)
	}
	buildIgnoreText := string(buildIgnore)
	for _, expected := range []string{
		"README.md",
		"PLAN.md",
	} {
		if !strings.Contains(buildIgnoreText, expected) {
			t.Fatalf("build.ignore missing expected entry: %q", expected)
		}
	}

	indexHTML, err := os.ReadFile(filepath.Join(created, "tool/index.html"))
	if err != nil {
		t.Fatalf("read index html: %v", err)
	}
	indexText := string(indexHTML)
	for _, expected := range []string{
		"./utilivo/base.css",
		"./utilivo/layout.css",
		"./utilivo/components.css",
		"./utilivo/tool.js",
		"uv-tool-page",
		"data-i18n=",
		"data-i18n-placeholder=",
	} {
		if !strings.Contains(indexText, expected) {
			t.Fatalf("index template missing expected text: %q", expected)
		}
	}

	mainJS, err := os.ReadFile(filepath.Join(created, "tool/main.js"))
	if err != nil {
		t.Fatalf("read main js: %v", err)
	}
	mainText := string(mainJS)
	for _, expected := range []string{
		"initLocalization",
		"./locales",
		`ui.t("result.ready")`,
	} {
		if !strings.Contains(mainText, expected) {
			t.Fatalf("main template missing expected text: %q", expected)
		}
	}
}

func TestScaffoldIncludesFrameworkPrimitivesAndThemeGuards(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	target := filepath.Join(dir, "framework-plugin")
	if _, err := scaffold.Create(scaffold.Options{TargetPath: target}); err != nil {
		t.Fatalf("create scaffold: %v", err)
	}

	baseCSS, err := os.ReadFile(filepath.Join(target, "tool", "utilivo", "base.css"))
	if err != nil {
		t.Fatalf("read base.css: %v", err)
	}
	baseText := string(baseCSS)
	for _, expected := range []string{
		":root[data-uv-theme=\"dark\"]",
		"prefers-reduced-motion",
		"--uv-motion-fast",
		"--uv-border-subtle",
		":focus-visible",
	} {
		if !strings.Contains(baseText, expected) {
			t.Fatalf("base.css missing expected token/style marker: %q", expected)
		}
	}

	utilitiesCSS, err := os.ReadFile(filepath.Join(target, "tool", "utilivo", "utilities.css"))
	if err != nil {
		t.Fatalf("read utilities.css: %v", err)
	}
	utilitiesText := string(utilitiesCSS)
	for _, expected := range []string{
		".uv-mt-1",
		".uv-d-flex",
		".uv-truncate",
		".uv-is-selected",
		".uv-hidden-mobile",
	} {
		if !strings.Contains(utilitiesText, expected) {
			t.Fatalf("utilities.css missing expected utility marker: %q", expected)
		}
	}

	componentsCSS, err := os.ReadFile(filepath.Join(target, "tool", "utilivo", "components.css"))
	if err != nil {
		t.Fatalf("read components.css: %v", err)
	}
	componentText := string(componentsCSS)
	for _, expected := range []string{
		"@import \"./utilities.css\";",
		"@import \"./patterns.css\";",
		".uv-textarea",
		".uv-switch",
		".uv-tabs",
		".uv-modal",
		".uv-dropdown",
		".uv-skeleton",
	} {
		if !strings.Contains(componentText, expected) {
			t.Fatalf("components.css missing expected component marker: %q", expected)
		}
	}

	patternsCSS, err := os.ReadFile(filepath.Join(target, "tool", "utilivo", "patterns.css"))
	if err != nil {
		t.Fatalf("read patterns.css: %v", err)
	}
	patternsText := string(patternsCSS)
	for _, expected := range []string{
		".uv-pattern-settings",
		".uv-pattern-form",
		".uv-pattern-results",
		".uv-pattern-dashboard",
		".uv-pattern-search-list",
		".uv-pattern-detail",
		".uv-action-bar",
		"@media (max-width:",
	} {
		if !strings.Contains(patternsText, expected) {
			t.Fatalf("patterns.css missing expected pattern marker: %q", expected)
		}
	}

	layoutCSS, err := os.ReadFile(filepath.Join(target, "tool", "utilivo", "layout.css"))
	if err != nil {
		t.Fatalf("read layout.css: %v", err)
	}
	layoutText := string(layoutCSS)
	for _, expected := range []string{
		".uv-stack",
		".uv-inline",
		".uv-grid",
		".uv-split",
		"@media (max-width:",
	} {
		if !strings.Contains(layoutText, expected) {
			t.Fatalf("layout.css missing expected layout marker: %q", expected)
		}
	}
}

func TestScaffoldKeepsCustomCSSMinimal(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	target := filepath.Join(dir, "minimal-css-plugin")
	if _, err := scaffold.Create(scaffold.Options{TargetPath: target}); err != nil {
		t.Fatalf("create scaffold: %v", err)
	}

	styleCSS, err := os.ReadFile(filepath.Join(target, "tool", "style.css"))
	if err != nil {
		t.Fatalf("read style.css: %v", err)
	}
	lines := strings.Split(string(styleCSS), "\n")
	meaningful := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "*/") {
			continue
		}
		meaningful++
	}
	if meaningful > 2 {
		t.Fatalf("expected minimal custom CSS, found %d meaningful line(s)", meaningful)
	}
}
