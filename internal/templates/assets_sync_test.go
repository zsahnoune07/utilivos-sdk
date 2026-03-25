package templates_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestEmbeddedAssetsStaySyncedWithUISources(t *testing.T) {
	t.Parallel()
	_, file, _, _ := runtime.Caller(0)
	sdkRoot := filepath.Join(filepath.Dir(file), "..", "..")

	files := []string{"base.css", "layout.css", "components.css", "utilities.css", "patterns.css", "tool.js"}
	for _, name := range files {
		sourcePath := filepath.Join(sdkRoot, "ui", name)
		embeddedPath := filepath.Join(sdkRoot, "internal", "templates", "assets", name)

		source, err := os.ReadFile(sourcePath)
		if err != nil {
			t.Fatalf("read source %s: %v", sourcePath, err)
		}
		embedded, err := os.ReadFile(embeddedPath)
		if err != nil {
			t.Fatalf("read embedded %s: %v", embeddedPath, err)
		}
		if !bytes.Equal(source, embedded) {
			t.Fatalf("asset mismatch for %s; run scripts/sync_ui_assets.sh", name)
		}
	}
}
