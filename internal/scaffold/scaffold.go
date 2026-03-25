package scaffold

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/utilivo/utilivo-sdk/internal/manifest"
	"github.com/utilivo/utilivo-sdk/internal/templates"
	"github.com/utilivo/utilivo-sdk/internal/util"
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
		opts.Author = "Utilivo Developer"
	}
	if len(opts.Tags) == 0 {
		opts.Tags = []string{"starter", "tool", "utilivo"}
	}
	sort.Strings(opts.Tags)

	m := manifest.Default(opts.Name, opts.Slug, opts.Author)
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
		"tool/utilivo",
		"tool/locales",
		"tool/assets/screenshots",
		"tests",
		"dist",
		".utilivo",
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
		"AGENTS.md":                   templates.AgentsMarkdown(opts.Name, opts.Slug),
		"build.ignore":                templates.BuildIgnore(),
		"README.md":                   templates.ReadmeMarkdown(opts.Name, opts.Slug),
		"PLAN.md":                     templates.PlanMarkdown(opts.Name, opts.Slug),
		".env.example":                templates.EnvExample(),
		"tool/index.html":             templates.IndexHTML(opts.Name),
		"tool/main.js":                templates.MainJS(),
		"tool/style.css":              templates.StylesCSS(),
		"tool/locales/en.json":        templates.LocaleENJSON(opts.Name),
		"tool/utilivo/base.css":       templates.BaseCSS(),
		"tool/utilivo/components.css": templates.ComponentsCSS(),
		"tool/utilivo/layout.css":     templates.LayoutCSS(),
		"tool/utilivo/utilities.css":  templates.UtilitiesCSS(),
		"tool/utilivo/patterns.css":   templates.PatternsCSS(),
		"tool/utilivo/tool.js":        templates.ToolJS(),
		"tests/manifest.test.json":    templates.ManifestTestJSON(),
		"tests/structure.test.yaml":   templates.StructureTestYAML(),
		".utilivo/project.json":       templates.ProjectJSON(opts.Slug),
		".gitignore":                  defaultGitignore(),
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
dist/*.utilivo.zip
dist/*.sig
dist/*.sha256
dist/*.manifest.json
.utilivo/tmp/
`
}
