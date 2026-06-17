package cli

import (
	"github.com/grimdork/climate/fx"
	"github.com/grimdork/creo/internal/oci"
)

// RunLogin prompts for registry credentials and stores them in Docker config.
func RunLogin() error {
	if err := oci.Login(); err != nil {
		return err
	}
	fx.Println("{success}Credentials stored{@}")
	return nil
}
