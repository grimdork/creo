package runner

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/grimdork/climate/fx"
	"github.com/grimdork/creo/internal/fiat"
	"github.com/grimdork/creo/internal/targets"
)

// findFiatInDir looks for a fiat file (named "fiat" or "*.fiat") in the given directory and returns its path.
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
			fx.Println(`  {muted}Skipped {} (multiple .fiat files){@}`, dir)
		}
	}
	return "", false
}

// RunRecursive walks a directory tree looking for fiat files and builds the given target in each.
func RunRecursive(dir string, targetName string, opts RunOpts) error {
	if opts.Results == nil {
		opts.Results = &TargetResults{}
	}
	var walkErr []error
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if opts.Verbose {
				fx.Fprint(os.Stderr, "  {warning}{}: {}{@}\n", path, err)
			}
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") && path != dir {
			return filepath.SkipDir
		}

		fiatPath, ok := findFiatInDir(path, opts.Verbose)
		if !ok {
			return nil
		}

		file, err := fiat.Parse(fiatPath)
		if err != nil {
			walkErr = append(walkErr, fmt.Errorf("parsing %s: %w", fiatPath, err))
			return nil
		}
		if opts.BuildDir != "" {
			file.Vars["BUILDDIR"] = &fiat.Var{Name: "BUILDDIR", Value: opts.BuildDir}
		}
		if err := targets.Apply(file); err != nil {
			walkErr = append(walkErr, fmt.Errorf("applying defaults to %s: %w", fiatPath, err))
			return nil
		}

		if opts.Verbose {
			fx.Println(`{cyan}Entering {}{@}`, path)
		}
		if err := RunTarget(file, targetName, opts); err != nil {
			walkErr = append(walkErr, fmt.Errorf("in %s: %w", path, err))
		}
		return nil
	})
	return errors.Join(walkErr...)
}
