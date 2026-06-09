package runner

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func execCmd(cmd, dir string, env []string) error {
	if strings.HasPrefix(cmd, "#!") {
		return execShebang(cmd, dir, env)
	}
	c := exec.Command("sh", "-c", cmd)
	c.Dir = dir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Env = env
	return c.Run()
}

func execShebang(cmd, dir string, env []string) error {
	tmp, err := os.CreateTemp("", "creo-shebang-*")
	if err != nil {
		return fmt.Errorf("creating temp script: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.WriteString(cmd); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("writing script: %w", err)
	}
	if err := tmp.Chmod(0755); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("chmod script: %w", err)
	}
	tmp.Close()

	c := exec.Command(tmpPath)
	c.Dir = dir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Env = env
	err = c.Run()
	os.Remove(tmpPath)
	return err
}
