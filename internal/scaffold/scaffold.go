package scaffold

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"utilivos/sdk/internal/manifest"
	"utilivos/sdk/internal/templates"
	"utilivos/sdk/internal/util"
)

type Options struct {
	TargetPath  string
	Name        string
	Slug        string
	Category    string
	Tier        string
	Author      string
	Description string
	Tags        []string
	Template    string
	RuntimeMode string
}

func Create(opts Options) (string, error) {
	if strings.TrimSpace(opts.TargetPath) == "" {
		return "", errors.New("target path is required")
	}

	absTarget, err := filepath.Abs(opts.TargetPath)
	if err != nil {
		return "", err
	}

	if util.Exists(absTarget) {
		entries, err := os.ReadDir(absTarget)
		if err != nil {
			return "", err
		}
		if len(entries) > 0 {
			return "", fmt.Errorf("target directory is not empty: %s", absTarget)
		}
	} else if err := os.MkdirAll(absTarget, 0o755); err != nil {
		return "", err
	}

	defaultName := filepath.Base(absTarget)
	if strings.TrimSpace(opts.Name) == "" {
		opts.Name = humanize(defaultName)
	}
	if strings.TrimSpace(opts.Slug) == "" {
		opts.Slug = util.NormalizeSlug(defaultName)
	}
	if !manifest.IsValidSlug(opts.Slug) {
		return "", fmt.Errorf("invalid slug %q", opts.Slug)
	}
	if strings.TrimSpace(opts.Author) == "" {
		opts.Author = "Utilivos Developer"
	}
	if len(opts.Tags) == 0 {
		opts.Tags = []string{"starter", "tool", "utilivos"}
	}
	sort.Strings(opts.Tags)

	m := manifest.Default(opts.Name, opts.Slug, opts.Author)
	if mode := manifest.RuntimeModeOrDefault(opts.RuntimeMode); mode == "host" {
		m.RuntimeMode = "host"
		m.RuntimeContractVersion = 1
	}
	template := strings.TrimSpace(strings.ToLower(opts.Template))
	if template == "" {
		template = "basic"
	}
	switch template {
	case "basic":
	case "widget":
		m.UIType = "widget"
	default:
		return "", fmt.Errorf("unsupported template %q (allowed: basic, widget)", template)
	}
	if strings.TrimSpace(opts.Category) != "" {
		m.Category = opts.Category
	}
	if strings.TrimSpace(opts.Tier) != "" {
		m.Tier = opts.Tier
	}
	if strings.TrimSpace(opts.Description) != "" {
		m.Description = opts.Description
	}
	m.Tags = opts.Tags

	mr := manifest.Validate(m)
	if mr.Errors > 0 {
		return "", fmt.Errorf("manifest defaults are invalid: %d errors", mr.Errors)
	}

	dirs := []string{
		"tool/locales",
		"tool/assets/screenshots",
		"tests",
		"dist",
		".utilivos",
	}
	if manifest.RuntimeModeOrDefault(m.RuntimeMode) == "bundled" {
		dirs = append(dirs, "tool/utilivos")
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(absTarget, dir), 0o755); err != nil {
			return "", err
		}
	}

	if err := manifest.Save(filepath.Join(absTarget, "manifest.json"), m); err != nil {
		return "", err
	}
	files := map[string]string{
		"AGENTS.md":                 templates.AgentsMarkdown(opts.Name, opts.Slug),
		"build.ignore":              templates.BuildIgnore(),
		"README.md":                 templates.ReadmeMarkdown(opts.Name, opts.Slug),
		"PLAN.md":                   templates.PlanMarkdown(opts.Name, opts.Slug),
		".env.example":              templates.EnvExample(),
		"tool/index.html":           templates.IndexHTML(opts.Name, manifest.RuntimeModeOrDefault(m.RuntimeMode)),
		"tool/main.js":              templates.MainJS(),
		"tool/style.css":            templates.StylesCSS(),
		"tool/locales/en.json":      templates.LocaleENJSON(opts.Name),
		"tests/manifest.test.json":  templates.ManifestTestJSON(),
		"tests/structure.test.yaml": templates.StructureTestYAML(manifest.RuntimeModeOrDefault(m.RuntimeMode)),
		".utilivos/project.json":     templates.ProjectJSON(opts.Slug),
		".gitignore":                defaultGitignore(),
	}
	if manifest.RuntimeModeOrDefault(m.RuntimeMode) == "bundled" {
		files["tool/utilivos/base.css"] = templates.BaseCSS()
		files["tool/utilivos/components.css"] = templates.ComponentsCSS()
		files["tool/utilivos/layout.css"] = templates.LayoutCSS()
		files["tool/utilivos/utilities.css"] = templates.UtilitiesCSS()
		files["tool/utilivos/patterns.css"] = templates.PatternsCSS()
		files["tool/utilivos/tool.js"] = templates.ToolJS()
	}
	for rel, content := range files {
		path := filepath.Join(absTarget, rel)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return "", err
		}
	}

	if err := os.WriteFile(filepath.Join(absTarget, "tool/assets/icon.png"), templates.IconPNG(), 0o644); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(absTarget, "tool/assets/thumb.png"), templates.ThumbPNG(), 0o644); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(absTarget, "tool/assets/screenshots/home.png"), templates.ScreenshotPNG(), 0o644); err != nil {
		return "", err
	}

	return absTarget, nil
}

func humanize(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "My Plugin"
	}
	v = strings.ReplaceAll(v, "-", " ")
	v = strings.ReplaceAll(v, "_", " ")
	parts := strings.Fields(v)
	for i := range parts {
		parts[i] = strings.ToUpper(parts[i][:1]) + strings.ToLower(parts[i][1:])
	}
	return strings.Join(parts, " ")
}

func defaultGitignore() string {
	return `dist/*.ubb
dist/*.utilivos.zip
dist/*.sig
dist/*.sha256
dist/*.manifest.json
.utilivos/tmp/
`
}
