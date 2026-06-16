package main

import (
	"os"

	"github.com/grimdork/climate/fx"
)

var version string

func fail(err error) {
	fx.Fprint(os.Stderr, "{red}Error: {}{}{@}\n", err)
	os.Exit(1)
}
