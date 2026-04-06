package verify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"utilivos/sdk/internal/archive"
	"utilivos/sdk/internal/manifest"
	"utilivos/sdk/internal/util"
)

var (
	remoteScriptPattern       = regexp.MustCompile(`(?i)<script[^>]+src=["']https?://`)                               // nolint:gochecknoglobals
	absolutePathPattern       = regexp.MustCompile(`(?i)(/home/[A-Za-z0-9._-]+/|/Users/[A-Za-z0-9._-]+/|[A-Z]:\\\\)`) // nolint:gochecknoglobals
	baseCSSImportPattern      = regexp.MustCompile(`(?i)<link[^>]+href=["'][^"']*utilivos/base\.css["']`)              // nolint:gochecknoglobals
	layoutCSSImportPattern    = regexp.MustCompile(`(?i)<link[^>]+href=["'][^"']*utilivos/layout\.css["']`)            // nolint:gochecknoglobals
	componentsImportPattern   = regexp.MustCompile(`(?i)<link[^>]+href=["'][^"']*utilivos/components\.css["']`)        // nolint:gochecknoglobals
	toolJSImportPattern       = regexp.MustCompile(`(?i)<script[^>]+src=["'][^"']*utilivos/tool\.js["']`)              // nolint:gochecknoglobals
	toolPageContainerPattern  = regexp.MustCompile(`(?i)class=["'][^"']*\buv-tool-page\b[^"']*["']`)                  // nolint:gochecknoglobals
	localizationMarkerPattern = regexp.MustCompile(`(?i)data-i18n(?:-placeholder|-title|-aria-label|-document-title)?=`)
	visibleHTMLTextPattern    = regexp.MustCompile(`>\s*[^<\s][^<]*[A-Za-z][^<]*<`)
	htmlCommentPattern        = regexp.MustCompile(`(?s)<!--.*?-->`)
	localizationInitPattern   = regexp.MustCompile(`\binitLocalization\s*\(`)
	tokenOverridePattern      = regexp.MustCompile(`--uv-[a-z0-9-]+\s*:`) // nolint:gochecknoglobals
	sessionBridgePattern      = regexp.MustCompile(`UtilivosRuntime\.syncState\s*\(|utilivos\.persistState\s*\(|\.persistState\s*\(`) // nolint:gochecknoglobals
)

var localizationStringPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?:textContent|innerText|innerHTML)\s*=\s*["'` + "`" + `][^"'` + "`" + `]*[A-Za-z][^"'` + "`" + `]*["'` + "`" + `]`),
	regexp.MustCompile(`setAttribute\(\s*["'](?:placeholder|title|aria-label)["']\s*,\s*["'` + "`" + `][^"'` + "`" + `]*[A-Za-z][^"'` + "`" + `]*["'` + "`" + `]`),
	regexp.MustCompile(`\b(?:alert|confirm|prompt)\(\s*["'` + "`" + `][^"'` + "`" + `]*[A-Za-z][^"'` + "`" + `]*["'` + "`" + `]`),
	regexp.MustCompile(`\b(?:text|label|placeholder)\s*:\s*["'` + "`" + `][^"'` + "`" + `]*[A-Za-z][^"'` + "`" + `]*["'` + "`" + `]`),
}

type Result struct {
	Root     string             `json:"root"`
	Manifest *manifest.Manifest `json:"manifest,omitempty"`
	Report   util.Report        `json:"report"`
}

func Run(root string) (Result, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return Result{}, err
	}
	r := util.NewReport()

	requiredSource := []string{
		"manifest.json",
		"AGENTS.md",
		"README.md",
		"tool/index.html",
		"tool/main.js",
		"tool/style.css",
		"tool/locales/en.json",
		"tool/assets/icon.png",
		"tool/assets/thumb.png",
		"tool/assets/screenshots/home.png",
		"tests",
		".utilivos/project.json",
	}
	for _, rel := range requiredSource {
		p := filepath.Join(absRoot, rel)
		if !util.Exists(p) {
			r.Add(util.SeverityError, "structure.required_source", rel, "required source/config path is missing")
		}
	}

	generatedOutput := []string{"dist"}
	for _, rel := range generatedOutput {
		p := filepath.Join(absRoot, rel)
		if util.Exists(p) && !util.IsDir(p) {
			r.Add(util.SeverityError, "structure.generated_type", rel, "generated output path exists but is not a directory")
		}
	}
	if !util.Exists(filepath.Join(absRoot, "PLAN.md")) {
		r.Add(util.SeverityWarning, "structure.recommended", "PLAN.md", "PLAN.md is recommended by the project guidelines")
	}
	if !util.Exists(filepath.Join(absRoot, ".env.example")) {
		r.Add(util.SeverityWarning, "structure.recommended", ".env.example", ".env.example is recommended by the project guidelines")
	}

	manifestPath := filepath.Join(absRoot, "manifest.json")
	var parsed manifest.Manifest
	runtimeMode := "bundled"
	if util.Exists(manifestPath) {
		parsed, err = manifest.Load(manifestPath)
		if err != nil {
			r.Add(util.SeverityError, "manifest.parse", "manifest.json", err.Error())
		} else {
			runtimeMode = manifest.RuntimeModeOrDefault(parsed.RuntimeMode)
			r.Merge(manifest.Validate(parsed))
			r.Merge(validateManifestReferences(absRoot, parsed))
			r.Merge(validateLocaleFiles(absRoot, parsed))
		}
	}
	if runtimeMode == "bundled" {
		requiredSource = append(requiredSource,
			"tool/utilivos/base.css",
			"tool/utilivos/components.css",
			"tool/utilivos/layout.css",
			"tool/utilivos/tool.js",
		)
	}

	if util.IsDir(filepath.Join(absRoot, "tests")) {
		entries, err := os.ReadDir(filepath.Join(absRoot, "tests"))
		if err == nil && len(entries) == 0 {
			r.Add(util.SeverityWarning, "tests.empty", "tests", "tests directory is empty")
		}
	}

	entryRelPath := "tool/index.html"
	if strings.TrimSpace(parsed.Entry) != "" {
		entryRelPath = filepath.ToSlash(parsed.Entry)
	}
	entryPath := filepath.Join(absRoot, filepath.FromSlash(entryRelPath))
	if util.Exists(entryPath) {
		content, err := os.ReadFile(entryPath)
		if err == nil {
			if remoteScriptPattern.Match(content) {
				r.Add(util.SeverityError, "entry.remote_script", entryRelPath, "remote script tags are not allowed")
			}
			r.Merge(validateEntryDesignSystem(entryRelPath, content, runtimeMode))
			r.Merge(validateEntryLocalization(entryRelPath, content))
		}
	} else {
		r.Add(util.SeverityError, "entry.missing", entryRelPath, "entry html file is missing")
	}

	if list, err := archive.CollectPackageFiles(absRoot); err != nil {
		r.Add(util.SeverityError, "archive.collect", "", err.Error())
	} else {
		var total int64
		packaged := make(map[string]struct{}, len(list))
		for _, e := range list {
			total += e.Size
			packaged[e.RelPath] = struct{}{}
		}
		if total > 20*1024*1024 {
			r.Add(util.SeverityWarning, "archive.size", "", "package payload exceeds 20MB")
		}
		for _, rel := range requiredPackagedFiles(runtimeMode) {
			if _, ok := packaged[rel]; !ok {
				r.Add(util.SeverityError, "archive.required_package", rel, "required file is excluded from the final package payload")
			}
		}
	}

	r.Merge(scanProjectFiles(absRoot))
	r.Merge(scanTokenOverrides(absRoot))
	r.Merge(scanLocalizationUsage(absRoot))
	r.Merge(scanSessionPersistence(absRoot))

	result := Result{
		Root:   absRoot,
		Report: r,
	}
	if parsed.Slug != "" {
		result.Manifest = &parsed
	}
	return result, nil
}

func requiredPackagedFiles(runtimeMode string) []string {
	required := []string{
		"manifest.json",
		"tool/index.html",
		"tool/main.js",
		"tool/style.css",
		"tool/locales/en.json",
		"tool/assets/icon.png",
		"tool/assets/thumb.png",
		"tool/assets/screenshots/home.png",
	}
	if runtimeMode == "bundled" {
		required = append(required,
			"tool/utilivos/base.css",
			"tool/utilivos/components.css",
			"tool/utilivos/layout.css",
			"tool/utilivos/tool.js",
		)
	}
	return required
}

