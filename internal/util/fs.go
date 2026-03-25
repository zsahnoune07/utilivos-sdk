package util

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var placeholderPattern = regexp.MustCompile(`\{\{[^}]+\}\}|<your-[^>]+>|REPLACE_ME|__PLACEHOLDER__`) // nolint:gochecknoglobals

func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func IsDir(path string) bool {
	st, err := os.Stat(path)
	if err != nil {
		return false
	}
	return st.IsDir()
}

func EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func ExpandHome(path string) (string, error) {
	if path == "" || path[0] != '~' {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if path == "~" {
		return home, nil
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:]), nil
	}
	return "", errors.New("unsupported home expansion")
}

func CopyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := EnsureDir(filepath.Dir(dst)); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func NormalizeSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, " ", "-")
	clean := make([]rune, 0, len(s))
	prevDash := false
	for _, r := range s {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlphaNum {
			clean = append(clean, r)
			prevDash = false
			continue
		}
		if r == '-' && !prevDash {
			clean = append(clean, r)
			prevDash = true
		}
	}
	out := strings.Trim(string(clean), "-")
	if out == "" {
		return "plugin"
	}
	return out
}

func ContainsPlaceholder(value string) bool {
	return placeholderPattern.MatchString(value)
}

func IsSubPath(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != "." && !strings.HasPrefix(rel, "..")
}
