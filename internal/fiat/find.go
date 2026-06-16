package fiat

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/grimdork/creo/internal/picker"
)

// FindFiat locates a fiat file by explicit path, or by searching the current directory for "fiat" or "*.fiat" files.
func FindFiat(explicitPath string) (string, bool) {
	if explicitPath != "" {
		if _, err := os.Stat(explicitPath); err == nil {
			return explicitPath, true
		}
		return "", false
	}
	if _, err := os.Stat("fiat"); err == nil {
		return "fiat", true
	}

	matches, err := filepath.Glob("*.fiat")
	if err != nil {
		return "", false
	}

	if len(matches) == 0 {
		return "", false
	}

	sort.Strings(matches)

	if len(matches) == 1 {
		return matches[0], true
	}

	selected, err := picker.Run(matches)
	if err != nil {
		return "", false
	}
	if selected == "" {
		return "", false
	}
	return selected, true
}
