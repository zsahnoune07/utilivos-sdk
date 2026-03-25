package util

import "encoding/json"

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

type Issue struct {
	Severity Severity `json:"severity"`
	Code     string   `json:"code"`
	Path     string   `json:"path,omitempty"`
	Message  string   `json:"message"`
}

type Report struct {
	OK       bool    `json:"ok"`
	Errors   int     `json:"errors"`
	Warnings int     `json:"warnings"`
	Issues   []Issue `json:"issues"`
}

func NewReport() Report {
	return Report{OK: true}
}

func (r *Report) Add(severity Severity, code, path, message string) {
	if severity == SeverityError {
		r.Errors++
		r.OK = false
	} else if severity == SeverityWarning {
		r.Warnings++
	}
	r.Issues = append(r.Issues, Issue{
		Severity: severity,
		Code:     code,
		Path:     path,
		Message:  message,
	})
}

func (r *Report) Merge(other Report) {
	if !other.OK {
		r.OK = false
	}
	r.Errors += other.Errors
	r.Warnings += other.Warnings
	r.Issues = append(r.Issues, other.Issues...)
}

func (r Report) ExitCode(strict bool) int {
	if r.Errors > 0 {
		return ExitValidationError
	}
	if strict && r.Warnings > 0 {
		return ExitValidationError
	}
	return ExitOK
}

func (r Report) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
