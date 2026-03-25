package manifest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/utilivo/utilivo-sdk/internal/util"
)

var (
	slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)   // nolint:gochecknoglobals
	idPattern   = regexp.MustCompile(`^[a-z0-9]+(?:\.[a-z0-9-]+)+$`) // nolint:gochecknoglobals
	tagPattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)    // nolint:gochecknoglobals
	apiPattern  = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?$`)        // nolint:gochecknoglobals
	localeRegex = regexp.MustCompile(`^[a-z]{2}(?:-[A-Z]{2})?$`)     // nolint:gochecknoglobals
	pathPattern = regexp.MustCompile(`^[^\x00]+$`)                   // nolint:gochecknoglobals
	categories  = []string{"productivity", "utilities", "developer-tools", "writing", "design", "automation", "finance", "health", "education", "media", "shopping", "communication", "lifestyle", "business", "travel", "custom"}
	permissions = []string{"storage", "clipboard.read", "clipboard.write", "network", "notifications", "camera", "microphone", "share", "filesystem.read", "filesystem.write"}
	uiTypes     = []string{"page", "widget", "modal", "panel", "background"}
	tiers       = []string{"free", "premium"}
)

type ContentSecurityPolicy struct {
	ScriptSrc  []string `json:"script_src,omitempty"`
	ConnectSrc []string `json:"connect_src,omitempty"`
	ImgSrc     []string `json:"img_src,omitempty"`
}

type Manifest struct {
	ID                    string                 `json:"id"`
	Name                  string                 `json:"name"`
	Slug                  string                 `json:"slug"`
	Version               string                 `json:"version"`
	Description           string                 `json:"description"`
	Author                string                 `json:"author"`
	Entry                 string                 `json:"entry"`
	Icon                  string                 `json:"icon"`
	Thumb                 string                 `json:"thumb"`
	Category              string                 `json:"category"`
	Tags                  []string               `json:"tags"`
	Tier                  string                 `json:"tier"`
	Permissions           []string               `json:"permissions"`
	UIType                string                 `json:"ui_type"`
	APIVersion            string                 `json:"api_version"`
	Screenshots           []string               `json:"screenshots"`
	MinAppVersion         string                 `json:"min_app_version"`
	Homepage              string                 `json:"homepage,omitempty"`
	Repository            string                 `json:"repository,omitempty"`
	License               string                 `json:"license,omitempty"`
	Keywords              []string               `json:"keywords,omitempty"`
	Locales               []string               `json:"locales,omitempty"`
	DefaultLocale         string                 `json:"default_locale,omitempty"`
	ContentSecurityPolicy *ContentSecurityPolicy `json:"content_security_policy,omitempty"`
	ManifestSchemaVersion int                    `json:"manifest_schema_version,omitempty"`
	DesignSystemVersion   int                    `json:"design_system_version"`
	SupportsDarkMode      bool                   `json:"supports_dark_mode"`
}

func AllowedCategories() []string {
	out := make([]string, len(categories))
	copy(out, categories)
	return out
}

func AllowedPermissions() []string {
	out := make([]string, len(permissions))
	copy(out, permissions)
	return out
}

func AllowedUITypes() []string {
	out := make([]string, len(uiTypes))
	copy(out, uiTypes)
	return out
}

func Default(pluginName, slug, author string) Manifest {
	tags := []string{"starter", "tool", "utilivo"}
	sort.Strings(tags)
	return Manifest{
		ID:                    fmt.Sprintf("com.utilivo.plugin.%s", slug),
		Name:                  pluginName,
		Slug:                  slug,
		Version:               "1.0.0",
		Description:           fmt.Sprintf("%s plugin for Utilivo.", pluginName),
		Author:                author,
		Entry:                 "tool/index.html",
		Icon:                  "tool/assets/icon.png",
		Thumb:                 "tool/assets/thumb.png",
		Category:              "utilities",
		Tags:                  tags,
		Tier:                  "free",
		Permissions:           []string{"storage"},
		UIType:                "page",
		APIVersion:            "1",
		Screenshots:           []string{"tool/assets/screenshots/home.png"},
		MinAppVersion:         "1.0.0",
		Locales:               []string{"en"},
		DefaultLocale:         "en",
		ManifestSchemaVersion: 1,
		DesignSystemVersion:   1,
		SupportsDarkMode:      true,
	}
}

func Load(path string) (Manifest, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	return ParseJSON(b)
}

func ParseJSON(b []byte) (Manifest, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var m Manifest
	if err := dec.Decode(&m); err != nil {
		return Manifest{}, fmt.Errorf("invalid manifest json: %w", err)
	}
	return m, nil
}

func Save(path string, m Manifest) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}

func Validate(m Manifest) util.Report {
	r := util.NewReport()

	required := map[string]string{
		"id":              m.ID,
		"name":            m.Name,
		"slug":            m.Slug,
		"version":         m.Version,
		"description":     m.Description,
		"author":          m.Author,
		"entry":           m.Entry,
		"icon":            m.Icon,
		"thumb":           m.Thumb,
		"category":        m.Category,
		"tier":            m.Tier,
		"ui_type":         m.UIType,
		"api_version":     m.APIVersion,
		"min_app_version": m.MinAppVersion,
	}
	for key, value := range required {
		if strings.TrimSpace(value) == "" {
			r.Add(util.SeverityError, "manifest.required", "manifest.json", fmt.Sprintf("field %q is required", key))
		}
		if util.ContainsPlaceholder(value) {
			r.Add(util.SeverityError, "manifest.placeholder", "manifest.json", fmt.Sprintf("field %q contains unresolved placeholder", key))
		}
	}

	if !idPattern.MatchString(m.ID) {
		r.Add(util.SeverityError, "manifest.id", "manifest.json", "id must look like com.utilivo.plugin.my-plugin")
	}
	if !IsValidSlug(m.Slug) {
		r.Add(util.SeverityError, "manifest.slug", "manifest.json", "slug must match ^[a-z0-9]+(?:-[a-z0-9]+)*$")
	}
	if !util.IsSemver(m.Version) {
		r.Add(util.SeverityError, "manifest.version", "manifest.json", "version must be semver (for example 1.0.0)")
	}
	if !util.IsSemver(m.MinAppVersion) {
		r.Add(util.SeverityError, "manifest.min_app_version", "manifest.json", "min_app_version must be semver")
	}
	if !isOneOf(m.Category, categories) {
		r.Add(util.SeverityError, "manifest.category", "manifest.json", "category is not allowed")
	}
	if !isOneOf(m.Tier, tiers) {
		r.Add(util.SeverityError, "manifest.tier", "manifest.json", "tier must be free or premium")
	}
	if !isOneOf(m.UIType, uiTypes) {
		r.Add(util.SeverityError, "manifest.ui_type", "manifest.json", fmt.Sprintf("ui_type must be one of %s", strings.Join(uiTypes, ", ")))
	}
	if !apiPattern.MatchString(m.APIVersion) {
		r.Add(util.SeverityError, "manifest.api_version", "manifest.json", "api_version must be numeric")
	}
	if m.DesignSystemVersion == 0 {
		r.Add(util.SeverityError, "manifest.design_system_version", "manifest.json", "design_system_version is required and must be 1")
	} else if m.DesignSystemVersion != 1 {
		r.Add(util.SeverityError, "manifest.design_system_version", "manifest.json", "design_system_version must be 1")
	}
	if !m.SupportsDarkMode {
		r.Add(util.SeverityError, "manifest.supports_dark_mode", "manifest.json", "supports_dark_mode must be true")
	}
	if len(m.Tags) == 0 {
		r.Add(util.SeverityError, "manifest.tags", "manifest.json", "tags must be non-empty")
	} else if len(m.Tags) < 3 || len(m.Tags) > 8 {
		r.Add(util.SeverityWarning, "manifest.tags.recommended_count", "manifest.json", "tags should usually contain 3 to 8 values")
	}
	tagSet := make(map[string]bool, len(m.Tags))
	for i, tag := range m.Tags {
		if !tagPattern.MatchString(tag) {
			r.Add(util.SeverityError, "manifest.tag", "manifest.json", fmt.Sprintf("tag at index %d is invalid", i))
		}
		if util.ContainsPlaceholder(tag) {
			r.Add(util.SeverityError, "manifest.placeholder", "manifest.json", fmt.Sprintf("tag at index %d contains unresolved placeholder", i))
		}
		if tagSet[tag] {
			r.Add(util.SeverityError, "manifest.tag.duplicate", "manifest.json", fmt.Sprintf("duplicate tag %q", tag))
		}
		tagSet[tag] = true
	}
	if len(m.Screenshots) == 0 {
		r.Add(util.SeverityError, "manifest.screenshots", "manifest.json", "screenshots must contain at least one path")
	}
	if len(m.Permissions) == 0 {
		r.Add(util.SeverityError, "manifest.permissions", "manifest.json", "permissions must contain at least one permission")
	}
	if len(m.Locales) == 0 {
		r.Add(util.SeverityError, "manifest.locales", "manifest.json", "locales must contain at least one locale and include en")
	}
	if m.ManifestSchemaVersion == 0 {
		r.Add(util.SeverityWarning, "manifest.schema_version", "manifest.json", "manifest_schema_version is recommended and should be 1")
	}

	permissionSet := toSet(permissions)
	for _, permission := range m.Permissions {
		if !permissionSet[permission] {
			r.Add(util.SeverityError, "manifest.permission", "manifest.json", fmt.Sprintf("permission %q is not allowed", permission))
		}
	}
	localeSet := make(map[string]bool, len(m.Locales))
	for i, locale := range m.Locales {
		if !localeRegex.MatchString(locale) {
			r.Add(util.SeverityError, "manifest.locale", "manifest.json", fmt.Sprintf("locale at index %d is invalid (expected en or en-US format)", i))
		}
		if localeSet[locale] {
			r.Add(util.SeverityError, "manifest.locale.duplicate", "manifest.json", fmt.Sprintf("duplicate locale %q", locale))
		}
		localeSet[locale] = true
	}
	if len(m.Locales) > 0 && !localeSet["en"] {
		r.Add(util.SeverityError, "manifest.locales", "manifest.json", "locales must include en")
	}
	if strings.TrimSpace(m.DefaultLocale) == "" {
		r.Add(util.SeverityError, "manifest.default_locale", "manifest.json", "default_locale is required and must be en")
	} else {
		if len(m.Locales) == 0 {
			r.Add(util.SeverityError, "manifest.default_locale", "manifest.json", "default_locale requires locales to be set")
		} else if !localeSet[m.DefaultLocale] {
			r.Add(util.SeverityError, "manifest.default_locale", "manifest.json", "default_locale must be present in locales")
		}
		if m.DefaultLocale != "en" {
			r.Add(util.SeverityError, "manifest.default_locale", "manifest.json", "default_locale must be en")
		}
	}

	pathFields := []string{m.Entry, m.Icon, m.Thumb}
	pathFields = append(pathFields, m.Screenshots...)
	for _, p := range pathFields {
		if err := ValidateRelativePath(p); err != nil {
			r.Add(util.SeverityError, "manifest.path", "manifest.json", err.Error())
		}
	}

	return r
}

func ValidateRelativePath(pathValue string) error {
	if strings.TrimSpace(pathValue) == "" {
		return errors.New("path values cannot be empty")
	}
	if !pathPattern.MatchString(pathValue) {
		return fmt.Errorf("path %q contains invalid bytes", pathValue)
	}
	if filepath.IsAbs(pathValue) {
		return fmt.Errorf("path %q must be relative", pathValue)
	}
	if strings.Contains(pathValue, "\\") {
		return fmt.Errorf("path %q must use forward slashes", pathValue)
	}
	clean := filepath.ToSlash(filepath.Clean(pathValue))
	if strings.HasPrefix(clean, "../") || clean == ".." {
		return fmt.Errorf("path %q cannot escape project root", pathValue)
	}
	if clean != pathValue {
		return fmt.Errorf("path %q must be normalized (expected %q)", pathValue, clean)
	}
	return nil
}

func BumpVersion(version, level string) (string, error) {
	if !util.IsSemver(version) {
		return "", errors.New("version is not semver")
	}
	parts := strings.SplitN(version, "-", 2)
	core := strings.Split(parts[0], ".")
	if len(core) != 3 {
		return "", errors.New("version is not semver")
	}
	var major, minor, patch int
	_, err := fmt.Sscanf(parts[0], "%d.%d.%d", &major, &minor, &patch)
	if err != nil {
		return "", err
	}
	switch level {
	case "major":
		major++
		minor = 0
		patch = 0
	case "minor":
		minor++
		patch = 0
	case "patch":
		patch++
	default:
		return "", errors.New("bump level must be patch, minor, or major")
	}
	return fmt.Sprintf("%d.%d.%d", major, minor, patch), nil
}

func IsValidSlug(slug string) bool {
	return slugPattern.MatchString(slug)
}

func isOneOf(value string, allowed []string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

func toSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}
