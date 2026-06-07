package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type RunOpts struct {
	Rebuild   bool
	Clean     bool
	Recursive bool
	Verbose   bool
}

func runTarget(f *FiatFile, name string, opts RunOpts) error {
	return runTargetWithDeps(f, name, opts, map[string]bool{}, map[string]bool{})
}

func runTargetWithDeps(f *FiatFile, name string, opts RunOpts, visited, done map[string]bool) error {
	if name == "all" {
		if bt := findTarget(f, "build"); bt != nil {
			if err := runTargetWithDeps(f, "build", opts, map[string]bool{}, done); err != nil {
				return err
			}
		}
		for _, t := range f.Targets {
			if t.Name != "build" {
				if err := runTargetWithDeps(f, t.Name, opts, map[string]bool{}, done); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if done[name] {
		return nil
	}

	if visited[name] {
		return fmt.Errorf("circular dependency detected for target %q", name)
	}

	t := findTarget(f, name)
	if t == nil {
		return fmt.Errorf("target %q not found in %s", name, f.Path)
	}

	visited[name] = true
	dir := filepath.Dir(f.Path)

	if !opts.Clean {
		for _, pattern := range t.Tmp {
			expanded := expandWithTarget(pattern, f.Vars, t)
			matches := globFiles(expanded, dir)
			for _, m := range matches {
				if err := os.Remove(m); err == nil && opts.Verbose {
					fmt.Printf("  Removed stale %s\n", m)
				}
			}
		}
	}

	for _, dep := range t.Requires {
		if findTarget(f, dep) == nil {
			return fmt.Errorf("dependency target %q not found for %q", dep, name)
		}
		if err := runTargetWithDeps(f, dep, opts, visited, done); err != nil {
			return err
		}
	}

	if opts.Clean {
		if t.Bin != "" {
			binPath := expandWithTarget(t.Bin, f.Vars, t)
			if _, err := os.Stat(binPath); err == nil {
				if err := os.Remove(binPath); err == nil && opts.Verbose {
					fmt.Printf("  Removed %s\n", binPath)
				}
			}
		}
		done[name] = true
		return nil
	}

	needsRun := true
	var existsBinPath string
	if !opts.Rebuild && t.Bin != "" && t.Sources != "" {
		existsBinPath = expandWithTarget(t.Bin, f.Vars, t)
		binStat, err := os.Stat(existsBinPath)
		if err == nil {
			binMod := binStat.ModTime()
			needsRun = false
			srcPatterns := strings.Fields(expandWithTarget(t.Sources, f.Vars, t))
			for _, pat := range srcPatterns {
				files := globFiles(expandWithTarget(pat, f.Vars, t), dir)
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

	if needsRun {
		if opts.Rebuild && t.Bin != "" {
			binPath := expandWithTarget(t.Bin, f.Vars, t)
			os.Remove(binPath)
		}

		start := time.Now()
		goEnv := os.Environ()
		if t.Arch != "" {
			goEnv = append(goEnv, "GOARCH="+expandWithTarget(t.Arch, f.Vars, t))
		}
		if t.OS != "" {
			goEnv = append(goEnv, "GOOS="+expandWithTarget(t.OS, f.Vars, t))
		}
		for _, cmd := range t.Cmds {
			expanded := expandWithTarget(cmd, f.Vars, t)
			if opts.Verbose {
				fmt.Printf("  Running: %s\n", expanded)
			}
			if err := execCmd(expanded, dir, goEnv); err != nil {
				return fmt.Errorf("command failed: %w", err)
			}
		}

		if t.Bin != "" {
			binPath := expandWithTarget(t.Bin, f.Vars, t)
			if _, err := os.Stat(binPath); os.IsNotExist(err) {
				return fmt.Errorf("binary %q was not created by target %q", binPath, name)
			}
		}

		if opts.Verbose {
			fmt.Printf("  Done in %v\n", time.Since(start))
		}

		for _, pattern := range t.Tmp {
			expanded := expandWithTarget(pattern, f.Vars, t)
			matches := globFiles(expanded, dir)
			for _, m := range matches {
				if err := os.Remove(m); err == nil && opts.Verbose {
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

func expandWithTarget(s string, global map[string]*Var, t *Target) string {
	vars := make(map[string]*Var)
	for k, v := range global {
		vars[k] = v
	}
	for _, v := range t.Vars {
		vars[v.Name] = v
	}
	if t.Bin != "" {
		vars["bin"] = &Var{Name: "bin", Value: expand(t.Bin, vars, 0)}
	}
	if t.Sources != "" {
		vars["sources"] = &Var{Name: "sources", Value: expand(t.Sources, vars, 0)}
	}
	arch := t.Arch
	if arch == "" {
		arch = runtime.GOARCH
	}
	osval := t.OS
	if osval == "" {
		osval = runtime.GOOS
	}
	vars["arch"] = &Var{Name: "arch", Value: arch}
	vars["os"] = &Var{Name: "os", Value: osval}
	return expand(s, vars, 0)
}

func execCmd(cmd, dir string, env []string) error {
	c := exec.Command("sh", "-c", cmd)
	c.Dir = dir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Env = env
	return c.Run()
}

func globFiles(pattern, dir string) []string {
	if strings.HasPrefix(pattern, "**") {
		ext := strings.TrimPrefix(pattern, "**")
		var files []string
		filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() && strings.HasSuffix(path, ext) {
				rel, _ := filepath.Rel(dir, path)
				files = append(files, filepath.Join(dir, rel))
			}
			return nil
		})
		return files
	}

	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return nil
	}
	return matches
}

func removeMatching(pattern, dir string) []string {
	matches := globFiles(pattern, dir)
	for _, m := range matches {
		os.Remove(m)
	}
	return matches
}

func runRecursive(dir string, opts RunOpts) {
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

		file, err := parseFiat(fiatPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", fiatPath, err)
			return nil
		}

		if opts.Verbose {
			fmt.Printf("Entering %s\n", path)
		}
		if err := runTarget(file, "build", opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error in %s: %v\n", path, err)
		}
		return nil
	})
}
