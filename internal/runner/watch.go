package runner

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/grimdork/climate/fx"
	"github.com/grimdork/creo/internal/fiat"
	"github.com/grimdork/creo/internal/targets"
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

	fx.Println(`{cyan}Watching target "{}" for changes...{@}`, name)

	prevHash := fileHash(t, f, dir)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		curFiat, err := fiat.Parse(f.Path())
		if err != nil {
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "  Re-parse error: %v\n", err)
			}
			continue
		}
		if err := targets.Apply(curFiat); err != nil {
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

		curHash := fileHash(curT, curFiat, dir)

		if curHash != prevHash {
			fx.Println("{warning}  Change detected, rebuilding...{@}")
			if err := RunTarget(curFiat, name, opts); err != nil {
				fmt.Fprintf(os.Stderr, "  Error: %v\n", err)
			}
			prevHash = curHash
		}
	}
}

func fileHash(t *fiat.Target, f *fiat.File, dir string) string {
	paths := collectFilePaths(t, f, dir)
	h := sha256.New()
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		h.Write(data)
	}
	return hex.EncodeToString(h.Sum(nil))
}
