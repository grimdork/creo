package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/grimdork/climate/fx"
	"github.com/grimdork/creo/internal/fiat"
	"github.com/grimdork/creo/internal/targets"
)

// ListTargets returns a formatted listing of all targets in the fiat file.
func ListTargets(explicitPath string) (string, error) {
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

// RunList prints all available targets from the fiat file.
func RunList(filePath string) error {
	out, err := ListTargets(filePath)
	if err != nil {
		return err
	}
	fx.Fprint(os.Stdout, "{}", out)
	return nil
}
