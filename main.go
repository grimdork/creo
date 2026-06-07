package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/grimdork/climate/arg"
)

func runFiat(path string) {
	fmt.Printf("Selected: %s\n", path)
}

func main() {
	opt := arg.New("creo", "A make-like build tool")
	opt.SetDefaultHelp(true)
	opt.SetFlag(arg.GroupDefault, "i", "init", "Initialise project with base files")
	opt.SetFlag(arg.GroupDefault, "f", "force", "Force overwrite existing files")

	err := opt.Parse(os.Args[1:])
	if err != nil {
		if !errors.Is(err, arg.ErrNonFatal) {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	if opt.GetBool("i") {
		initProject(opt.GetBool("f"))
		return
	}

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
