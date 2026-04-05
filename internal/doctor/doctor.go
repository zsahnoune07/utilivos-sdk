package doctor

import (
	"os/exec"
	"path/filepath"
	"runtime"

	"utilivo/sdk/internal/signing"
	"utilivo/sdk/internal/util"
)

type Status string

const (
	StatusPass Status = "pass"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
)

type Check struct {
	Name    string `json:"name"`
	Status  Status `json:"status"`
	Details string `json:"details"`
}

type Report struct {
	OK     bool    `json:"ok"`
	Checks []Check `json:"checks"`
}

func Run(root string) Report {
	checks := []Check{
		{Name: "utilivo", Status: StatusPass, Details: "version " + util.Version},
		{Name: "runtime", Status: StatusPass, Details: runtime.GOOS + "/" + runtime.GOARCH},
		{Name: "go", Status: StatusPass, Details: runtime.Version()},
	}

	manifestPath := filepath.Join(root, "manifest.json")
	if util.Exists(manifestPath) {
		checks = append(checks, Check{Name: "project", Status: StatusPass, Details: "manifest.json found"})
	} else {
		checks = append(checks, Check{Name: "project", Status: StatusWarn, Details: "manifest.json not found in current directory"})
	}

	priv, _, err := signing.DefaultKeyPaths()
	if err == nil && util.Exists(priv) {
		checks = append(checks, Check{Name: "signing-key", Status: StatusPass, Details: priv})
	} else {
		checks = append(checks, Check{Name: "signing-key", Status: StatusWarn, Details: "dev key missing; run utilivo keygen"})
	}

	if _, err := exec.LookPath("npm"); err == nil {
		checks = append(checks, Check{Name: "npm", Status: StatusPass, Details: "npm available"})
	} else {
		checks = append(checks, Check{Name: "npm", Status: StatusWarn, Details: "npm not installed (external JS tests disabled)"})
	}

	if _, err := exec.LookPath("dpkg-deb"); err == nil {
		checks = append(checks, Check{Name: "dpkg-deb", Status: StatusPass, Details: "deb packaging available"})
	} else {
		checks = append(checks, Check{Name: "dpkg-deb", Status: StatusWarn, Details: "dpkg-deb missing; deb packaging unavailable"})
	}

	ok := true
	for _, c := range checks {
		if c.Status == StatusFail {
			ok = false
			break
		}
	}
	return Report{OK: ok, Checks: checks}
}
