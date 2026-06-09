package util

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

func GlobFiles(pattern, dir string) []string {
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
