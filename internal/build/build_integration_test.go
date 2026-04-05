package build_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"utilivo/sdk/internal/archive"
	"utilivo/sdk/internal/build"
	"utilivo/sdk/internal/scaffold"
	"utilivo/sdk/internal/signing"
	"utilivo/sdk/internal/templates"
	"utilivo/sdk/internal/util"
	"utilivo/sdk/internal/verify"
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

func TestPackHostModeOmitsToolUtilivo(t *testing.T) {
	t.Parallel()
	project := filepath.Join(t.TempDir(), "plugin")
	if _, err := scaffold.Create(scaffold.Options{TargetPath: project}); err != nil {
		t.Fatalf("scaffold create: %v", err)
	}
	manPath := filepath.Join(project, "manifest.json")
	manRaw, err := os.ReadFile(manPath)
	if err != nil {
		t.Fatal(err)
	}
	var man map[string]any
	if err := json.Unmarshal(manRaw, &man); err != nil {
		t.Fatal(err)
	}
	man["runtime_mode"] = "host"
	man["runtime_contract_version"] = 1
	out, err := json.MarshalIndent(man, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manPath, append(out, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "tool/index.html"), []byte(templates.IndexHTML("Pack Host", "host")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "tests/structure.test.yaml"), []byte(templates.StructureTestYAML("host")), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := verify.Run(project)
	if err != nil {
		t.Fatal(err)
	}
	if v.Report.Errors != 0 {
		t.Fatalf("verify: %d errors", v.Report.Errors)
	}
	artifacts, err := build.Pack(project, "")
	if err != nil {
		t.Fatalf("pack: %v", err)
	}
	zr, err := zip.OpenReader(artifacts.ArchivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "tool/utilivo/") {
			t.Fatalf("host .ubb must not contain %s", f.Name)
		}
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

	t.Run("large_rectangular_scales_max_edge_to_512", func(t *testing.T) {
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
		if iconConfig.Width != 512 || iconConfig.Height != 384 {
			t.Fatalf("expected packaged icon 512x384, got %dx%d", iconConfig.Width, iconConfig.Height)
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
	})

	t.Run("square_scales_to_512_per_edge", func(t *testing.T) {
		t.Parallel()
		project := filepath.Join(t.TempDir(), "plugin")
		if _, err := scaffold.Create(scaffold.Options{TargetPath: project}); err != nil {
			t.Fatalf("scaffold create: %v", err)
		}
		iconPath := filepath.Join(project, "tool", "assets", "icon.png")
		sourceIcon := generateTestIconPNG(t, 1024, 1024)
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
			t.Fatalf("expected packaged icon 512x512, got %dx%d", iconConfig.Width, iconConfig.Height)
		}
	})

	t.Run("small_icon_keeps_dimensions_recompresses", func(t *testing.T) {
		t.Parallel()
		project := filepath.Join(t.TempDir(), "plugin")
		if _, err := scaffold.Create(scaffold.Options{TargetPath: project}); err != nil {
			t.Fatalf("scaffold create: %v", err)
		}
		iconPath := filepath.Join(project, "tool", "assets", "icon.png")
		sourceIcon := generateTestIconPNG(t, 128, 128)
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
		if iconConfig.Width != 128 || iconConfig.Height != 128 {
			t.Fatalf("expected packaged icon 128x128, got %dx%d", iconConfig.Width, iconConfig.Height)
		}
		if len(iconBytes) >= len(sourceIcon) {
			t.Fatalf("expected recompressed icon to be smaller than no-compression source (%d >= %d)", len(iconBytes), len(sourceIcon))
		}
	})

	t.Run("transparent_icon_flattens_on_white_all_pixels_opaque", func(t *testing.T) {
		t.Parallel()
		const w, h = 256, 256
		project := filepath.Join(t.TempDir(), "plugin")
		if _, err := scaffold.Create(scaffold.Options{TargetPath: project}); err != nil {
			t.Fatalf("scaffold create: %v", err)
		}
		iconPath := filepath.Join(project, "tool", "assets", "icon.png")
		sourceIcon := generatePartiallyTransparentTestIconPNG(t, w, h)
		if err := os.WriteFile(iconPath, sourceIcon, 0o644); err != nil {
			t.Fatalf("write icon source: %v", err)
		}
		artifacts, err := build.Pack(project, "")
		if err != nil {
			t.Fatalf("pack: %v", err)
		}
		iconBytes := readArchiveEntryBytes(t, artifacts.ArchivePath, "tool/assets/icon.png")
		decoded, err := png.Decode(bytes.NewReader(iconBytes))
		if err != nil {
			t.Fatalf("decode archive icon: %v", err)
		}
		assertImageFullyOpaque(t, decoded)
		// Left half was fully transparent → should become white after flatten.
		r, g, b, _ := decoded.At(w/4, h/2).RGBA()
		if uint8(r>>8) != 255 || uint8(g>>8) != 255 || uint8(b>>8) != 255 {
			t.Fatalf("expected transparent region on white, got RGB(%d,%d,%d)", r>>8, g>>8, b>>8)
		}
		// Right half was opaque green.
		r2, g2, b2, _ := decoded.At(w*3/4, h/2).RGBA()
		if uint8(g2>>8) < 120 || uint8(r2>>8) > 40 || uint8(b2>>8) > 40 {
			t.Fatalf("expected green region preserved, got RGB(%d,%d,%d)", r2>>8, g2>>8, b2>>8)
		}
	})
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

func generatePartiallyTransparentTestIconPNG(t *testing.T, width, height int) []byte {
	t.Helper()

	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if x < width/2 {
				img.SetNRGBA(x, y, color.NRGBA{R: 200, G: 10, B: 10, A: 0})
			} else {
				img.SetNRGBA(x, y, color.NRGBA{R: 10, G: 180, B: 20, A: 255})
			}
		}
	}

	var out bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.NoCompression}
	if err := encoder.Encode(&out, img); err != nil {
		t.Fatalf("encode test icon: %v", err)
	}
	return out.Bytes()
}

func assertImageFullyOpaque(t *testing.T, img image.Image) {
	t.Helper()
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a != 0xffff {
				t.Fatalf("pixel at (%d,%d) is not fully opaque (alpha %d)", x, y, a>>8)
			}
		}
	}
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
