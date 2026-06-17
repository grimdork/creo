package runner

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/grimdork/creo/internal/fiat"
	"github.com/grimdork/creo/internal/oci"
	"github.com/grimdork/creo/internal/util"
)

var (
	goVersionOnce sync.Once
	goVersion     string
)

func getGoVersion() string {
	goVersionOnce.Do(func() {
		out, err := exec.Command("go", "version").Output()
		if err == nil {
			goVersion = strings.TrimSpace(string(out))
		}
	})
	return goVersion
}

type cacheEntry struct {
	Key string `json:"key"`
}

func cachePath(dir, targetName string) string {
	return filepath.Join(dir, ".creo", "cache", targetName+".json")
}

// CleanCache removes all cached build artifacts from .creo/cache and the OCI image cache.
func CleanCache(dir string) error {
	creoDir := filepath.Join(dir, ".creo")
	if _, err := os.Stat(creoDir); err == nil {
		if err := os.RemoveAll(creoDir); err != nil {
			return fmt.Errorf("cleaning %s: %w", creoDir, err)
		}
	}
	if p, err := oci.OCICachePath(); err == nil {
		if err := os.RemoveAll(p); err != nil {
			return fmt.Errorf("removing OCI cache: %w", err)
		}
	}
	return nil
}

func computeCacheKey(sources []string, cmds []string) (string, error) {
	sorted := make([]string, len(sources))
	copy(sorted, sources)
	sort.Strings(sorted)
	h := sha256.New()
	for _, src := range sorted {
		data, err := os.ReadFile(src)
		if err != nil {
			return "", err
		}
		h.Write([]byte(src))
		h.Write([]byte{0})
		h.Write(data)
	}
	for _, cmd := range cmds {
		h.Write([]byte(cmd))
		h.Write([]byte{0})
	}
	if v := getGoVersion(); v != "" {
		h.Write([]byte(v))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func checkCache(dir, targetName string, sources []string, cmds []string) bool {
	expected, err := computeCacheKey(sources, cmds)
	if err != nil {
		return false
	}
	data, err := os.ReadFile(cachePath(dir, targetName))
	if err != nil {
		return false
	}
	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return false
	}
	return entry.Key == expected
}

func writeCache(dir, targetName string, sources []string, cmds []string) error {
	key, err := computeCacheKey(sources, cmds)
	if err != nil {
		return err
	}
	entry := cacheEntry{Key: key}
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}
	p := cachePath(dir, targetName)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}

func collectFilePaths(t *fiat.Target, f *fiat.File, dir string) ([]string, error) {
	visited := util.NewSet[string]()
	var paths []string
	var errs []error
	var walk func(t *fiat.Target)
	walk = func(t *fiat.Target) {
		if visited.Has(t.Name) {
			return
		}
		visited.Add(t.Name)
		if t.Sources != "" {
			srcPatterns := strings.Fields(fiat.ExpandWithTarget(t.Sources, f.Vars, t))
			for _, pat := range srcPatterns {
				files, err := util.GlobFiles(fiat.ExpandWithTarget(pat, f.Vars, t), dir)
				if err != nil {
					errs = append(errs, fmt.Errorf("pattern %q: %w", pat, err))
					continue
				}
				paths = append(paths, files...)
			}
		}
		for _, dep := range t.Requires {
			dt := fiat.FindTarget(f, dep)
			if dt != nil {
				walk(dt)
			}
		}
	}
	walk(t)
	paths = append(paths, f.Path())
	if len(errs) > 0 {
		return paths, errors.Join(errs...)
	}
	return paths, nil
}
