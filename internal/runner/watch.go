package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/grimdork/creo/internal/lang"
)

func RunWatch(f *lang.FiatFile, name string, opts RunOpts) {
	dir := filepath.Dir(f.Path)

	t := lang.FindTarget(f, name)
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

	for range ticker.C {
		curFiat, err := lang.ParseFiat(f.Path)
		if err != nil {
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "  Re-parse error: %v\n", err)
			}
			continue
		}
		lang.Apply(curFiat)

		curT := lang.FindTarget(curFiat, name)
		if curT == nil {
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "  Target %q no longer exists\n", name)
			}
			continue
		}

		currentMods := make(map[string]time.Time)
		srcPatterns := strings.Fields(lang.ExpandWithTarget(curT.Sources, curFiat.Vars, curT))
		for _, pat := range srcPatterns {
			files := globFiles(lang.ExpandWithTarget(pat, curFiat.Vars, curT), dir)
			for _, sf := range files {
				si, err := os.Stat(sf)
				if err == nil {
					currentMods[sf] = si.ModTime()
				}
			}
		}

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
