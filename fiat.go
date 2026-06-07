package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

	selected, err := Run(matches)
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

func findFiatInDir(dir string, verbose bool) (string, bool) {
	path := filepath.Join(dir, "fiat")
	if _, err := os.Stat(path); err == nil {
		return path, true
	}

	matches, err := filepath.Glob(filepath.Join(dir, "*.fiat"))
	if err != nil {
		return "", false
	}

	if len(matches) == 1 {
		return matches[0], true
	}

	if len(matches) > 1 {
		if verbose {
			fmt.Printf("  Skipped %s (multiple .fiat files)\n", dir)
		}
	}
	return "", false
}
