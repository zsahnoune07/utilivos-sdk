package build_test

import (
	"archive/zip"
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/utilivo/utilivo-sdk/internal/archive"
	"github.com/utilivo/utilivo-sdk/internal/build"
	"github.com/utilivo/utilivo-sdk/internal/scaffold"
	"github.com/utilivo/utilivo-sdk/internal/signing"
	"github.com/utilivo/utilivo-sdk/internal/util"
	"github.com/utilivo/utilivo-sdk/internal/verify"
)

func TestCreateVerifyBuildFlow(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	project := filepath.Join(dir, "hello-plugin")
	if _, err := scaffold.Create(scaffold.Options{TargetPath: project}); err != nil {
		t.Fatalf("scaffold create: %v", err)
	}

	v, err := verify.Run(project)
	if err != nil {
		t.Fatalf("verify run: %v", err)
	}
	if v.Report.Errors != 0 {
		t.Fatalf("verify expected no errors, got %d", v.Report.Errors)
	}

	pair, err := signing.GenerateKeyPair(filepath.Join(dir, "keys"), "ci")
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}

	artifacts, err := build.Pack(project, "")
	if err != nil {
		t.Fatalf("pack: %v", err)
	}
	if err := build.SignArchive(&artifacts, pair.PrivatePath); err != nil {
		t.Fatalf("sign archive: %v", err)
	}

	required := []string{artifacts.ArchivePath, artifacts.ManifestPath, artifacts.SHA256Path, artifacts.SignaturePath}
	for _, path := range required {
		if !util.Exists(path) {
			t.Fatalf("expected artifact at %s", path)
		}
	}
	if err := signing.VerifyFileSignature(pair.PublicPath, artifacts.ArchivePath, artifacts.SignaturePath); err != nil {
		t.Fatalf("verify signed archive: %v", err)
	}
}

func TestPackCreatesGeneratedDistWhenMissing(t *testing.T) {
	t.Parallel()
	project := filepath.Join(t.TempDir(), "plugin")
	if _, err := scaffold.Create(scaffold.Options{TargetPath: project}); err != nil {
		t.Fatalf("scaffold create: %v", err)
	}
	if err := os.RemoveAll(filepath.Join(project, "dist")); err != nil {
		t.Fatalf("remove dist: %v", err)
	}

	artifacts, err := build.Pack(project, "")
	if err != nil {
		t.Fatalf("pack without dist: %v", err)
	}
	if !util.IsDir(artifacts.DistDir) {
		t.Fatalf("expected generated dist directory at %s", artifacts.DistDir)
	}
}

func TestPackNormalizesRelativeOutDirAgainstProject(t *testing.T) {
	t.Parallel()
	project := filepath.Join(t.TempDir(), "plugin")
	if _, err := scaffold.Create(scaffold.Options{TargetPath: project}); err != nil {
		t.Fatalf("scaffold create: %v", err)
	}

	artifacts, err := build.Pack(project, "artifacts")
	if err != nil {
		t.Fatalf("pack with relative out dir: %v", err)
	}

	expectedDist := filepath.Join(project, "artifacts")
	if artifacts.DistDir != expectedDist {
		t.Fatalf("expected dist dir %s, got %s", expectedDist, artifacts.DistDir)
	}
}

func TestBuildCreatesGeneratedDistWhenMissing(t *testing.T) {
	t.Parallel()
	project := filepath.Join(t.TempDir(), "plugin")
	if _, err := scaffold.Create(scaffold.Options{TargetPath: project}); err != nil {
		t.Fatalf("scaffold create: %v", err)
	}
	if err := os.RemoveAll(filepath.Join(project, "dist")); err != nil {
		t.Fatalf("remove dist: %v", err)
	}

	result, err := build.Build(build.BuildOptions{
		Root:      project,
		SkipTests: true,
	})
	if err != nil {
		t.Fatalf("build with missing dist: %v", err)
	}
	if !util.IsDir(result.Artifacts.DistDir) {
		t.Fatalf("expected generated dist directory at %s", result.Artifacts.DistDir)
	}
	if !util.Exists(result.Artifacts.ArchivePath) {
		t.Fatalf("expected archive at %s", result.Artifacts.ArchivePath)
	}
}