func validateManifestReferences(root string, m manifest.Manifest) util.Report {
	r := util.NewReport()
	paths := []struct {
		field string
		path  string
	}{
		{field: "entry", path: m.Entry},
		{field: "icon", path: m.Icon},
		{field: "thumb", path: m.Thumb},
	}
	for _, shot := range m.Screenshots {
		paths = append(paths, struct {
			field string
			path  string
		}{field: "screenshots", path: shot})
	}
	for _, item := range paths {
		if strings.TrimSpace(item.path) == "" {
			continue
		}
		p := filepath.Join(root, filepath.FromSlash(item.path))
		if !util.Exists(p) {
			r.Add(util.SeverityError, "manifest.ref_missing", item.path, fmt.Sprintf("%s path does not exist", item.field))
		}
	}
	return r
}

func scanProjectFiles(root string) util.Report {
	r := util.NewReport()
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		if d.IsDir() {
			if rel == ".git" || rel == "node_modules" {
				r.Add(util.SeverityWarning, "forbidden.dir", rel, "directory should not be present in plugin project")
			}
			return nil
		}

		if archive.IsForbiddenArchivePath(rel) && strings.HasPrefix(rel, "src/") {
			r.Add(util.SeverityError, "forbidden.file", rel, "forbidden file in package scope")
		}

		if !isTextFile(rel) {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if !isLocaleResource(rel) && util.ContainsPlaceholder(string(b)) {
			r.Add(util.SeverityError, "placeholder.unresolved", rel, "unresolved placeholder found")
		}
		if absolutePathPattern.Match(bytes.ToLower(b)) {
			r.Add(util.SeverityError, "path.absolute_leak", rel, "absolute local filesystem path found")
		}
		return nil
	})
	if walkErr != nil {
		r.Add(util.SeverityError, "verify.walk", "", walkErr.Error())
	}
	return r
}

func validateEntryDesignSystem(entryPath string, content []byte, runtimeMode string) util.Report {
	r := util.NewReport()
	if runtimeMode == "bundled" {
		if !baseCSSImportPattern.Match(content) {
			r.Add(util.SeverityError, "design.import.base", entryPath, "entry html must import utilivos/base.css")
		}
		if !componentsImportPattern.Match(content) {
			r.Add(util.SeverityError, "design.import.components", entryPath, "entry html must import utilivos/components.css")
		}
		if !layoutCSSImportPattern.Match(content) {
			r.Add(util.SeverityError, "design.import.layout", entryPath, "entry html must import utilivos/layout.css")
		}
		if !toolJSImportPattern.Match(content) {
			r.Add(util.SeverityError, "design.import.tool_js", entryPath, "entry html must import utilivos/tool.js")
		}
	} else {
		if baseCSSImportPattern.Match(content) || componentsImportPattern.Match(content) || layoutCSSImportPattern.Match(content) || toolJSImportPattern.Match(content) {
			r.Add(util.SeverityError, "design.import.host_mode", entryPath, "runtime_mode=host must not import local utilivos runtime files")
		}
	}
	if !toolPageContainerPattern.Match(content) {
		r.Add(util.SeverityError, "design.tool_page", entryPath, "layout must use ToolPage container class uv-tool-page")
	}
	return r
}

func validateEntryLocalization(entryPath string, content []byte) util.Report {
	r := util.NewReport()
	if !localizationMarkerPattern.Match(content) {
		r.Add(util.SeverityError, "i18n.markers", entryPath, "entry html must use data-i18n markers for user-facing copy")
	}

	sanitized := htmlCommentPattern.ReplaceAll(content, []byte(""))
	if visibleHTMLTextPattern.Match(sanitized) {
		r.Add(util.SeverityError, "i18n.inline_html", entryPath, "entry html cannot contain visible hardcoded copy; use locale files and data-i18n attributes")
	}
	return r
}

