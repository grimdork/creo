package main

import (
	"fmt"
	"os"
	"strings"
)

func initProject(force bool) {
	if force {
		if _, err := os.Stat(".creo"); err == nil {
			os.RemoveAll(".creo")
			fmt.Println("  Removed .creo/")
		}
	}

	if _, err := os.Stat("fiat"); err == nil {
		if force {
			os.WriteFile("fiat", []byte("build: go\n"), 0644)
			fmt.Println("  Replaced fiat")
		} else {
			fmt.Println("  Skipped fiat (already exists)")
		}
	} else {
		os.WriteFile("fiat", []byte("build: go\n"), 0644)
		fmt.Println("  Created fiat")
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
			fmt.Println("  Added /.creo/ to .gitignore")
		} else {
			fmt.Println("  Skipped .gitignore (already has /.creo/)")
		}
	} else {
		os.WriteFile(".gitignore", []byte("/.creo/\n"), 0644)
		fmt.Println("  Created .gitignore")
	}
}
