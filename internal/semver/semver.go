package semver

import (
	"os/exec"
	"strings"
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
