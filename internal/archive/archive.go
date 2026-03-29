package archive

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/utilivo/utilivo-sdk/internal/manifest"
	"github.com/utilivo/utilivo-sdk/internal/util"
)

var fixedZipTime = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC) // nolint:gochecknoglobals

type Entry struct {
	RelPath string
	AbsPath string
	Mode    fs.FileMode
	Size    int64
	Content []byte
}

type Inspection struct {
	ArchivePath  string             `json:"archive_path"`
	SHA256       string             `json:"sha256"`
	EntryCount   int                `json:"entry_count"`
	Entries      []string           `json:"entries"`
	Manifest     *manifest.Manifest `json:"manifest,omitempty"`
	HasSignature bool               `json:"has_signature"`
}

func CollectPackageFiles(root string) ([]Entry, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	ignored, err := loadBuildIgnore(root)
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, 32)
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
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
			if shouldSkipDir(rel) {
				return filepath.SkipDir
			}
			return nil
		}

		if !shouldInclude(rel) {
			return nil
		}
		if ignored.matches(rel) {
			return nil
		}
		if IsForbiddenArchivePath(rel) {
			return fmt.Errorf("forbidden file for package: %s", rel)
		}

		stat, err := os.Stat(path)
		if err != nil {
			return err
		}
		entries = append(entries, Entry{
			RelPath: rel,
			AbsPath: path,
			Mode:    stat.Mode(),
			Size:    stat.Size(),
		})
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].RelPath < entries[j].RelPath
	})
	if len(entries) == 0 {
		return nil, fmt.Errorf("no package files found in %s", root)
	}
	return entries, nil
}

// DropToolUtilivoForHostMode removes tool/utilivo/* from the package when runtime_mode is host.
// Plugins may keep that directory for styled local dev (utilivo run injects these assets); it must not ship in the .ubb.
func DropToolUtilivoForHostMode(entries []Entry, hostMode bool) []Entry {
	if !hostMode {
		return entries
	}
	out := make([]Entry, 0, len(entries))
	for _, e := range entries {
		if strings.HasPrefix(e.RelPath, "tool/utilivo/") {
			continue
		}
		out = append(out, e)
	}
	return out
}

type buildIgnore struct {
	exact       map[string]struct{}
	directories []string
}

func loadBuildIgnore(root string) (buildIgnore, error) {
	path := filepath.Join(root, "build.ignore")
	if !util.Exists(path) {
		return buildIgnore{exact: map[string]struct{}{}}, nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return buildIgnore{}, err
	}

	ignored := buildIgnore{
		exact: make(map[string]struct{}),
	}
	for lineNo, raw := range strings.Split(string(content), "\n") {
		line := normalizeBuildIgnoreLine(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if filepath.IsAbs(line) {
			return buildIgnore{}, fmt.Errorf("build.ignore line %d must use relative paths", lineNo+1)
		}
		if strings.HasPrefix(line, "../") || line == ".." {
			return buildIgnore{}, fmt.Errorf("build.ignore line %d cannot escape the project root", lineNo+1)
		}
		if strings.HasSuffix(line, "/") {
			ignored.directories = append(ignored.directories, strings.TrimSuffix(line, "/"))
			continue
		}
		ignored.exact[line] = struct{}{}
	}
	return ignored, nil
}

func normalizeBuildIgnoreLine(line string) string {
	line = strings.TrimSpace(filepath.ToSlash(line))
	line = strings.TrimPrefix(line, "./")
	return line
}

func (i buildIgnore) matches(rel string) bool {
	if _, ok := i.exact[rel]; ok {
		return true
	}
	for _, dir := range i.directories {
		if rel == dir || strings.HasPrefix(rel, dir+"/") {
			return true
		}
	}
	return false
}

func CreateDeterministicZip(zipPath string, entries []Entry) error {
	if err := os.MkdirAll(filepath.Dir(zipPath), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(zipPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	for _, entry := range entries {
		var (
			in  io.ReadCloser
			err error
		)
		if entry.Content != nil {
			in = io.NopCloser(bytes.NewReader(entry.Content))
		} else {
			in, err = os.Open(entry.AbsPath)
			if err != nil {
				_ = zw.Close()
				return err
			}
		}

		header := &zip.FileHeader{
			Name:   entry.RelPath,
			Method: zip.Deflate,
		}
		header.SetModTime(fixedZipTime)
		mode := fs.FileMode(0o644)
		if entry.Mode&0o111 != 0 {
			mode = 0o755
		}
		header.SetMode(mode)

		writer, err := zw.CreateHeader(header)
		if err != nil {
			in.Close()
			_ = zw.Close()
			return err
		}
		if _, err := io.Copy(writer, in); err != nil {
			in.Close()
			_ = zw.Close()
			return err
		}
		in.Close()
	}
	return zw.Close()
}

func SHA256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func WriteSHA256File(path, targetFile, hash string) error {
	line := fmt.Sprintf("%s  %s\n", hash, filepath.Base(targetFile))
	return os.WriteFile(path, []byte(line), 0o644)
}

func Inspect(zipPath string) (Inspection, error) {
	hash, err := SHA256File(zipPath)
	if err != nil {
		return Inspection{}, err
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return Inspection{}, err
	}
	defer zr.Close()

	entries := make([]string, 0, len(zr.File))
	var m *manifest.Manifest
	for _, file := range zr.File {
		entries = append(entries, file.Name)
		if file.Name != "manifest.json" {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return Inspection{}, err
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return Inspection{}, err
		}
		parsed, err := manifest.ParseJSON(b)
		if err == nil {
			m = &parsed
		}
	}
	sort.Strings(entries)

	sigPathA := strings.TrimSuffix(zipPath, filepath.Ext(zipPath)) + ".sig"
	sigPathB := strings.TrimSuffix(strings.TrimSuffix(zipPath, ".ubb"), ".zip") + ".sig"
	return Inspection{
		ArchivePath:  zipPath,
		SHA256:       hash,
		EntryCount:   len(entries),
		Entries:      entries,
		Manifest:     m,
		HasSignature: util.Exists(sigPathA) || util.Exists(sigPathB),
	}, nil
}

func IsForbiddenArchivePath(rel string) bool {
	rel = filepath.ToSlash(rel)
	base := strings.ToLower(filepath.Base(rel))
	if strings.HasPrefix(base, ".env") {
		return true
	}
	if base == ".ds_store" || base == "thumbs.db" {
		return true
	}
	if strings.HasSuffix(base, ".pem") || strings.HasSuffix(base, ".key") || strings.HasSuffix(base, ".tmp") || strings.HasSuffix(base, ".swp") {
		return true
	}
	if strings.Contains(strings.ToLower(rel), "private") && strings.HasSuffix(base, ".txt") {
		return true
	}
	if strings.Contains(rel, ".git/") || strings.HasPrefix(rel, ".git/") {
		return true
	}
	return false
}

func shouldSkipDir(rel string) bool {
	rel = strings.TrimSpace(filepath.ToSlash(rel))
	return rel == "dist" || rel == "tests" || rel == ".utilivo" || rel == "node_modules" || rel == ".git"
}

func shouldInclude(rel string) bool {
	rel = filepath.ToSlash(rel)
	if rel == "manifest.json" || rel == "AGENTS.md" || rel == "README.md" || rel == "LICENSE" || rel == "LICENSE.txt" || rel == "NOTICE" {
		return true
	}
	if strings.HasPrefix(rel, "tool/") {
		return true
	}
	return false
}
