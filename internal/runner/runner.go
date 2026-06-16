package runner

import (
	"fmt"
	"os"
	"os/exec"
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

type Outputs struct {
	mu sync.RWMutex
	m  map[string]string
}

func (o *Outputs) Store(target, arch, os, bin string) {
	o.mu.Lock()
	o.m[target+"/"+arch+"+"+os] = bin
	o.mu.Unlock()
}

func (o *Outputs) Load(target, arch, os string) string {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.m[target+"/"+arch+"+"+os]
}

func (o *Outputs) LoadAll(target string) []string {
	o.mu.RLock()
	defer o.mu.RUnlock()
	prefix := target + "/"
	var out []string
	for k := range o.m {
		if strings.HasPrefix(k, prefix) {
			out = append(out, k[len(prefix):])
		}
	}
	return out
}

type RunOpts struct {
	Rebuild        bool
	Clean          bool
	Recursive      bool
	Verbose        bool
	Jobs           int
	KeepGoing      bool
	DryRun         bool
	RefreshCACerts bool
	BuildDir       string
	NoColor        bool
	Results        *TargetResults
}

type combo struct {
	arch, osval, bin string
}

func RunTarget(f *fiat.File, name string, opts RunOpts) error {
	if opts.Results == nil {
		opts.Results = &TargetResults{}
	}
	return runTargetWithDeps(f, name, opts, map[string]bool{}, map[string]bool{}, &Outputs{m: make(map[string]string)})
}

func runTargetWithDeps(f *fiat.File, name string, opts RunOpts, visited, done map[string]bool, outputs *Outputs) error {
	if name == "all" {
		var allErrs []error
		report := func(err error) {
			if opts.KeepGoing {
				allErrs = append(allErrs, err)
			}
		}
		allVisited := map[string]bool{}
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

	if done[name] {
		return nil
	}

	if visited[name] {
		return fmt.Errorf("%s: circular dependency for target %q", f.Path(), name)
	}

	t := fiat.FindTarget(f, name)
	if t == nil {
		return fmt.Errorf("%s: target %q not found", f.Path(), name)
	}

	visited[name] = true
	dir := filepath.Dir(f.Path())

	if opts.Verbose {
		fx.Println(`{cyan}Target "{}"{@}`, name)
	}

	if !opts.Clean && !opts.DryRun {
		for _, pattern := range t.Tmp {
			expanded := fiat.ExpandWithTarget(pattern, f.Vars, t)
			matches := util.GlobFiles(expanded, dir)
			for _, m := range matches {
				if err := os.RemoveAll(m); err != nil {
					if opts.Verbose {
						fmt.Fprintf(os.Stderr, "  Failed to remove stale %s: %v\n", m, err)
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
					fmt.Fprintf(os.Stderr, "Warning: %s: %q built %s but %q only targets subset %s/%s\n",
						f.Path(), dep, key, t.Name, strings.Join(t.Arch, ","), strings.Join(t.OS, ","))
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
						fmt.Fprintf(os.Stderr, "  Failed to remove build directory %s: %v\n", bd, err)
					}
				} else if opts.Verbose {
					fx.Println(`  {success}Removed build directory {}{@}`, bd)
				}
			}
			for _, pattern := range t.Tmp {
				expanded := fiat.ExpandWithTarget(pattern, f.Vars, t)
				matches := util.GlobFiles(expanded, dir)
				for _, m := range matches {
					if err := os.RemoveAll(m); err != nil {
						if opts.Verbose {
							fmt.Fprintf(os.Stderr, "  %s\n", fmt.Errorf(errFailedClean, m, err))
						}
					} else if opts.Verbose {
						fx.Println(`  {cyan}Cleaned {}{@}`, m)
					}
				}
			}
		}
		done[name] = true
		return nil
	}

	needsRun := true
	var existsBinPath string
	var sources []string
	var buildStart time.Time
	if !t.IsVirtual && t.Bin != "" && t.Sources != "" {
		sources = collectFilePaths(t, f, dir)
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
			} else if t.Sources != "" && !checkCache(dir, name, sources, t.Cmds) {
				needsRun = true
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
			done[name] = true
			return nil
		}

		numJobs := opts.Jobs
		if numJobs <= 0 {
			numJobs = runtime.NumCPU()
		}

		var wg sync.WaitGroup
		sem := make(chan struct{}, numJobs)
		errCh := make(chan error, len(combos))

		for _, c := range combos {
			sem <- struct{}{}
			wg.Add(1)
			go func(c combo) {
				defer wg.Done()
				defer func() { <-sem }()

				comboEnv := os.Environ()
				activeArch := ensureArch(c.arch)
				activeOS := ensureOS(c.osval)
				comboEnv = append(comboEnv, targets.CrossEnv(t.Language, c.arch, c.osval)...)

				comboVars := baseComboVars(f, t, activeArch, activeOS, outputs)

				if !opts.Rebuild && !t.IsVirtual && c.bin != "" && (t.Sources != "" || t.OCI != nil) {
					if _, err := os.Stat(c.bin); err == nil {
						cached := t.Sources == ""
						if t.Sources != "" {
							comboKey := name + "_" + activeArch + "_" + activeOS
							cached = checkCache(dir, comboKey, sources, t.Cmds)
						}
						if cached {
							if opts.Verbose {
								fx.Println(`  {warning}{} up to date (cached){@}`, c.bin)
							}
							outputs.Store(name, activeArch, activeOS, c.bin)
							return
						}
					}
				}

				if t.Bin != "" {
					comboVars["bin"] = &fiat.Var{Name: "bin", Value: c.bin}
					if !opts.DryRun && opts.Rebuild && len(t.Cmds) > 0 {
						if err := os.Remove(c.bin); err != nil && !os.IsNotExist(err) && opts.Verbose {
							fmt.Fprintf(os.Stderr, "  Failed to remove %s: %v\n", c.bin, err)
						}
					}
				}
				if t.Sources != "" {
					comboVars["sources"] = &fiat.Var{Name: "sources", Value: fiat.Expand(t.Sources, comboVars, 0)}
				}

				if len(t.Cmds) > 0 && (opts.DryRun || opts.Verbose) {
					if t.Bin != "" {
						fx.Println(`  {cyan}Building {} ...{@}`, c.bin)
					}
				}
				for _, cmd := range t.Cmds {
					expanded := fiat.Expand(cmd, comboVars, 0)
					if opts.DryRun || opts.Verbose {
						fx.Println(`  {cyan}{}{@}`, expanded)
					}
					if opts.DryRun {
						continue
					}
					if err := execCmd(expanded, dir, comboEnv); err != nil {
						errCh <- fmt.Errorf("%s: command failed: %w", f.Path(), err)
						return
					}
				}

				if !opts.DryRun && len(t.Cmds) > 0 && t.Bin != "" {
					if _, err := os.Stat(c.bin); os.IsNotExist(err) {
						errCh <- fmt.Errorf("%s: binary %q was not created by target %q", f.Path(), c.bin, name)
						return
					}
					outputs.Store(name, activeArch, activeOS, c.bin)
					if t.Sources != "" && multi {
						comboKey := name + "_" + activeArch + "_" + activeOS
						if err := writeCache(dir, comboKey, sources, t.Cmds); err != nil && opts.Verbose {
							fmt.Fprintf(os.Stderr, "  %s\n", fmt.Errorf(errCacheWrite, err))
						}
					}
				}

				for _, inst := range t.Install {
					expanded := fiat.Expand(inst, comboVars, 0)
					expanded = os.ExpandEnv(expanded)
					src := c.bin
					dest := expanded
					if idx := strings.IndexByte(expanded, ':'); idx >= 0 {
						src = expanded[:idx]
						dest = expanded[idx+1:]
					}
					if si, err := os.Stat(dest); err == nil && si.IsDir() {
						dest = filepath.Join(dest, filepath.Base(src))
					}
					fx.Println(`  {cyan}Installed {} -> {}{@}`, src, dest)
					if opts.DryRun {
						continue
					}
					if err := util.CopyFile(src, dest); err != nil {
						errCh <- fmt.Errorf("%s: install of %s: %w", f.Path(), src, err)
						return
					}
				}

				// Homebrew formula generation
				if t.Brew != nil && !opts.DryRun {
					handleBrew(f, t, c, comboVars, dir, opts, name, outputs, errCh)
				}

				// OCI packaging: use $OUTPUT_<dep> to find binary
				if t.OCI != nil && !opts.DryRun {
					handleOCI(f, t, c, comboVars, comboEnv, dir, activeArch, activeOS, opts, name, outputs, errCh)
				}
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
				fmt.Fprintf(os.Stderr, "  %s\n", fmt.Errorf(errCacheWrite, err))
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
				matches := util.GlobFiles(expanded, dir)
				for _, m := range matches {
					if err := os.RemoveAll(m); err != nil {
						if opts.Verbose {
							fmt.Fprintf(os.Stderr, "  %s\n", fmt.Errorf(errFailedClean, m, err))
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

	done[name] = true
	return nil
}

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

func RunRecursive(dir string, targetName string, opts RunOpts) error {
	if opts.Results == nil {
		opts.Results = &TargetResults{}
	}
	var walkErr error
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
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
			walkErr = fmt.Errorf("parsing %s: %w", fiatPath, err)
			return nil
		}
		if opts.BuildDir != "" {
			file.Vars["BUILDDIR"] = &fiat.Var{Name: "BUILDDIR", Value: opts.BuildDir}
		}
		if err := targets.Apply(file); err != nil {
			walkErr = fmt.Errorf("applying defaults to %s: %w", fiatPath, err)
			return nil
		}

		if opts.Verbose {
			fx.Println(`{cyan}Entering {}{@}`, path)
		}
		if err := RunTarget(file, targetName, opts); err != nil {
			walkErr = fmt.Errorf("in %s: %w", path, err)
		}
		return nil
	})
	return walkErr
}

func baseComboVars(f *fiat.File, t *fiat.Target, activeArch, activeOS string, outputs *Outputs) map[string]*fiat.Var {
	comboVars := make(map[string]*fiat.Var)
	for k, v := range f.Vars {
		comboVars[k] = v
	}
	for _, v := range t.Vars {
		comboVars[v.Name] = v
	}
	comboVars["arch"] = &fiat.Var{Name: "arch", Value: activeArch}
	comboVars["os"] = &fiat.Var{Name: "os", Value: activeOS}
	comboVars["THIS"] = &fiat.Var{Name: "THIS", Value: t.Name}
	for _, dep := range t.Requires {
		if binPath := outputs.Load(dep, activeArch, activeOS); binPath != "" {
			comboVars["OUTPUT_"+dep] = &fiat.Var{Name: "OUTPUT_" + dep, Value: binPath}
		}
	}
	return comboVars
}

func archOrEmpty(a []string) []string {
	if len(a) == 0 {
		return []string{""}
	}
	return a
}

func osOrEmpty(o []string) []string {
	if len(o) == 0 {
		return []string{""}
	}
	return o
}

func ensureArch(a string) string {
	if a == "" {
		return runtime.GOARCH
	}
	return a
}

func ensureOS(o string) string {
	if o == "" {
		return runtime.GOOS
	}
	return o
}

func hasCombo(archs, oses []string, arch, os string) bool {
	for _, a := range archs {
		for _, o := range oses {
			if a == arch && o == os {
				return true
			}
		}
	}
	return false
}

func execCredHelper(helper, dir string) (user, pass string, err error) {
	parts := strings.Fields(helper)
	if len(parts) == 0 {
		return "", "", fmt.Errorf("empty credential helper")
	}
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = dir
	out, execErr := cmd.Output()
	if execErr != nil {
		return "", "", fmt.Errorf("%s: %w", helper, execErr)
	}
	line := strings.TrimSpace(string(out))
	idx := strings.IndexByte(line, ':')
	if idx < 0 {
		return "", line, nil
	}
	return line[:idx], line[idx+1:], nil
}
