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
	"strconv"
	"strings"

	"utilivos/sdk/internal/util"
)

var (
	slugPattern  = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)   // nolint:gochecknoglobals
	idPattern    = regexp.MustCompile(`^[a-z0-9]+(?:\.[a-z0-9-]+)+$`) // nolint:gochecknoglobals
	tagPattern   = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)    // nolint:gochecknoglobals
	apiPattern   = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?$`)        // nolint:gochecknoglobals
	localeRegex  = regexp.MustCompile(`^[a-z]{2}(?:-[A-Z]{2})?$`)     // nolint:gochecknoglobals
	pathPattern  = regexp.MustCompile(`^[^\x00]+$`)                   // nolint:gochecknoglobals
	categories   = []string{"productivity", "utilities", "developer-tools", "writing", "design", "automation", "finance", "health", "education", "media", "shopping", "communication", "lifestyle", "business", "travel", "custom"}
	permissions  = []string{"storage", "clipboard.read", "clipboard.write", "network", "notifications", "camera", "microphone", "share", "filesystem.read", "filesystem.write"}
	uiTypes      = []string{"page", "widget", "modal", "panel", "background"}
	tiers        = []string{"free", "premium"}
	runtimeModes = []string{"bundled", "host"}
)

// CurrentRuntimeContractVersion is the contract revision the SDK and host use for runtime_mode=host.
const CurrentRuntimeContractVersion = 1

// flexInt accepts JSON numbers or quoted numeric strings (e.g. "1") for hand-edited manifests.
type flexInt int

func (f *flexInt) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		*f = 0
		return nil
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		s = strings.TrimSpace(s)
		if s == "" {
			*f = 0
			return nil
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("expected integer, got %q", s)
		}
		*f = flexInt(n)
		return nil
	}
	var n int
	if err := json.Unmarshal(data, &n); err != nil {
		return err
	}
	*f = flexInt(n)
	return nil
}

func (f flexInt) MarshalJSON() ([]byte, error) {
	return json.Marshal(int(f))
}

type ContentSecurityPolicy struct {
	ScriptSrc  []string `json:"script_src,omitempty"`
	ConnectSrc []string `json:"connect_src,omitempty"`
	ImgSrc     []string `json:"img_src,omitempty"`
}

type Manifest struct {
	ID                     string                 `json:"id"`
	Name                   string                 `json:"name"`
	Slug                   string                 `json:"slug"`
	Version                string                 `json:"version"`
	Description            string                 `json:"description"`
	Author                 string                 `json:"author"`
	Entry                  string                 `json:"entry"`
	Icon                   string                 `json:"icon"`
	Thumb                  string                 `json:"thumb"`
	Category               string                 `json:"category"`
	Tags                   []string               `json:"tags"`
	Tier                   string                 `json:"tier"`
	Permissions            []string               `json:"permissions"`
	UIType                 string                 `json:"ui_type"`
	APIVersion             string                 `json:"api_version"`
	Screenshots            []string               `json:"screenshots"`
	MinAppVersion          string                 `json:"min_app_version"`
	Homepage               string                 `json:"homepage,omitempty"`
	Repository             string                 `json:"repository,omitempty"`
	License                string                 `json:"license,omitempty"`
	Keywords               []string               `json:"keywords,omitempty"`
	Locales                []string               `json:"locales,omitempty"`
	DefaultLocale          string                 `json:"default_locale,omitempty"`
	ContentSecurityPolicy  *ContentSecurityPolicy `json:"content_security_policy,omitempty"`
	ManifestSchemaVersion  flexInt                `json:"manifest_schema_version,omitempty"`
	DesignSystemVersion    flexInt                `json:"design_system_version"`
	SupportsDarkMode       bool                   `json:"supports_dark_mode"`
	RuntimeMode            string                 `json:"runtime_mode,omitempty"`
	RuntimeContractVersion flexInt                `json:"runtime_contract_version,omitempty"`
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
	tags := []string{"starter", "tool", "utilivos"}
	sort.Strings(tags)
	return Manifest{
		ID:                    fmt.Sprintf("com.utilivos.plugin.%s", slug),
		Name:                  pluginName,
		Slug:                  slug,
		Version:               "1.0.0",
		Description:           fmt.Sprintf("%s plugin for Utilivos.", pluginName),
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
		RuntimeMode:           "bundled",
	}
}

func RuntimeModeOrDefault(mode string) string {
	if strings.TrimSpace(mode) == "" {
		return "bundled"
	}
	return strings.ToLower(strings.TrimSpace(mode))
}

// MergeLatestRequirements returns a copy of m with fields required by the current manifest
// contract filled from defaults when they are missing or invalid for validation (for example
// empty slices, zero schema versions, or runtime_mode host without a contract version).
// It does not overwrite non-empty user values. The second return lists JSON field names
// that were added or changed.
func MergeLatestRequirements(m Manifest) (Manifest, []string) {
	var added []string
	note := func(field string) {
		added = append(added, field)
	}
	out := m

	plugName := strings.TrimSpace(out.Name)
	if plugName == "" {
		plugName = "Plugin"
	}
	slugForDef := strings.TrimSpace(out.Slug)
	if slugForDef == "" {
		slugForDef = "plugin"
	}
	auth := strings.TrimSpace(out.Author)
	if auth == "" {
		auth = "Unknown"
	}
	def := Default(plugName, slugForDef, auth)

	if strings.TrimSpace(out.ID) == "" {
		slugTrim := strings.TrimSpace(out.Slug)
		if slugTrim != "" && IsValidSlug(slugTrim) {
			out.ID = fmt.Sprintf("com.utilivos.plugin.%s", slugTrim)
			note("id")
		}
	}
	if strings.TrimSpace(out.Name) == "" {
		out.Name = def.Name
		note("name")
	}
	// Do not invent slug when missing; authors must set it explicitly.
	if strings.TrimSpace(out.Description) == "" {
		out.Description = def.Description
		note("description")
	}
	if strings.TrimSpace(out.Author) == "" {
		out.Author = def.Author
		note("author")
	}
	if strings.TrimSpace(out.Version) == "" {
		out.Version = def.Version
		note("version")
	}
	if strings.TrimSpace(out.Entry) == "" {
		out.Entry = def.Entry
		note("entry")
	}
	if strings.TrimSpace(out.Icon) == "" {
		out.Icon = def.Icon
		note("icon")
	}
	if strings.TrimSpace(out.Thumb) == "" {
		out.Thumb = def.Thumb
		note("thumb")
	}
	if strings.TrimSpace(out.Category) == "" {
		out.Category = def.Category
		note("category")
	}
	if strings.TrimSpace(out.Tier) == "" {
		out.Tier = def.Tier
		note("tier")
	}
	if strings.TrimSpace(out.UIType) == "" {
		out.UIType = def.UIType
		note("ui_type")
	}
	if strings.TrimSpace(out.APIVersion) == "" {
		out.APIVersion = def.APIVersion
		note("api_version")
	}
	if strings.TrimSpace(out.MinAppVersion) == "" {
		out.MinAppVersion = def.MinAppVersion
		note("min_app_version")
	}

	if out.ManifestSchemaVersion == 0 {
		out.ManifestSchemaVersion = 1
		note("manifest_schema_version")
	}
	if out.DesignSystemVersion == 0 {
		out.DesignSystemVersion = 1
		note("design_system_version")
	}
	if !out.SupportsDarkMode {
		out.SupportsDarkMode = true
		note("supports_dark_mode")
	}
	if strings.TrimSpace(out.RuntimeMode) == "" {
		out.RuntimeMode = "bundled"
		note("runtime_mode")
	}
	mode := RuntimeModeOrDefault(out.RuntimeMode)
	if mode == "host" && out.RuntimeContractVersion <= 0 {
		out.RuntimeContractVersion = CurrentRuntimeContractVersion
		note("runtime_contract_version")
	}

	if len(out.Tags) == 0 {
		out.Tags = append([]string(nil), def.Tags...)
		note("tags")
	}
	if len(out.Permissions) == 0 {
		out.Permissions = append([]string(nil), def.Permissions...)
		note("permissions")
	}
	if len(out.Screenshots) == 0 {
		out.Screenshots = append([]string(nil), def.Screenshots...)
		note("screenshots")
	}
	if len(out.Locales) == 0 {
		out.Locales = append([]string(nil), def.Locales...)
		note("locales")
	}
	if strings.TrimSpace(out.DefaultLocale) == "" {
		out.DefaultLocale = "en"
		note("default_locale")
	}

	return out, added
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
		r.Add(util.SeverityError, "manifest.id", "manifest.json", "id must look like com.utilivos.plugin.my-plugin")
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
	mode := RuntimeModeOrDefault(m.RuntimeMode)
	if !isOneOf(mode, runtimeModes) {
		r.Add(util.SeverityError, "manifest.runtime_mode", "manifest.json", "runtime_mode must be one of bundled, host")
	}
	if mode == "host" && m.RuntimeContractVersion <= 0 {
		r.Add(util.SeverityError, "manifest.runtime_contract_version", "manifest.json", "runtime_contract_version is required for runtime_mode=host and must be > 0")
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
