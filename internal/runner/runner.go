package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/grimdork/creo/internal/fiat"
	"github.com/grimdork/creo/internal/lang"
	"github.com/grimdork/creo/internal/oci"
)

type RunOpts struct {
	Rebuild   bool
	Clean     bool
	Recursive bool
	Verbose   bool
	Jobs      int
	KeepGoing bool
	DryRun    bool
}

func RunTarget(f *fiat.File, name string, opts RunOpts) error {
	return runTargetWithDeps(f, name, opts, map[string]bool{}, map[string]bool{}, map[string]string{})
}

func runTargetWithDeps(f *fiat.File, name string, opts RunOpts, visited, done map[string]bool, outputs map[string]string) error {
	if name == "all" {
		var allErrs []error
		report := func(err error) {
			if opts.KeepGoing {
				allErrs = append(allErrs, err)
			}
		}
		if bt := fiat.FindTarget(f, "build"); bt != nil {
			if err := runTargetWithDeps(f, "build", opts, map[string]bool{}, done, outputs); err != nil {
				report(err)
			}
		}
		for _, t := range f.Targets {
			if t.Name != "build" {
				if err := runTargetWithDeps(f, t.Name, opts, map[string]bool{}, done, outputs); err != nil {
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
		fmt.Printf("Target %q\n", name)
	}

	if !opts.Clean {
		for _, pattern := range t.Tmp {
			expanded := fiat.ExpandWithTarget(pattern, f.Vars, t)
			matches := globFiles(expanded, dir)
			for _, m := range matches {
				if err := os.Remove(m); err != nil {
					if opts.Verbose {
						fmt.Printf("  Failed to remove stale %s: %v\n", m, err)
					}
				} else if opts.Verbose {
					fmt.Printf("  Removed stale %s\n", m)
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
			dt := fiat.FindTarget(f, dep)
			if dt != nil && dt.Bin != "" {
				expandedBin := fiat.ExpandWithTarget(dt.Bin, f.Vars, dt)
				outputs[dep] = expandedBin
			}
		}
	}

	if opts.Clean {
		if !t.IsVirtual {
			archs := t.Arch
			if len(archs) == 0 {
				archs = []string{""}
			}
			oses := t.OS
			if len(oses) == 0 {
				oses = []string{""}
			}

			cleanCombos := func(cb func(arch, osval string, cv map[string]*fiat.Var)) {
				for _, arch := range archs {
					for _, osval := range oses {
						activeArch := arch
						activeOS := osval
						if activeArch == "" {
							activeArch = runtime.GOARCH
						}
						if activeOS == "" {
							activeOS = runtime.GOOS
						}

						cv := make(map[string]*fiat.Var)
						for k, v := range f.Vars {
							cv[k] = v
						}
						for _, v := range t.Vars {
							cv[v.Name] = v
						}
						cv["arch"] = &fiat.Var{Name: "arch", Value: activeArch}
						cv["os"] = &fiat.Var{Name: "os", Value: activeOS}
						cv["THIS"] = &fiat.Var{Name: "THIS", Value: t.Name}

						cb(activeArch, activeOS, cv)
					}
				}
			}

			cleanCombos(func(arch, osval string, cv map[string]*fiat.Var) {
				if t.Bin != "" {
					bp := fiat.Expand(t.Bin, cv, 0)
					cv["bin"] = &fiat.Var{Name: "bin", Value: bp}
					if len(t.Cmds) > 0 {
						if _, err := os.Stat(bp); err == nil {
							if err := os.Remove(bp); err != nil {
								if opts.Verbose {
									fmt.Printf("  Failed to remove %s: %v\n", bp, err)
								}
							} else if opts.Verbose {
								fmt.Printf("  Removed %s\n", bp)
							}
						}
					}
				}
				for _, inst := range t.Install {
					expanded := fiat.Expand(inst, cv, 0)
					expanded = os.ExpandEnv(expanded)
					src := ""
					dest := expanded
					if idx := strings.IndexByte(expanded, ':'); idx >= 0 {
						src = expanded[:idx]
						dest = expanded[idx+1:]
					}
					if si, err := os.Stat(dest); err == nil && si.IsDir() && src != "" {
						dest = filepath.Join(dest, filepath.Base(src))
					}
					if _, err := os.Stat(dest); err == nil {
						if err := os.Remove(dest); err != nil {
							if opts.Verbose {
								fmt.Printf("  Failed to remove installed %s: %v\n", dest, err)
							}
						} else if opts.Verbose {
							fmt.Printf("  Removed installed %s\n", dest)
						}
					}
				}
			})
		}
		done[name] = true
		return nil
	}

	needsRun := true
	var existsBinPath string
	archs := t.Arch
	if len(archs) == 0 {
		archs = []string{""}
	}
	oses := t.OS
	if len(oses) == 0 {
		oses = []string{""}
	}
	multi := len(archs) > 1 || len(oses) > 1

	if !t.IsVirtual && !multi && !opts.Rebuild && t.Bin != "" && t.Sources != "" {
		existsBinPath = fiat.ExpandWithTarget(t.Bin, f.Vars, t)
		binStat, err := os.Stat(existsBinPath)
		if err == nil {
			binMod := binStat.ModTime()
			needsRun = false
			srcPatterns := strings.Fields(fiat.ExpandWithTarget(t.Sources, f.Vars, t))
			for _, pat := range srcPatterns {
				files := globFiles(fiat.ExpandWithTarget(pat, f.Vars, t), dir)
				for _, sf := range files {
					sStat, sErr := os.Stat(sf)
					if sErr != nil || sStat.ModTime().After(binMod) {
						needsRun = true
						break
					}
				}
				if needsRun {
					break
				}
			}
		}
	}

	if len(t.Install) > 0 {
		needsRun = true
	}

	if needsRun || multi {
		start := time.Now()

		type combo struct {
			arch, osval, bin string
		}
		var combos []combo
		for _, arch := range archs {
			for _, osval := range oses {
				activeArch := arch
				activeOS := osval
				if activeArch == "" {
					activeArch = runtime.GOARCH
				}
				if activeOS == "" {
					activeOS = runtime.GOOS
				}

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

				for depName, depBin := range outputs {
					comboVars["OUTPUT_"+depName] = &fiat.Var{Name: "OUTPUT_" + depName, Value: depBin}
				}

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

		allExist := !t.IsVirtual && !opts.Rebuild && t.Bin != "" && len(t.Install) == 0 && t.OCI == nil
		if allExist {
			for _, c := range combos {
				if c.bin != "" {
					if _, err := os.Stat(c.bin); err != nil {
						allExist = false
						break
					}
				}
			}
		}
		if allExist {
			fmt.Printf("Target %q: binaries already exist. Skipping.\n", name)
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
				activeArch := c.arch
				activeOS := c.osval
				if activeArch == "" {
					activeArch = runtime.GOARCH
				}
				if activeOS == "" {
					activeOS = runtime.GOOS
				}
				comboEnv = append(comboEnv, lang.CrossEnv(t.Language, c.arch, c.osval)...)

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

				for depName, depBin := range outputs {
					comboVars["OUTPUT_"+depName] = &fiat.Var{Name: "OUTPUT_" + depName, Value: depBin}
				}

				if t.Bin != "" {
					comboVars["bin"] = &fiat.Var{Name: "bin", Value: c.bin}
					if !opts.DryRun && opts.Rebuild && len(t.Cmds) > 0 {
						if err := os.Remove(c.bin); err != nil && !os.IsNotExist(err) && opts.Verbose {
							fmt.Printf("  Failed to remove %s: %v\n", c.bin, err)
						}
					}
				}
				if t.Sources != "" {
					comboVars["sources"] = &fiat.Var{Name: "sources", Value: fiat.Expand(t.Sources, comboVars, 0)}
				}

				if len(t.Cmds) > 0 && (opts.DryRun || opts.Verbose) {
					if t.Bin != "" {
						fmt.Printf("  Building %s ...\n", c.bin)
					}
				}
				for _, cmd := range t.Cmds {
					expanded := fiat.Expand(cmd, comboVars, 0)
					if opts.DryRun || opts.Verbose {
						fmt.Println("  " + expanded)
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
					fmt.Printf("  Installed %s -> %s\n", src, dest)
					if opts.DryRun {
						continue
					}
					if err := copyFile(src, dest); err != nil {
						errCh <- fmt.Errorf("%s: install of %s: %w", f.Path(), src, err)
						return
					}
				}

			// OCI packaging: use $OUTPUT_<dep> to find binary
			if t.OCI != nil && !opts.DryRun {
				binSrc := ""
				for _, dep := range t.Requires {
					outVar := "OUTPUT_" + dep
					if v, ok := comboVars[outVar]; ok && v.Value != "" {
						binSrc = v.Value
						break
					}
				}
				if binSrc == "" {
					binSrc = c.bin
				}

				if binSrc != "" {
					absBin, err := filepath.Abs(binSrc)
					if err != nil {
						errCh <- fmt.Errorf("%s: resolving binary path: %w", f.Path(), err)
						return
					}
					if _, err := os.Stat(absBin); err != nil {
						errCh <- fmt.Errorf("%s: OCI binary %q not found: %w", f.Path(), absBin, err)
						return
					}
					binSrc = absBin
					binaryName := filepath.Base(binSrc)
					appDir := t.OCI.AppDir
					if appDir == "" {
						appDir = "/app"
					}

					img, err := oci.Build(oci.Config{
						Binary: binSrc,
						AppDir: appDir,
						Name:   binaryName,
					})
					if err != nil {
						errCh <- fmt.Errorf("%s: OCI build: %w", f.Path(), err)
						return
					}

					tarballPath := fiat.Expand(t.OCI.Tarball, comboVars, 0)
					if tarballPath != "" {
						tag := fiat.Expand(t.OCI.Tag, comboVars, 0)
						if tag == "" {
							tag = "latest"
						}
						if err := oci.WriteTarball(img, tarballPath, tag); err != nil {
							errCh <- fmt.Errorf("%s: OCI tarball: %w", f.Path(), err)
							return
						}
						fmt.Printf("  Wrote %s\n", tarballPath)
					}

					repo := fiat.Expand(t.OCI.Repo, comboVars, 0)
					if repo != "" {
						tag := fiat.Expand(t.OCI.Tag, comboVars, 0)
						user := os.ExpandEnv(fiat.Expand(t.OCI.User, comboVars, 0))
						pass := os.ExpandEnv(fiat.Expand(t.OCI.Pass, comboVars, 0))
						if err := oci.Push(img, oci.PushConfig{
							Repo: repo,
							Tag:  tag,
							User: user,
							Pass: pass,
						}); err != nil {
							errCh <- fmt.Errorf("%s: OCI push: %w", f.Path(), err)
							return
						}
						ref := repo
						if tag != "" {
							ref += ":" + tag
						}
						fmt.Printf("  Pushed %s\n", ref)
					}
				}
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
			if !opts.KeepGoing {
				return errs[0]
			}
			return fmt.Errorf("some combos failed")
		}

		if opts.Verbose {
			fmt.Printf("  Done in %v\n", time.Since(start))
		}

		for _, pattern := range t.Tmp {
			expanded := fiat.ExpandWithTarget(pattern, f.Vars, t)
			matches := globFiles(expanded, dir)
			for _, m := range matches {
				if err := os.Remove(m); err != nil {
					if opts.Verbose {
						fmt.Printf("  Failed to clean %s: %v\n", m, err)
					}
				} else if opts.Verbose {
					fmt.Printf("  Cleaned %s\n", m)
				}
			}
		}
	} else {
		fmt.Printf("Target %q: binary %q already exists. Skipping.\n", name, existsBinPath)
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
			fmt.Printf("  Skipped %s (multiple .fiat files)\n", dir)
		}
	}
	return "", false
}

func RunRecursive(dir string, targetName string, opts RunOpts) {
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
			fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", fiatPath, err)
			return nil
		}
		if err := lang.Apply(file); err != nil {
			fmt.Fprintf(os.Stderr, "Error applying defaults to %s: %v\n", fiatPath, err)
			return nil
		}

		if opts.Verbose {
			fmt.Printf("Entering %s\n", path)
		}
		if err := RunTarget(file, targetName, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error in %s: %v\n", path, err)
		}
		return nil
	})
}
