package version

import (
	"regexp"
	"strconv"
	"strings"
)

// Version specifies the current version of Draupnir
// This value is injected in at compile time (see the Makefile)
var Version string

// ParseSemver extracts the major minor and patch level versions from a version string.
func ParseSemver(version string) (int, int, int) {
	if !regexp.MustCompile("^\\d+\\.\\d+\\.\\d+$").Match([]byte(version)) {
		return -1, -1, -1
	}

	mustAtoi := func(s string) int { i, _ := strconv.Atoi(s); return i }

	components := strings.Split(version, ".")
	return mustAtoi(components[0]), mustAtoi(components[1]), mustAtoi(components[2])
}
