package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func runFiat(path string) {
	fmt.Printf("Selected: %s\n", path)
}

func main() {
	if _, err := os.Stat("fiat"); err == nil {
		runFiat("fiat")
		return
	}

	matches, err := filepath.Glob("*.fiat")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning for .fiat files: %v\n", err)
		os.Exit(1)
	}

	if len(matches) == 0 {
		fmt.Fprintln(os.Stderr, "No .fiat files found")
		os.Exit(1)
	}

	sort.Strings(matches)

	if len(matches) == 1 {
		runFiat(matches[0])
		return
	}

	selected, err := Run(matches)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Picker error: %v\n", err)
		os.Exit(1)
	}
	if selected == "" {
		fmt.Fprintln(os.Stderr, "Cancelled")
		os.Exit(1)
	}
	runFiat(selected)
}
