package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/grimdork/creo/internal/lang"
)

func initProject(langName, ver string, force, verbose bool) {
	if force {
		if _, err := os.Stat(".creo"); err == nil {
			os.RemoveAll(".creo")
			if verbose {
				fmt.Println("  Removed .creo/")
			}
		}
	}

	if langName == "go" {
		ignores, err := lang.Init(".", ver, force, verbose)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		writeIgnores(ignores, verbose)
		fmt.Println("Project initialised")
		return
	}

	writeFiat(force, verbose)
	writeIgnores([]string{"/.creo"}, verbose)
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
