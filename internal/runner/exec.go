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
	if !strings.Contains(pattern, "**") {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			return nil
		}
		return matches
	}

	idx := strings.Index(pattern, "**")
	prefix := pattern[:idx]
	suffix := pattern[idx+2:]

	var walkRoot string
	if prefix != "" {
		prefix = strings.TrimSuffix(prefix, "/")
		walkRoot = filepath.Join(dir, prefix)
	} else {
		walkRoot = dir
	}

	suffix = strings.TrimPrefix(suffix, "/")

	var files []string
	filepath.WalkDir(walkRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			matched := false
			if suffix == "" {
				matched = true
			} else if strings.ContainsAny(suffix, "*?[") {
				matched, _ = filepath.Match(suffix, filepath.Base(path))
			} else {
				matched = strings.HasSuffix(filepath.Base(path), suffix)
			}
			if matched {
				files = append(files, path)
			}
		}
		return nil
	})
	return files
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
