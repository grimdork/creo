package targets

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/grimdork/climate/fx"
	"github.com/grimdork/creo/internal/fiat"
)

// InitRust scaffolds a Rust project with a Cargo.toml and fiat file.
func InitRust(dir string, force, verbose bool) ([]string, error) {
	file, err := ensureFiat(dir)
	if err != nil {
		return nil, err
	}

	if fiat.FindTarget(file, "build") == nil {
		bt := &fiat.Target{
			Name:     "build",
			Language: "rust",
			Desc:     "Build the Rust binary",
		}
		file.AddTarget(bt)
	}

	cargoPath := filepath.Join(dir, "Cargo.toml")
	if _, err := os.Stat(cargoPath); os.IsNotExist(err) {
		cmd := exec.Command("cargo", "init", "--name", filepath.Base(dir))
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf(errCargoInit, strings.TrimSpace(string(out)))
		}
		if verbose {
			fx.Println("  {success}Initialised Cargo project{@}")
		}
	} else if force {
		if err := os.Remove(cargoPath); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("removing Cargo.toml: %w", err)
		}
		cmd := exec.Command("cargo", "init", "--name", filepath.Base(dir))
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf(errCargoInit, strings.TrimSpace(string(out)))
		}
		if verbose {
			fx.Println("  {success}Reinitialised Cargo project{@}")
		}
	} else if verbose {
		fx.Println("  {warning}Skipped Cargo.toml (already exists){@}")
	}

	if err := file.Write(); err != nil {
		return nil, err
	}

	return []string{"/build", "/.creo", "/target"}, nil
}
