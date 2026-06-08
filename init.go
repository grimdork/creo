package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/grimdork/creo/internal/lang"
)

func initProject(langs []string, force, verbose bool) {
	if force {
		if _, err := os.Stat(".creo"); err == nil {
			os.RemoveAll(".creo")
			if verbose {
				fmt.Println("  Removed .creo/")
			}
		}
	}

	if len(langs) == 0 {
		writeFiat(force, verbose)
		writeIgnores([]string{"/.creo"}, verbose)
		fmt.Println("Project initialised")
		return
	}

	var allIgnores []string

	for _, spec := range langs {
		langName, ver := spec, ""
		if idx := strings.IndexByte(spec, ':'); idx >= 0 {
			langName, ver = spec[:idx], spec[idx+1:]
		}

		var ignores []string
		var err error

		switch langName {
		case "go":
			ignores, err = lang.Init(".", ver, force, verbose)
		case "c":
			ignores, err = lang.InitC(".", force, verbose)
		case "cxx", "cpp":
			ignores, err = lang.InitCxx(".", force, verbose)
		case "oci":
			ignores, err = lang.InitOci(".", force, verbose)
		default:
			ignores, err = nil, fmt.Errorf("unknown language: %s", langName)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		allIgnores = append(allIgnores, ignores...)
	}

	writeIgnores(allIgnores, verbose)
	fmt.Println("Project initialised")
}

func writeFiat(force, verbose bool) {
	if _, err := os.Stat("fiat"); err == nil {
		if force {
			os.WriteFile("fiat", []byte("build: go\n"), 0644)
			if verbose {
				fmt.Println("  Replaced fiat")
			}
		} else if verbose {
			fmt.Println("  Skipped fiat (already exists)")
		}
	} else {
		os.WriteFile("fiat", []byte("build: go\n"), 0644)
		if verbose {
			fmt.Println("  Created fiat")
		}
	}
}

func writeIgnores(lines []string, verbose bool) {
	lines = unique(lines)
	if _, err := os.Stat(".gitignore"); err == nil {
		data, _ := os.ReadFile(".gitignore")
		content := string(data)
		added := false
		for _, line := range lines {
			if !strings.Contains(content, line+"\n") && !strings.Contains(content, line+" ") {
				f, err := os.OpenFile(".gitignore", os.O_APPEND|os.O_WRONLY, 0644)
				if err != nil {
					return
				}
				f.WriteString(line + "\n")
				f.Close()
				added = true
			}
		}
		if added && verbose {
			fmt.Println("  Updated .gitignore")
		} else if verbose {
			fmt.Println("  Skipped .gitignore")
		}
	} else {
		content := strings.Join(lines, "\n") + "\n"
		os.WriteFile(".gitignore", []byte(content), 0644)
		if verbose {
			fmt.Println("  Created .gitignore")
		}
	}
}

func unique(s []string) []string {
	seen := map[string]bool{}
	r := make([]string, 0, len(s))
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			r = append(r, v)
		}
	}
	return r
}
