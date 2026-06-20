package cli

import (
	"fmt"
	"os"

	"github.com/grimdork/climate/arg"
	"github.com/grimdork/climate/fx"
	"github.com/grimdork/creo/internal/fiat"
	"github.com/grimdork/creo/internal/runner"
	"github.com/grimdork/creo/internal/targets"
)

// RunBuild parses the fiat file and runs the specified build targets.
func RunBuild(opt *arg.Options) error {
	if opt.GetBool("no-color") || opt.GetBool("no-colour") {
		os.Setenv("NO_COLOR", "1")
	}

	results := &runner.TargetResults{}
	cacheRemote := opt.GetString("cache-remote")
	if cacheRemote == "" {
		cacheRemote = os.Getenv("CREO_CACHE_REMOTE")
	}
	cacheStats := &runner.CacheStats{}
	opts := runner.RunOpts{
		Rebuild:        opt.GetBool("F"),
		Recursive:      opt.GetBool("r"),
		Clean:          opt.GetBool("c"),
		Verbose:        opt.GetBool("v"),
		Jobs:           opt.GetInt("j"),
		KeepGoing:      opt.GetBool("k"),
		DryRun:         opt.GetBool("n"),
		RefreshCACerts: opt.GetBool("refresh-cacerts"),
		BuildDir:       opt.GetString("output"),
		NoColor:        opt.GetBool("no-color") || opt.GetBool("no-colour"),
		CacheRemote:    cacheRemote,
		CacheStats:     cacheStats,
		Results:        results,
	}

	names := opt.GetPosStringSlice("targets")
	if len(names) == 0 {
		names = []string{"build"}
	}

	if opts.Recursive {
		if err := runner.RunRecursive(".", names[0], opts); err != nil {
			return fmt.Errorf("recursive build: %w", err)
		}
		results.Print()
		if opt.GetBool("cache-stats") {
			cacheStats.Print()
		}
		return nil
	}

	fiatPath, ok := fiat.FindFiat(opt.GetString("file"))
	if !ok {
		return fmt.Errorf("no fiat file found or file inaccessible")
	}

	file, err := fiat.Parse(fiatPath)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", fiatPath, err)
	}
	if bd := opt.GetString("output"); bd != "" {
		InjectBuildDir(file, bd)
	}
	if err := targets.Apply(file); err != nil {
		return fmt.Errorf("applying defaults to %s: %w", fiatPath, err)
	}

	if opt.GetBool("w") {
		runner.RunWatch(file, names[0], opts)
		return nil
	}

	var errCount int
	for _, name := range names {
		if err := runner.RunTarget(file, name, opts); err != nil {
			fx.Fprint(os.Stderr, "{red}Error: {}{@}\n", err)
			errCount++
			if !opts.KeepGoing {
				break
			}
		}
	}
	results.Print()
	if opt.GetBool("cache-stats") {
		cacheStats.Print()
	}
	if errCount > 0 {
		return fmt.Errorf("some targets failed")
	}
	return nil
}
