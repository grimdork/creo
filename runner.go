package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
		for _, t := range f.Targets {
			if err := runTargetWithDeps(f, t.Name, opts, visited, done); err != nil {
				return err
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

	vars := make(map[string]*Var)
	for k, v := range f.Vars {
		vars[k] = v
	}
	for _, v := range t.Vars {
		vars[v.Name] = v
	}

	if !opts.Clean {
		for _, pattern := range t.Tmp {
			expanded := expandVars(pattern, f.Vars, t.Vars)
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
			binPath := expandVars(t.Bin, f.Vars, t.Vars)
			if _, err := os.Stat(binPath); err == nil {
				if err := os.Remove(binPath); err == nil {
					fmt.Printf("  Removed %s\n", binPath)
				}
			}
		}
		done[name] = true
		return nil
	}

	needsRun := true
	if t.Bin != "" && t.Sources != "" {
		binPath := expandVars(t.Bin, f.Vars, t.Vars)
		binStat, err := os.Stat(binPath)
		if err == nil {
			binMod := binStat.ModTime()
			needsRun = false
			srcPatterns := strings.Fields(expandVars(t.Sources, f.Vars, t.Vars))
			for _, pat := range srcPatterns {
				files := globFiles(expandVars(pat, f.Vars, t.Vars), dir)
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
			binPath := expandVars(t.Bin, f.Vars, t.Vars)
			os.Remove(binPath)
		}

		start := time.Now()
		for _, cmd := range t.Cmds {
			expanded := expandVars(cmd, f.Vars, t.Vars)
			if opts.Verbose {
				fmt.Printf("  Running: %s\n", expanded)
			}
			if err := execCmd(expanded, dir); err != nil {
				return fmt.Errorf("command failed: %w", err)
			}
		}
		if opts.Verbose {
			fmt.Printf("  Done in %v\n", time.Since(start))
		}

		for _, pattern := range t.Tmp {
			expanded := expandVars(pattern, f.Vars, t.Vars)
			matches := globFiles(expanded, dir)
			for _, m := range matches {
				if err := os.Remove(m); err == nil && opts.Verbose {
					fmt.Printf("  Cleaned %s\n", m)
				}
			}
		}
	} else if opts.Verbose {
		fmt.Printf("  %s is up to date\n", name)
	}

	done[name] = true
	return nil
}

func execCmd(cmd, dir string) error {
	c := exec.Command("sh", "-c", cmd)
	c.Dir = dir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
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

		fiatPath, ok := findFiatInDir(path)
		if !ok {
			return nil
		}

		file, err := parseFiat(fiatPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", fiatPath, err)
			return nil
		}

		fmt.Printf("Entering %s\n", path)
		if err := runTarget(file, "build", opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error in %s: %v\n", path, err)
		}
		return nil
	})
}