func TestBuildFailsForMissingRequiredSource(t *testing.T) {
	t.Parallel()
	project := filepath.Join(t.TempDir(), "plugin")
	if _, err := scaffold.Create(scaffold.Options{TargetPath: project}); err != nil {
		t.Fatalf("scaffold create: %v", err)
	}
	if err := os.Remove(filepath.Join(project, "tool", "main.js")); err != nil {
		t.Fatalf("remove tool/main.js: %v", err)
	}

	result, err := build.Build(build.BuildOptions{
		Root:      project,
		SkipTests: true,
	})
	if err == nil {
		t.Fatal("expected build failure for missing required source")
	}
	if !strings.Contains(err.Error(), "verification failed") {
		t.Fatalf("expected verification failure error, got: %v", err)
	}
	if result.VerifyResult.Report.Errors == 0 {
		t.Fatalf("expected verify errors in build result, got %+v", result.VerifyResult.Report)
	}

	found := false
	for _, issue := range result.VerifyResult.Report.Issues {
		if issue.Code == "structure.required_source" && issue.Path == "tool/main.js" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected structure.required_source issue for tool/main.js, got %+v", result.VerifyResult.Report.Issues)
	}
}

func TestPackHonorsBuildIgnore(t *testing.T) {
	t.Parallel()
	project := filepath.Join(t.TempDir(), "plugin")
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

	artifacts, err := build.Pack(project, "")
	if err != nil {
		t.Fatalf("pack: %v", err)
	}

	info, err := archive.Inspect(artifacts.ArchivePath)
	if err != nil {
		t.Fatalf("inspect archive: %v", err)
	}
	if slices.Contains(info.Entries, "README.md") {
		t.Fatal("expected README.md to be excluded from archive")
	}
	if slices.Contains(info.Entries, "PLAN.md") {
		t.Fatal("expected PLAN.md to be excluded from archive")
	}
	if slices.Contains(info.Entries, "tool/debug.txt") {
		t.Fatal("expected tool/debug.txt to be excluded from archive")
	}
	if !slices.Contains(info.Entries, "tool/index.html") {
		t.Fatal("expected tool/index.html to be included in archive")
	}
}

func TestPackOptimizesManifestIconPNG(t *testing.T) {
	t.Parallel()
	project := filepath.Join(t.TempDir(), "plugin")
	if _, err := scaffold.Create(scaffold.Options{TargetPath: project}); err != nil {
		t.Fatalf("scaffold create: %v", err)
	}

	iconPath := filepath.Join(project, "tool", "assets", "icon.png")
	sourceIcon := generateTestIconPNG(t, 1024, 768)
	if err := os.WriteFile(iconPath, sourceIcon, 0o644); err != nil {
		t.Fatalf("write icon source: %v", err)
	}

	artifacts, err := build.Pack(project, "")
	if err != nil {
		t.Fatalf("pack: %v", err)
	}

	iconBytes := readArchiveEntryBytes(t, artifacts.ArchivePath, "tool/assets/icon.png")
	iconConfig, err := png.DecodeConfig(bytes.NewReader(iconBytes))
	if err != nil {
		t.Fatalf("decode archive icon: %v", err)
	}
	if iconConfig.Width != 512 || iconConfig.Height != 512 {
		t.Fatalf("expected packaged icon to be 512x512, got %dx%d", iconConfig.Width, iconConfig.Height)
	}
	if len(iconBytes) >= len(sourceIcon) {
		t.Fatalf("expected optimized icon to be smaller than source (%d >= %d)", len(iconBytes), len(sourceIcon))
	}

	currentSource, err := os.ReadFile(iconPath)
	if err != nil {
		t.Fatalf("read icon source after pack: %v", err)
	}
	if !bytes.Equal(currentSource, sourceIcon) {
		t.Fatal("expected source icon to remain unchanged after pack")
	}
}

func generateTestIconPNG(t *testing.T, width, height int) []byte {
	t.Helper()

	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8((x * 255) / max(1, width-1)),
				G: uint8((y * 255) / max(1, height-1)),
				B: uint8(((x + y) * 255) / max(1, width+height-2)),
				A: 255,
			})
		}
	}

	var out bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.NoCompression}
	if err := encoder.Encode(&out, img); err != nil {
		t.Fatalf("encode test icon: %v", err)
	}
	return out.Bytes()
}

func readArchiveEntryBytes(t *testing.T, archivePath, relPath string) []byte {
	t.Helper()

	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		if file.Name != relPath {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("open archive entry: %v", err)
		}
		defer rc.Close()

		data, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("read archive entry: %v", err)
		}
		return data
	}

	t.Fatalf("archive entry not found: %s", relPath)
	return nil
}
