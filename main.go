package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/grimdork/climate/arg"
	"github.com/grimdork/creo/internal/lang"
	"github.com/grimdork/creo/internal/runner"
)

func main() {
	opt := arg.New("creo", "A make-like build tool")
	opt.SetDefaultHelp(true)
	opt.SetFlag(arg.GroupDefault, "i", "init", "Initialise project with base files")
	opt.SetFlag(arg.GroupDefault, "f", "force", "Force rebuild")
	opt.SetFlag(arg.GroupDefault, "r", "recursive", "Recurse into subdirectories")
	opt.SetFlag(arg.GroupDefault, "c", "clean", "Remove target binaries")
	opt.SetFlag(arg.GroupDefault, "v", "verbose", "Verbose diagnostic output")
	opt.SetPositional("targets", "Targets to run or clean", nil, false, arg.VarStringSlice)

	err := opt.Parse(os.Args[1:])
	if err != nil {
		if !errors.Is(err, arg.ErrNonFatal) {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	if opt.GetBool("i") {
		initProject(opt.GetBool("f"), opt.GetBool("v"))
		return
	}

	opts := runner.RunOpts{
		Rebuild:   opt.GetBool("f"),
		Recursive: opt.GetBool("r"),
		Clean:     opt.GetBool("c"),
		Verbose:   opt.GetBool("v"),
	}

	targets := opt.GetPosStringSlice("targets")
	if len(targets) == 0 {
		targets = []string{"build"}
	}

	if opts.Recursive {
		runner.RunRecursive(".", opts)
		return
	}

	fiatPath, ok := findFiat()
	if !ok {
		os.Exit(1)
	}

	file, err := lang.ParseFiat(fiatPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", fiatPath, err)
		os.Exit(1)
	}
	lang.Apply(file)

	for _, name := range targets {
		if err := runner.RunTarget(file, name, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}
