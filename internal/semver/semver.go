package semver

import (
	"os/exec"
	"strings"
	"time"
)

// String returns a version from git describe. Appends -dirty if the
// working tree has uncommitted changes. Returns "dev" if not in a git
// repo or if no tags exist.
func String() string {
	out, err := exec.Command("git", "describe", "--tags", "--abbrev=8").Output()
	if err != nil {
		return "dev"
	}
	ver := strings.TrimSpace(string(out))

	if idx := strings.LastIndex(ver, "-g"); idx >= 0 {
		ver = ver[:idx+1] + ver[idx+2:]
	}

	if exec.Command("git", "diff-index", "--quiet", "HEAD").Run() != nil {
		ver += "-dirty"
	}
	return ver
}

// CommitString returns the short commit hash. Returns "unknown" if not
// in a git repo.
func CommitString() string {
	out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

// DateString returns the current UTC timestamp in ISO 8601 format.
func DateString() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}
