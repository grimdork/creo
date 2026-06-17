package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/grimdork/climate/fx"
	"github.com/grimdork/creo/internal/fiat"
	"github.com/grimdork/creo/internal/targets"
	"github.com/grimdork/creo/internal/util"
)

func runCombo(bt *buildTask) {
	bt.comboEnv = os.Environ()
	bt.activeArch = ensureArch(bt.c.arch)
	bt.activeOS = ensureOS(bt.c.osval)
	bt.comboEnv = append(bt.comboEnv, targets.CrossEnv(bt.t.Language, bt.c.arch, bt.c.osval)...)

	bt.comboVars = baseComboVars(bt.f, bt.t, bt.activeArch, bt.activeOS, bt.outputs)

	if !bt.opts.Rebuild && !bt.t.IsVirtual && bt.c.bin != "" && (bt.t.Sources != "" || bt.t.OCI != nil) {
		if _, err := os.Stat(bt.c.bin); err == nil {
			cached := bt.t.Sources == ""
			if bt.t.Sources != "" {
				comboKey := bt.name + "_" + bt.activeArch + "_" + bt.activeOS
				cached = checkCache(bt.dir, comboKey, bt.sources, bt.t.Cmds)
				if bt.opts.CacheStats != nil {
					if cached {
						bt.opts.CacheStats.L1Hit()
					} else {
						bt.opts.CacheStats.L1Miss()
					}
				}
			}
			if cached {
				if bt.opts.Verbose {
					fx.Println(`  {warning}{} up to date (cached){@}`, bt.c.bin)
				}
				bt.outputs.Store(bt.name, bt.activeArch, bt.activeOS, bt.c.bin)
				return
			}
		}
		if bt.opts.CacheRemote != "" && bt.t.Sources != "" {
			comboKey := bt.name + "_" + bt.activeArch + "_" + bt.activeOS
			hash, ok := tryRemoteCache(bt.opts.CacheRemote, comboKey, bt.sources, bt.t.Cmds)
			if bt.opts.CacheStats != nil {
				if ok {
					bt.opts.CacheStats.L2Hit()
				} else {
					bt.opts.CacheStats.L2Miss()
				}
			}
			if ok {
				if pullAndSave(bt.opts.CacheRemote, hash, comboKey, bt.c.bin) {
					if err := writeCache(bt.dir, comboKey, bt.sources, bt.t.Cmds); err != nil && bt.opts.Verbose {
						fx.Fprint(os.Stderr, "  {red}{}: cache write: {}{@}\n", comboKey, err)
					}
					bt.outputs.Store(bt.name, bt.activeArch, bt.activeOS, bt.c.bin)
					if bt.opts.Verbose {
						fx.Println(`  {warning}{} up to date (remote cache){@}`, bt.c.bin)
					}
					return
				}
			}
		}
	}

	if bt.t.Bin != "" {
		bt.comboVars["bin"] = &fiat.Var{Name: "bin", Value: bt.c.bin}
		if !bt.opts.DryRun && bt.opts.Rebuild && len(bt.t.Cmds) > 0 {
			if err := os.Remove(bt.c.bin); err != nil && !os.IsNotExist(err) && bt.opts.Verbose {
				fx.Fprint(os.Stderr, "  {red}{}: remove binary {:q}: {}{@}\n", bt.name, bt.c.bin, err)
			}
		}
	}
	if bt.t.Sources != "" {
		bt.comboVars["sources"] = &fiat.Var{Name: "sources", Value: fiat.Expand(bt.t.Sources, bt.comboVars, 0)}
	}

	if len(bt.t.Cmds) > 0 && (bt.opts.DryRun || bt.opts.Verbose) {
		if bt.t.Bin != "" {
			fx.Println(`  {cyan}Building {} ...{@}`, bt.c.bin)
		}
	}
	for _, cmd := range bt.t.Cmds {
		expanded := fiat.Expand(cmd, bt.comboVars, 0)
		if bt.opts.DryRun || bt.opts.Verbose {
			fx.Println(`  {cyan}{}{@}`, expanded)
		}
		if bt.opts.DryRun {
			continue
		}
		if err := execCmd(expanded, bt.dir, bt.comboEnv); err != nil {
			bt.errCh <- fmt.Errorf("%s: command failed: %w", bt.f.Path(), err)
			return
		}
	}

	if !bt.opts.DryRun && len(bt.t.Cmds) > 0 && bt.t.Bin != "" {
		if _, err := os.Stat(bt.c.bin); os.IsNotExist(err) {
			bt.errCh <- fmt.Errorf("%s: binary %q was not created by target %q", bt.f.Path(), bt.c.bin, bt.name)
			return
		}
		bt.outputs.Store(bt.name, bt.activeArch, bt.activeOS, bt.c.bin)
		if bt.t.Sources != "" && bt.multi {
			comboKey := bt.name + "_" + bt.activeArch + "_" + bt.activeOS
			if err := writeCache(bt.dir, comboKey, bt.sources, bt.t.Cmds); err != nil && bt.opts.Verbose {
				fx.Fprint(os.Stderr, "  {red}{}: cache write: {}{@}\n", bt.name, err)
			}
			if bt.opts.CacheRemote != "" {
				key, err := computeCacheKey(bt.sources, bt.t.Cmds)
				if err != nil {
					if bt.opts.Verbose {
						fx.Fprint(os.Stderr, "  {red}{}: cache key: {}{@}\n", bt.name, err)
					}
				} else {
					bt.wg.Add(1)
					go func() {
						defer bt.wg.Done()
						pushRemote(bt.opts.CacheRemote, key, comboKey, bt.c.bin, bt.dir, bt.sources, bt.t.Cmds)
					}()
				}
			}
		}
	}

	for _, inst := range bt.t.Install {
		expanded := fiat.Expand(inst, bt.comboVars, 0)
		expanded = os.ExpandEnv(expanded)
		src := bt.c.bin
		dest := expanded
		if idx := strings.IndexByte(expanded, ':'); idx >= 0 {
			src = expanded[:idx]
			dest = expanded[idx+1:]
		}
		if si, err := os.Stat(dest); err == nil && si.IsDir() {
			dest = filepath.Join(dest, filepath.Base(src))
		}
		fx.Println(`  {cyan}Installed {} -> {}{@}`, src, dest)
		if bt.opts.DryRun {
			continue
		}
		if err := util.CopyFile(src, dest); err != nil {
			bt.errCh <- fmt.Errorf("%s: install of %s: %w", bt.f.Path(), src, err)
			return
		}
	}

	if bt.t.Brew != nil && !bt.opts.DryRun {
		handleBrew(bt)
	}

	if bt.t.OCI != nil && !bt.opts.DryRun {
		handleOCI(bt)
	}
}
