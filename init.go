package main

import (
	"fmt"
	"os"
	"strings"
)

func initProject(force, verbose bool) {
	if force {
		if _, err := os.Stat(".creo"); err == nil {
			os.RemoveAll(".creo")
			if verbose {
				fmt.Println("  Removed .creo/")
			}
		}
	}

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

	if _, err := os.Stat(".gitignore"); err == nil {
		data, err := os.ReadFile(".gitignore")
		if err != nil {
			return
		}
		if !strings.Contains(string(data), "/.creo/") {
			f, err := os.OpenFile(".gitignore", os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				return
			}
			defer f.Close()
			f.WriteString("/.creo/\n")
			if verbose {
				fmt.Println("  Added /.creo/ to .gitignore")
			}
		} else if verbose {
			fmt.Println("  Skipped .gitignore (already has /.creo/)")
		}
	} else {
		os.WriteFile(".gitignore", []byte("/.creo/\n"), 0644)
		if verbose {
			fmt.Println("  Created .gitignore")
		}
	}

	fmt.Println("Project initialised")
}
