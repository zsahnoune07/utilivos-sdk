package util

import "regexp"

var semverPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?$`) // nolint:gochecknoglobals

func IsSemver(v string) bool {
	return semverPattern.MatchString(v)
}
