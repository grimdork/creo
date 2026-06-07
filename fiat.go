package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/grimdork/creo/internal/picker"
)

func findFiat() (string, bool) {
	if _, err := os.Stat("fiat"); err == nil {
		return "fiat", true
	}

	matches, err := filepath.Glob("*.fiat")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error scanning for .fiat files:", err)
		return "", false
	}

	if len(matches) == 0 {
		fmt.Fprintln(os.Stderr, "No .fiat files found")
		return "", false
	}

	sort.Strings(matches)

	if len(matches) == 1 {
		return matches[0], true
	}

	selected, err := picker.Run(matches)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Picker error:", err)
		return "", false
	}
	if selected == "" {
		fmt.Fprintln(os.Stderr, "Cancelled")
		return "", false
	}
	return selected, true
}
