package targets

import (
	"os"

	"github.com/grimdork/climate/fx"
)

func tryWrite(path, content string, force, verbose bool, label string) error {
	if _, err := os.Stat(path); err == nil {
		if force {
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return err
			}
			if verbose {
				fx.Println("  {success}Replaced {}{@}", label)
			}
		} else {
			if verbose {
				fx.Println("  {warning}Skipped {} (already exists){@}", label)
			}
		}
	} else {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
		if verbose {
			fx.Println("  {success}Created {}{@}", label)
		}
	}
	return nil
}
