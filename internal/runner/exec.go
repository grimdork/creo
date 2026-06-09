package runner

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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

	if strings.Contains(suffix, "**") {
		segments := strings.Split(suffix, "/")
		filePat := segments[len(segments)-1]
		filepath.WalkDir(walkRoot, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() {
				matched, _ := filepath.Match(filePat, filepath.Base(path))
				if matched {
					files = append(files, path)
				}
			}
			return nil
		})
		return files
	}

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

func copyFile(src, dest string) (err error) {
	absSrc, err := filepath.Abs(src)
	if err != nil {
		return err
	}
	absDest, err := filepath.Abs(dest)
	if err != nil {
		return err
	}
	if absSrc == absDest {
		return nil
	}

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

	if _, err := io.Copy(df, sf); err != nil {
		df.Close()
		os.Remove(dest)
		return err
	}

	if err := df.Close(); err != nil {
		return err
	}
	return nil
}
