package main

import (
	"fmt"
	"os"

	"github.com/grimdork/climate/fx"
)

var version string

func fail(err error) {
	fx.Fprint(os.Stderr, "{red}Error: {}{}{@}\n", err)
	os.Exit(1)
}

func failf(msg string, args ...interface{}) {
	fx.Fprint(os.Stderr, "{red}Error: {}{}{@}\n", fmt.Errorf(msg, args...))
	os.Exit(1)
}
