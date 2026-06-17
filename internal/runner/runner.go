package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/grimdork/climate/fx"
	"github.com/grimdork/creo/internal/fiat"
	"github.com/grimdork/creo/internal/targets"
	"github.com/grimdork/creo/internal/util"
)

// RunTarget runs a named target from a parsed fiat configuration with the given options.
func RunTarget(f *fiat.File, name string, opts RunOpts) error {
	if opts.Results == nil {
		opts.Results = &TargetResults{}
	}
	return runTargetWithDeps(f, name, opts, util.NewSet[string](), util.NewSet[string](), &Outputs{m: make(map[string]string)})
}

func runTargetWithDeps(f *fiat.File, name string, opts RunOpts, visited, done util.Set[string], outputs *Outputs) error {
	if name == "all" {
		var allErrs []error
		report := func(err error) {
			if opts.KeepGoing {
				allErrs = append(allErrs, err)
			}
		}
		allVisited := util.NewSet[string]()
		if bt := fiat.FindTarget(f, "build"); bt != nil {
			if err := runTargetWithDeps(f, "build", opts, allVisited, done, outputs); err != nil {
				report(err)
			}
		}
		for _, t := range f.Targets {
			if t.Name != "build" {
				if err := runTargetWithDeps(f, t.Name, opts, allVisited, done, outputs); err != nil {
					report(err)
				}
			}
		}
		if len(allErrs) > 0 {
			if !opts.KeepGoing {
				return allErrs[0]
			}
			return fmt.Errorf("some targets failed")
		}
		return nil
	}

	if done.Has(name) {
		return nil
	}

	if visited.Has(name) {
		return fmt.Errorf("%s: circular dependency for target %q", f.Path(), name)
	}

	t := fiat.FindTarget(f, name)
	if t == nil {
		return fmt.Errorf("%s: target %q not found", f.Path(), name)
	}

	visited.Add(name)
	dir := filepath.Dir(f.Path())

	if opts.Verbose {
		fx.Println(`{cyan}Target "{}"{@}`, name)
	}

	if !opts.Clean && !opts.DryRun {
		for _, pattern := range t.Tmp {
			expanded := fiat.ExpandWithTarget(pattern, f.Vars, t)
			matches, err := util.GlobFiles(expanded, dir)
			if err != nil && opts.Verbose {
				fx.Fprint(os.Stderr, "  {warning}{}: pattern {:q}: {}{@}\n", name, pattern, err)
			}
			for _, m := range matches {
				if err := os.RemoveAll(m); err != nil {
					if opts.Verbose {
						fx.Fprint(os.Stderr, "  {red}{}: stale file {:q}: {}{@}\n", name, m, err)
					}
				} else if opts.Verbose {
					fx.Println(`  {cyan}Removed stale {}{@}`, m)
				}
			}
		}
	}

	if !opts.Clean {
		for _, dep := range t.Requires {
			if fiat.FindTarget(f, dep) == nil {
				return fmt.Errorf("%s: dependency %q not found for target %q", f.Path(), dep, name)
			}
			if err := runTargetWithDeps(f, dep, opts, visited, done, outputs); err != nil {
				return err
			}
		}
	}

	if t.OCI != nil && !opts.Clean && (len(t.Arch) > 0 || len(t.OS) > 0) {
		ociArchs := t.Arch
		if len(ociArchs) == 0 {
			ociArchs = []string{runtime.GOARCH}
		}
		ociOSs := t.OS
		if len(ociOSs) == 0 {
			ociOSs = []string{runtime.GOOS}
		}
		for _, dep := range t.Requires {
			for _, key := range outputs.LoadAll(dep) {
				parts := strings.SplitN(key, "+", 2)
				if len(parts) != 2 {
					continue
				}
				if !hasCombo(ociArchs, ociOSs, parts[0], parts[1]) {
					fx.Fprint(os.Stderr, "{warning}{}: dep {:q} produced {:q} but target {:q} restricts to {}/{}{@}\n",
						name, dep, key, t.Name, strings.Join(t.Arch, ","), strings.Join(t.OS, ","))
				}
			}
		}
	}

	if opts.Clean {
		if !opts.DryRun {
			if !t.IsVirtual {
				bd := targets.BuildDir(f)
				if err := os.RemoveAll(bd); err != nil {
					if opts.Verbose {
						fx.Fprint(os.Stderr, "  {red}{}: build dir {:q}: {}{@}\n", name, bd, err)
					}
				} else if opts.Verbose {
					fx.Println(`  {success}Removed build directory {}{@}`, bd)
				}
			}
			for _, pattern := range t.Tmp {
				expanded := fiat.ExpandWithTarget(pattern, f.Vars, t)
				matches, err := util.GlobFiles(expanded, dir)
				if err != nil && opts.Verbose {
					fx.Fprint(os.Stderr, "  {warning}{}: pattern {:q}: {}{@}\n", name, pattern, err)
				}
				for _, m := range matches {
					if err := os.RemoveAll(m); err != nil {
						if opts.Verbose {
							fx.Fprint(os.Stderr, "  {red}{}: clean {:q}: {}{@}\n", name, m, err)
						}
					} else if opts.Verbose {
						fx.Println(`  {cyan}Cleaned {}{@}`, m)
					}
				}
			}
		}
		done.Add(name)
			return nil
		}

		needsRun := true
	var existsBinPath string
	var sources []string
	var buildStart time.Time
	if !t.IsVirtual && t.Bin != "" && t.Sources != "" {
		var err error
		sources, err = collectFilePaths(t, f, dir)
		if err != nil {
			fx.Fprint(os.Stderr, "{warning}{}: source paths: {}{@}\n", name, err)
		}
	}
	archs := archOrEmpty(t.Arch)
	oses := osOrEmpty(t.OS)
	multi := len(archs) > 1 || len(oses) > 1

	if !t.IsVirtual && !multi && !opts.Rebuild && t.Bin != "" && (t.Sources != "" || t.OCI != nil) {
		existsBinPath = fiat.ExpandWithTarget(t.Bin, f.Vars, t)
		if _, err := os.Stat(existsBinPath); err == nil {
			needsRun = false
			if len(sources) == 0 && t.Sources != "" {
				needsRun = true
			} else if t.Sources != "" {
				l1ok := checkCache(dir, name, sources, t.Cmds)
				if opts.CacheStats != nil {
					if l1ok {
						opts.CacheStats.L1Hit()
					} else {
						opts.CacheStats.L1Miss()
					}
				}
				if !l1ok {
					needsRun = true
				}
			}
		}
		if needsRun && opts.CacheRemote != "" && t.Sources != "" {
			hash, ok := tryRemoteCache(opts.CacheRemote, name, sources, t.Cmds)
			if opts.CacheStats != nil {
				if ok {
					opts.CacheStats.L2Hit()
				} else {
					opts.CacheStats.L2Miss()
				}
			}
			if ok {
				if pullAndSave(opts.CacheRemote, hash, name, existsBinPath) {
					if err := writeCache(dir, name, sources, t.Cmds); err != nil && opts.Verbose {
						fx.Fprint(os.Stderr, "  {red}{}: cache write: {}{@}\n", name, err)
					}
					needsRun = false
				}
			}
		}
	}

	if len(t.Install) > 0 {
		needsRun = true
	}

	if needsRun || multi {
		buildStart = time.Now()

		var combos []combo
		for _, arch := range archs {
			for _, osval := range oses {
				activeArch := ensureArch(arch)
				activeOS := ensureOS(osval)

				comboVars := baseComboVars(f, t, activeArch, activeOS, outputs)

				bp := ""
				if t.Bin != "" {
					bp = fiat.Expand(t.Bin, comboVars, 0)
					if strings.Contains(bp, "$bin") {
						bp = strings.ReplaceAll(bp, "$bin", "")
					}
				}

				combos = append(combos, combo{arch, osval, bp})
			}
		}

		allExist := !t.IsVirtual && !opts.Rebuild && t.Bin != "" && len(t.Install) == 0
		if allExist && t.Sources != "" {
			allExist = false
			existsBinPath = fiat.ExpandWithTarget(t.Bin, f.Vars, t)
			if _, err := os.Stat(existsBinPath); err == nil {
				if len(sources) > 0 && checkCache(dir, name, sources, t.Cmds) {
					allExist = true
				}
			}
		} else if allExist && t.OCI != nil {
			allExist = false
			existsBinPath = fiat.ExpandWithTarget(t.Bin, f.Vars, t)
			if _, err := os.Stat(existsBinPath); err == nil {
				allExist = true
			}
		}
		if allExist {
			if t.Bin != "" {
				for _, arch := range archs {
					for _, osval := range oses {
						a := ensureArch(arch)
						o := ensureOS(osval)
						cv := baseComboVars(f, t, a, o, outputs)
						bp := fiat.Expand(t.Bin, cv, 0)
						outputs.Store(name, a, o, bp)
					}
				}
			}
			if t.Sources != "" {
				fx.Println(`{warning}Target "{}": up to date (cached){@}`, name)
			} else {
				fx.Println(`{warning}Target "{}": already exists. Skipping.{@}`, name)
			}
			if opts.Results != nil {
				opts.Results.Add(t.Name, "SKIPPED", 0, nil)
			}
			done.Add(name)
			return nil
		}

		numJobs := opts.Jobs
		if numJobs <= 0 {
			numJobs = runtime.NumCPU()
		}

		var wg sync.WaitGroup
		sem := make(chan struct{}, numJobs)
		errCh := make(chan error, 2*len(combos))

		for _, c := range combos {
			sem <- struct{}{}
			wg.Add(1)
			go func(c combo) {
				bt := &buildTask{
					f:       f,
					t:       t,
					c:       c,
					dir:     dir,
					opts:    opts,
					name:    name,
					outputs: outputs,
					sources: sources,
					multi:   multi,
					errCh:   errCh,
					wg:      &wg,
				}
				defer wg.Done()
				defer func() { <-sem }()
				runCombo(bt)
			}(c)
		}

		wg.Wait()
		close(sem)
		close(errCh)
		var errs []error
		for err := range errCh {
			errs = append(errs, err)
		}
		if len(errs) > 0 {
			if opts.Results != nil {
				opts.Results.Add(t.Name, "FAILED", time.Since(buildStart), errs[0])
			}
			if !opts.KeepGoing {
				return errs[0]
			}
			return fmt.Errorf("some combos failed")
		}

		if needsRun && !opts.DryRun && !t.IsVirtual && !multi && t.Bin != "" && t.Sources != "" {
			if err := writeCache(dir, name, sources, t.Cmds); err != nil && opts.Verbose {
				fx.Fprint(os.Stderr, "  {red}{}: cache write: {}{@}\n", name, err)
			}
			if opts.CacheRemote != "" {
				key, err := computeCacheKey(sources, t.Cmds)
				if err != nil {
					if opts.Verbose {
						fx.Fprint(os.Stderr, "  {red}{}: cache key: {}{@}\n", name, err)
					}
				} else {
					pushRemote(opts.CacheRemote, key, name, existsBinPath, dir, sources, t.Cmds)
				}
			}
		}

		if opts.Verbose {
			fx.Println(`  {success}Done in {}{@}`, time.Since(buildStart))
		}
		if opts.Results != nil {
			opts.Results.Add(t.Name, "OK", time.Since(buildStart), nil)
		}

		if !opts.DryRun {
			for _, pattern := range t.Tmp {
				expanded := fiat.ExpandWithTarget(pattern, f.Vars, t)
				matches, err := util.GlobFiles(expanded, dir)
				if err != nil && opts.Verbose {
					fx.Fprint(os.Stderr, "  {warning}{}: pattern {:q}: {}{@}\n", name, pattern, err)
				}
				for _, m := range matches {
					if err := os.RemoveAll(m); err != nil {
						if opts.Verbose {
							fx.Fprint(os.Stderr, "  {red}{}: clean {:q}: {}{@}\n", name, m, err)
						}
					} else if opts.Verbose {
						fx.Println(`  {cyan}Cleaned {}{@}`, m)
					}
				}
			}
		}
	} else if t.Sources != "" {
		if t.Bin != "" {
			outputs.Store(name, runtime.GOARCH, runtime.GOOS, existsBinPath)
		}
		fx.Println(`{warning}Target "{}": up to date (cached){@}`, name)
		if opts.Results != nil {
			opts.Results.Add(t.Name, "SKIPPED", 0, nil)
		}
	} else {
		fx.Println(`{warning}Target "{}": binary "{}" already exists. Skipping.{@}`, name, existsBinPath)
		if opts.Results != nil {
			opts.Results.Add(t.Name, "SKIPPED", 0, nil)
		}
	}

	done.Add(name)
	return nil
}
