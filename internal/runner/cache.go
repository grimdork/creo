package runner

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/grimdork/creo/internal/fiat"
)

type cacheEntry struct {
	Key string `json:"key"`
}

func cachePath(dir, targetName string) string {
	return filepath.Join(dir, ".creo", "cache", targetName+".json")
}

func computeCacheKey(sources []string, cmds []string) string {
	sort.Strings(sources)
	h := sha256.New()
	for _, src := range sources {
		data, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		h.Write([]byte(src))
		h.Write([]byte{0})
		h.Write(data)
	}
	for _, cmd := range cmds {
		h.Write([]byte(cmd))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func checkCache(dir, targetName string, sources []string, cmds []string) bool {
	expected := computeCacheKey(sources, cmds)
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
	key := computeCacheKey(sources, cmds)
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

func collectFilePaths(t *fiat.Target, f *fiat.File, dir string) []string {
	visited := map[string]bool{}
	var paths []string
	var walk func(t *fiat.Target)
	walk = func(t *fiat.Target) {
		if visited[t.Name] {
			return
		}
		visited[t.Name] = true
		if t.Sources != "" {
			srcPatterns := strings.Fields(fiat.ExpandWithTarget(t.Sources, f.Vars, t))
			for _, pat := range srcPatterns {
				files := globFiles(fiat.ExpandWithTarget(pat, f.Vars, t), dir)
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
	return paths
}
