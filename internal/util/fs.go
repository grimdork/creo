package util

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

func GlobFiles(pattern, dir string) ([]string, error) {
	if !strings.Contains(pattern, "**") {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			return nil, err
		}
		return matches, nil
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
	err := filepath.WalkDir(walkRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		var matched bool
		if suffix == "" {
			matched = true
		} else if strings.Contains(suffix, "**") {
			rel, _ := filepath.Rel(walkRoot, path)
			matched = matchGlob(suffix, rel)
		} else if strings.ContainsAny(suffix, "*?[") {
			if strings.Contains(suffix, "/") {
				rel, _ := filepath.Rel(walkRoot, path)
				matched, _ = filepath.Match(suffix, rel)
			} else {
				matched, _ = filepath.Match(suffix, filepath.Base(path))
			}
		} else {
			rel, _ := filepath.Rel(walkRoot, path)
			matched = rel == suffix || strings.HasSuffix(rel, "/"+suffix)
		}
		if matched {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return files, err
	}
	return files, nil
}

func matchGlob(pattern, name string) bool {
	parts := strings.Split(pattern, "/")
	nameParts := strings.Split(name, "/")

	var match func(pi, ni int) bool
	match = func(pi, ni int) bool {
		for pi < len(parts) && ni < len(nameParts) {
			if parts[pi] == "**" {
				pi++
				if pi >= len(parts) {
					return true
				}
				for ni <= len(nameParts) {
					if match(pi, ni) {
						return true
					}
					ni++
				}
				return false
			}
			m, _ := filepath.Match(parts[pi], nameParts[ni])
			if !m {
				return false
			}
			pi++
			ni++
		}
		return pi == len(parts) && ni == len(nameParts)
	}
	return match(0, 0)
}

func CopyFile(src, dest string) (err error) {
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

	tmpPath := dest + ".tmp"
	df, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, si.Mode())
	if err != nil {
		return err
	}

	if _, err := io.Copy(df, sf); err != nil {
		df.Close()
		os.Remove(tmpPath)
		return err
	}

	if err := df.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	if err := os.Rename(tmpPath, dest); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}
