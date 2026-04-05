package testing

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"utilivo/sdk/internal/util"
	"utilivo/sdk/internal/verify"
)

type Result struct {
	OK             bool         `json:"ok"`
	BuiltInPassed  bool         `json:"built_in_passed"`
	ExternalRun    bool         `json:"external_run"`
	ExternalPassed bool         `json:"external_passed"`
	Tests          int          `json:"tests"`
	Failures       int          `json:"failures"`
	Issues         []util.Issue `json:"issues,omitempty"`
	ExternalOutput string       `json:"external_output,omitempty"`
}

type ExternalMode string

const (
	ExternalAuto ExternalMode = "auto"
	ExternalSkip ExternalMode = "skip"
	ExternalOnly ExternalMode = "external"
)

func Run(root string, mode ExternalMode) (Result, error) {
	result := Result{OK: true}
	v, err := verify.Run(root)
	if err != nil {
		return Result{}, err
	}
	result.Tests++
	if v.Report.Errors == 0 {
		result.BuiltInPassed = true
	} else {
		result.OK = false
		result.Failures++
		result.Issues = append(result.Issues, v.Report.Issues...)
	}

	runExternal := mode == ExternalOnly || mode == ExternalAuto
	if runExternal {
		detected, passed, output, issues := runExternalTests(root)
		if detected {
			result.Tests++
			result.ExternalRun = true
			result.ExternalPassed = passed
			result.ExternalOutput = output
			if !passed {
				result.OK = false
				result.Failures++
			}
		}
		result.Issues = append(result.Issues, issues...)
	}

	if mode == ExternalOnly {
		if !result.ExternalRun {
			result.OK = false
			result.Failures++
			result.Issues = append(result.Issues, util.Issue{
				Severity: util.SeverityError,
				Code:     "test.external.missing",
				Message:  "external test mode requested but no external tests detected",
			})
		}
		result.BuiltInPassed = true
	}

	return result, nil
}

func runExternalTests(root string) (detected bool, passed bool, output string, issues []util.Issue) {
	pkgPath := filepath.Join(root, "package.json")
	if _, err := os.Stat(pkgPath); err != nil {
		return false, false, "", nil
	}
	b, err := os.ReadFile(pkgPath)
	if err != nil {
		issues = append(issues, util.Issue{
			Severity: util.SeverityWarning,
			Code:     "test.external.read",
			Path:     "package.json",
			Message:  "failed to read package.json for external tests",
		})
		return false, false, "", issues
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(b, &pkg); err != nil {
		issues = append(issues, util.Issue{
			Severity: util.SeverityWarning,
			Code:     "test.external.parse",
			Path:     "package.json",
			Message:  "failed to parse package.json for external tests",
		})
		return false, false, "", issues
	}
	testScript := strings.TrimSpace(pkg.Scripts["test"])
	if testScript == "" || strings.Contains(testScript, "no test specified") {
		return false, false, "", nil
	}

	npmPath, err := exec.LookPath("npm")
	if err != nil {
		issues = append(issues, util.Issue{
			Severity: util.SeverityWarning,
			Code:     "test.external.npm",
			Message:  "npm not found; skipping external tests",
		})
		return false, false, "", issues
	}

	cmd := exec.Command(npmPath, "test", "--silent")
	cmd.Dir = root
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err = cmd.Run()
	if err != nil {
		return true, false, buf.String(), issues
	}
	return true, true, buf.String(), issues
}
