package main

import (
	"fmt"
	"os"

	"github.com/grimdork/creo/internal/lang"
)

func initProject(langs []string, force, verbose bool) error {
	if force {
		if _, err := os.Stat(".creo"); err == nil {
			if err := os.RemoveAll(".creo"); err != nil {
				return fmt.Errorf("removing .creo: %w", err)
			}
			if verbose {
				fmt.Println("  Removed .creo/")
			}
		}
	}
	return lang.InitProject(langs, force, verbose)
}
