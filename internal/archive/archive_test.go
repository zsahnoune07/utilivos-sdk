package archive_test

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"utilivo/sdk/internal/archive"
	"utilivo/sdk/internal/scaffold"
)

func TestDropToolUtilivoForHostMode(t *testing.T) {
	t.Parallel()
	entries := []archive.Entry{
		{RelPath: "manifest.json"},
		{RelPath: "tool/index.html"},
		{RelPath: "tool/utilivo/base.css"},
		{RelPath: "tool/utilivo/tool.js"},
	}
	filtered := archive.DropToolUtilivoForHostMode(entries, true)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 entries, got %d: %#v", len(filtered), filtered)
	}
	if archive.DropToolUtilivoForHostMode(entries, false) == nil {
		t.Fatal("expected non-nil")
	}
	if len(archive.DropToolUtilivoForHostMode(entries, false)) != len(entries) {
		t.Fatal("bundled mode should keep all entries")
	}
}

func TestDeterministicZipPackaging(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	project := filepath.Join(dir, "plugin")
	if _, err := scaffold.Create(scaffold.Options{TargetPath: project}); err != nil {
		t.Fatalf("scaffold create: %v", err)
	}

	entriesA, err := archive.CollectPackageFiles(project)
	if err != nil {
		t.Fatalf("collect files A: %v", err)
	}
	zipA := filepath.Join(dir, "a.ubb")
	if err := archive.CreateDeterministicZip(zipA, entriesA); err != nil {
		t.Fatalf("create zip A: %v", err)
	}

	indexPath := filepath.Join(project, "tool", "index.html")
	now := time.Now().Add(10 * time.Minute)
	if err := os.Chtimes(indexPath, now, now); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	entriesB, err := archive.CollectPackageFiles(project)
	if err != nil {
		t.Fatalf("collect files B: %v", err)
	}
	zipB := filepath.Join(dir, "b.ubb")
	if err := archive.CreateDeterministicZip(zipB, entriesB); err != nil {
		t.Fatalf("create zip B: %v", err)
	}

	aBytes, err := os.ReadFile(zipA)
	if err != nil {
		t.Fatalf("read zip A: %v", err)
	}
	bBytes, err := os.ReadFile(zipB)
	if err != nil {
		t.Fatalf("read zip B: %v", err)
	}
	if !bytes.Equal(aBytes, bBytes) {
		t.Fatal("expected deterministic zip bytes to match")
	}
}

func TestCollectPackageFilesHonorsBuildIgnore(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	project := filepath.Join(dir, "plugin")
	if _, err := scaffold.Create(scaffold.Options{TargetPath: project}); err != nil {
		t.Fatalf("scaffold create: %v", err)
	}
	debugPath := filepath.Join(project, "tool", "debug.txt")
	if err := os.WriteFile(debugPath, []byte("debug\n"), 0o644); err != nil {
		t.Fatalf("write debug file: %v", err)
	}
	buildIgnorePath := filepath.Join(project, "build.ignore")
	buildIgnoreContent, err := os.ReadFile(buildIgnorePath)
	if err != nil {
		t.Fatalf("read build.ignore: %v", err)
	}
	updatedBuildIgnore := string(buildIgnoreContent) + "tool/debug.txt\n"
	if err := os.WriteFile(buildIgnorePath, []byte(updatedBuildIgnore), 0o644); err != nil {
		t.Fatalf("write build.ignore: %v", err)
	}

	entries, err := archive.CollectPackageFiles(project)
	if err != nil {
		t.Fatalf("collect files: %v", err)
	}

	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		paths = append(paths, entry.RelPath)
	}

	if slices.Contains(paths, "README.md") {
		t.Fatal("expected README.md to be excluded from package entries")
	}
	if slices.Contains(paths, "PLAN.md") {
		t.Fatal("expected PLAN.md to be excluded from package entries")
	}
	if slices.Contains(paths, "tool/debug.txt") {
		t.Fatal("expected tool/debug.txt to be excluded from package entries")
	}
	if !slices.Contains(paths, "AGENTS.md") {
		t.Fatal("expected AGENTS.md to remain in package entries")
	}
}
