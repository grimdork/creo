package runner

import (
	"fmt"
	"os/exec"
	"strings"
)

// execCredHelper runs a credential helper command and returns the user:pass pair from its output.
func execCredHelper(helper, dir string) (user, pass string, err error) {
	parts := strings.Fields(helper)
	if len(parts) == 0 {
		return "", "", fmt.Errorf("empty credential helper")
	}
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = dir
	out, execErr := cmd.Output()
	if execErr != nil {
		return "", "", fmt.Errorf("credential helper %q failed: %w", parts[0], execErr)
	}
	line := strings.TrimSpace(string(out))
	idx := strings.IndexByte(line, ':')
	if idx < 0 {
		return "", line, nil
	}
	return line[:idx], line[idx+1:], nil
}
