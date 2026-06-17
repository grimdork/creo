package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/grimdork/climate/fx"
	"github.com/grimdork/creo/internal/fiat"
	"github.com/grimdork/creo/internal/targets"
	"github.com/grimdork/creo/internal/util"
)

func runCombo(f *fiat.File, t *fiat.Target, c combo, dir string, opts RunOpts, name string, outputs *Outputs, sources []string, multi bool, errCh chan<- error, wg *sync.WaitGroup) {
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
				if opts.CacheStats != nil {
					if cached {
						opts.CacheStats.L1Hit()
					} else {
						opts.CacheStats.L1Miss()
					}
				}
			}
			if cached {
				if opts.Verbose {
					fx.Println(`  {warning}{} up to date (cached){@}`, c.bin)
				}
				outputs.Store(name, activeArch, activeOS, c.bin)
				return
			}
		}
		if opts.CacheRemote != "" && t.Sources != "" {
			comboKey := name + "_" + activeArch + "_" + activeOS
			hash, ok := tryRemoteCache(opts.CacheRemote, comboKey, sources, t.Cmds)
			if opts.CacheStats != nil {
				if ok {
					opts.CacheStats.L2Hit()
				} else {
					opts.CacheStats.L2Miss()
				}
			}
			if ok {
				if pullAndSave(opts.CacheRemote, hash, comboKey, c.bin) {
					if err := writeCache(dir, comboKey, sources, t.Cmds); err != nil && opts.Verbose {
						fx.Fprint(os.Stderr, "  {red}{}: cache write: {}{@}\n", comboKey, err)
					}
					outputs.Store(name, activeArch, activeOS, c.bin)
					if opts.Verbose {
						fx.Println(`  {warning}{} up to date (remote cache){@}`, c.bin)
					}
					return
				}
			}
		}
	}

	if t.Bin != "" {
		comboVars["bin"] = &fiat.Var{Name: "bin", Value: c.bin}
		if !opts.DryRun && opts.Rebuild && len(t.Cmds) > 0 {
			if err := os.Remove(c.bin); err != nil && !os.IsNotExist(err) && opts.Verbose {
				fx.Fprint(os.Stderr, "  {red}{}: remove binary {:q}: {}{@}\n", name, c.bin, err)
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
				fx.Fprint(os.Stderr, "  {red}{}: cache write: {}{@}\n", name, err)
			}
			if opts.CacheRemote != "" {
				key, _ := computeCacheKey(sources, t.Cmds)
				wg.Add(1)
				go func() {
					defer wg.Done()
					pushRemote(opts.CacheRemote, key, comboKey, c.bin, dir, sources, t.Cmds)
				}()
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

	bt := &buildTask{
		f:          f,
		t:          t,
		c:          c,
		comboVars:  comboVars,
		comboEnv:   comboEnv,
		dir:        dir,
		activeArch: activeArch,
		activeOS:   activeOS,
		opts:       opts,
		name:       name,
		outputs:    outputs,
		errCh:      errCh,
	}
	if t.Brew != nil && !opts.DryRun {
		handleBrew(bt)
	}

	if t.OCI != nil && !opts.DryRun {
		handleOCI(bt)
	}
}
