package testing_test

import (
	"path/filepath"
	"runtime"
	"testing"

	plugintesting "utilivos/sdk/internal/testing"
)

func TestRunBuiltInTests(t *testing.T) {
	t.Parallel()
	result, err := plugintesting.Run(fixturePath("valid-minimal"), plugintesting.ExternalSkip)
	if err != nil {
		t.Fatalf("testing run: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected success, got %+v", result)
	}
}

func TestRunBuiltInTestsFailure(t *testing.T) {
	t.Parallel()
	result, err := plugintesting.Run(fixturePath("invalid-tier"), plugintesting.ExternalSkip)
	if err != nil {
		t.Fatalf("testing run: %v", err)
	}
	if result.OK {
		t.Fatalf("expected failure, got %+v", result)
	}
}

func fixturePath(name string) string {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..")
	return filepath.Join(root, "tests", "fixtures", name)
}
