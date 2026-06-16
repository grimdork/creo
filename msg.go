package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/grimdork/climate/fx"
	"github.com/grimdork/creo/internal/fiat"
	"github.com/grimdork/creo/internal/targets"
)

var version string

func fail(err error) {
	fx.Fprint(os.Stderr, "{red}Error: {}{}{@}\n", err)
	os.Exit(1)
}

func failf(msg string, args ...interface{}) {
	fx.Fprint(os.Stderr, "{red}Error: {}{}{@}\n", fmt.Errorf(msg, args...))
	os.Exit(1)
}

func printVersion() {
	if version == "" {
		fx.Println("{bold}creo (dev){@}")
	} else {
		fx.Println("{bold}creo {}{@}", version)
	}
}

func listTargets(explicitPath string) (string, error) {
	fiatPath, ok := fiat.FindFiat(explicitPath)
	if !ok {
		return "", fmt.Errorf("no fiat file found")
	}
	file, err := fiat.Parse(fiatPath)
	if err != nil {
		return "", fmt.Errorf("parsing %s: %w", fiatPath, err)
	}
	if err := targets.Apply(file); err != nil {
		return "", fmt.Errorf("applying defaults to %s: %w", fiatPath, err)
	}

	var b strings.Builder
	b.WriteString("Available targets:\n")
	for _, t := range file.Targets {
		ln := t.Language
		if ln == "" {
			ln = "-"
		}
		if t.Desc != "" {
			desc := fiat.ExpandWithTarget(t.Desc, file.Vars, t)
			fmt.Fprintf(&b, "  %-15s (%s)   %s\n", t.Name, ln, desc)
		} else {
			fmt.Fprintf(&b, "  %-15s (%s)\n", t.Name, ln)
		}
	}
	return b.String(), nil
}

func runList(filePath string) {
	out, err := listTargets(filePath)
	if err != nil {
		fail(err)
	}
	fmt.Print(out)
}