func validateLocaleFiles(root string, m manifest.Manifest) util.Report {
	r := util.NewReport()
	if len(m.Locales) == 0 {
		return r
	}

	for _, locale := range m.Locales {
		localePath := filepath.Join(root, "tool", "locales", locale+".json")
		if !util.Exists(localePath) {
			r.Add(util.SeverityError, "i18n.locale_file", filepath.ToSlash(filepath.Join("tool", "locales", locale+".json")), "locale file is missing for manifest locale")
			continue
		}

		content, err := os.ReadFile(localePath)
		if err != nil {
			r.Add(util.SeverityError, "i18n.locale_file", filepath.ToSlash(filepath.Join("tool", "locales", locale+".json")), err.Error())
			continue
		}

		var parsed map[string]any
		if err := json.Unmarshal(content, &parsed); err != nil {
			r.Add(util.SeverityError, "i18n.locale_json", filepath.ToSlash(filepath.Join("tool", "locales", locale+".json")), "locale file must be a valid JSON object")
			continue
		}
		if len(parsed) == 0 {
			r.Add(util.SeverityError, "i18n.locale_empty", filepath.ToSlash(filepath.Join("tool", "locales", locale+".json")), "locale file cannot be empty")
		}
	}

	return r
}

func scanTokenOverrides(root string) util.Report {
	r := util.NewReport()
	toolRoot := filepath.Join(root, "tool")
	if !util.IsDir(toolRoot) {
		return r
	}

	walkErr := filepath.WalkDir(toolRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		if strings.HasPrefix(rel, "tool/utilivos/") {
			return nil
		}
		lower := strings.ToLower(rel)
		if !strings.HasSuffix(lower, ".css") && !strings.HasSuffix(lower, ".html") {
			return nil
		}

		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if tokenOverridePattern.Match(b) {
			r.Add(util.SeverityError, "design.override.tokens", rel, "plugin code cannot override core --uv-* design tokens")
		}
		return nil
	})
	if walkErr != nil {
		r.Add(util.SeverityError, "design.scan", "tool", walkErr.Error())
	}
	return r
}

func scanSessionPersistence(root string) util.Report {
	r := util.NewReport()
	toolRoot := filepath.Join(root, "tool")
	if !util.IsDir(toolRoot) {
		return r
	}

	found := false
	walkErr := filepath.WalkDir(toolRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "tool/utilivos/") || !strings.HasSuffix(strings.ToLower(rel), ".js") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if sessionBridgePattern.Match(content) {
			found = true
		}
		return nil
	})
	if walkErr != nil {
		r.Add(util.SeverityError, "session.scan", "tool", walkErr.Error())
		return r
	}
	if !found {
		r.Add(util.SeverityWarning, "session.persist_bridge", "tool",
			"no UtilivosRuntime.syncState or utilivos.persistState in plugin JavaScript under tool/ (excluding tool/utilivos); call UtilivosRuntime.syncState from tool/main.js so host Save session can read ephemeral state")
	}
	return r
}

func scanLocalizationUsage(root string) util.Report {
	r := util.NewReport()
	toolRoot := filepath.Join(root, "tool")
	if !util.IsDir(toolRoot) {
		return r
	}

	foundInit := false
	walkErr := filepath.WalkDir(toolRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "tool/utilivos/") || !strings.HasSuffix(strings.ToLower(rel), ".js") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if localizationInitPattern.Match(content) {
			foundInit = true
		}
		for _, pattern := range localizationStringPatterns {
			if pattern.Match(content) {
				r.Add(util.SeverityError, "i18n.inline_js", rel, "user-facing strings in JavaScript must come from locale files")
				break
			}
		}
		return nil
	})
	if walkErr != nil {
		r.Add(util.SeverityError, "i18n.scan", "tool", walkErr.Error())
	}
	if !foundInit {
		r.Add(util.SeverityError, "i18n.init", "tool/main.js", "plugin JavaScript must initialize localization via UtilivosUI.initLocalization")
	}
	return r
}

func isTextFile(rel string) bool {
	rel = strings.ToLower(rel)
	for _, ext := range []string{".json", ".md", ".txt", ".html", ".js", ".css", ".yaml", ".yml"} {
		if strings.HasSuffix(rel, ext) {
			return true
		}
	}
	return false
}

func isLocaleResource(rel string) bool {
	rel = filepath.ToSlash(strings.ToLower(rel))
	return strings.HasPrefix(rel, "tool/locales/") && strings.HasSuffix(rel, ".json")
}
