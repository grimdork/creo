package runner

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func execCmd(cmd, dir string, env []string) error {
	c := exec.Command("sh", "-c", cmd)
	c.Dir = dir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Env = env
	return c.Run()
}

func globFiles(pattern, dir string) []string {
	if strings.HasPrefix(pattern, "**") {
		ext := strings.TrimPrefix(pattern, "**")
		var files []string
		filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() && strings.HasSuffix(path, ext) {
				rel, _ := filepath.Rel(dir, path)
				files = append(files, filepath.Join(dir, rel))
			}
			return nil
		})
		return files
	}

	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return nil
	}
	return matches
}
