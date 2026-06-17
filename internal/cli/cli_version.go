package cli

import "github.com/grimdork/climate/fx"

// RunVersion prints the creo version string.
func RunVersion(ver string) {
	if ver == "" {
		fx.Println("{bold}creo (dev){@}")
	} else {
		fx.Println("{bold}creo {}{@}", ver)
	}
}
