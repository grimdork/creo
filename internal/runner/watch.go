package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/grimdork/creo/internal/fiat"
	"github.com/grimdork/creo/internal/lang"
)

func RunWatch(f *fiat.File, name string, opts RunOpts) {
	dir := filepath.Dir(f.Path())

	t := fiat.FindTarget(f, name)
	if t == nil {
		fmt.Fprintf(os.Stderr, "Error: target %q not found\n", name)
		return
	}

	if t.IsVirtual || t.Sources == "" {
		fmt.Fprintf(os.Stderr, "Error: target %q has no sources to watch\n", name)
		return
	}

	if err := RunTarget(f, name, opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	fmt.Printf("Watching target %q for changes...\n", name)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	prevMods := make(map[string]time.Time)
	collectSources(t, f, dir, prevMods)

	for range ticker.C {
		curFiat, err := fiat.Parse(f.Path())
		if err != nil {
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "  Re-parse error: %v\n", err)
			}
			continue
		}
		if err := lang.Apply(curFiat); err != nil {
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "  Apply error: %v\n", err)
			}
			continue
		}

		curT := fiat.FindTarget(curFiat, name)
		if curT == nil {
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "  Target %q no longer exists\n", name)
			}
			continue
		}

		currentMods := make(map[string]time.Time)
		collectSources(curT, curFiat, dir, currentMods)

		changed := false
		if len(currentMods) != len(prevMods) {
			changed = true
		} else {
			for f, m := range currentMods {
				if prev, ok := prevMods[f]; !ok || !m.Equal(prev) {
					changed = true
					break
				}
			}
		}

		if changed {
			fmt.Println("\n  Change detected, rebuilding...")
			if err := RunTarget(curFiat, name, opts); err != nil {
				fmt.Fprintf(os.Stderr, "  Error: %v\n", err)
			}
			prevMods = currentMods
		}
	}
}

func collectSources(t *fiat.Target, f *fiat.File, dir string, mods map[string]time.Time) {
	for _, p := range collectFilePaths(t, f, dir) {
		if si, err := os.Stat(p); err == nil {
			mods[p] = si.ModTime()
		}
	}
}
