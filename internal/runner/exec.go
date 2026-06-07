package runner

import (
	"io"
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

func copyFile(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	sf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sf.Close()

	si, err := sf.Stat()
	if err != nil {
		return err
	}

	df, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, si.Mode())
	if err != nil {
		return err
	}
	defer df.Close()

	if _, err := io.Copy(df, sf); err != nil {
		return err
	}
	return nil
}
