package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/grimdork/climate/arg"
	"github.com/grimdork/creo/internal/lang"
	"github.com/grimdork/creo/internal/runner"
)

func listTargets() {
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

	fmt.Println("Available targets:")
	for _, t := range file.Targets {
		ln := t.Language
		if ln == "" {
			ln = "-"
		}
		if t.Desc != "" {
			desc := lang.ExpandWithTarget(t.Desc, file.Vars, t)
			fmt.Printf("  %-15s (%s)   %s\n", t.Name, ln, desc)
		} else {
			fmt.Printf("  %-15s (%s)\n", t.Name, ln)
		}
	}
}

var version string

func main() {
	opt := arg.New("creo", "A make-like build tool")
	opt.SetDefaultHelp(true)
	opt.SetFlag(arg.GroupDefault, "i", "init", "Initialise project with base files")
	opt.SetFlag(arg.GroupDefault, "f", "force", "Force rebuild")
	opt.SetFlag(arg.GroupDefault, "r", "recursive", "Recurse into subdirectories")
	opt.SetFlag(arg.GroupDefault, "c", "clean", "Remove target binaries")
	opt.SetFlag(arg.GroupDefault, "v", "verbose", "Verbose diagnostic output")
	opt.SetFlag(arg.GroupDefault, "l", "list", "List available targets")
	opt.SetFlag(arg.GroupDefault, "w", "watch", "Watch sources and rebuild on change")
	opt.SetFlag(arg.GroupDefault, "k", "keep-going", "Continue despite errors")
	opt.SetFlag(arg.GroupDefault, "n", "dry-run", "Print commands without running them")
	opt.SetOption(arg.GroupDefault, "j", "jobs", "Parallel jobs (default: number of CPUs)", 0, false, arg.VarInt, nil)
	opt.SetFlag(arg.GroupDefault, "", "version", "Print version and exit")
	opt.SetPositional("targets", "Targets to run or clean", nil, false, arg.VarStringSlice)

	err := opt.Parse(os.Args[1:])
	if err != nil {
		if !errors.Is(err, arg.ErrNonFatal) {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	if opt.GetBool("version") {
		if version == "" {
			fmt.Println("creo (dev)")
		} else {
			fmt.Println("creo " + version)
		}
		return
	}

	if opt.GetBool("i") {
		langName, ver := "", ""
		targets := opt.GetPosStringSlice("targets")
		if len(targets) > 0 {
			spec := targets[0]
			if idx := strings.IndexByte(spec, ':'); idx >= 0 {
				langName, ver = spec[:idx], spec[idx+1:]
			} else {
				langName = spec
			}
		}
		initProject(langName, ver, opt.GetBool("f"), opt.GetBool("v"))
		return
	}

	opts := runner.RunOpts{
		Rebuild:   opt.GetBool("f"),
		Recursive: opt.GetBool("r"),
		Clean:     opt.GetBool("c"),
		Verbose:   opt.GetBool("v"),
		Jobs:      opt.GetInt("j"),
		KeepGoing: opt.GetBool("k"),
		DryRun:    opt.GetBool("n"),
	}

	if opt.GetBool("l") {
		listTargets()
		return
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

	if opt.GetBool("w") {
		runner.RunWatch(file, targets[0], opts)
		return
	}

	var errCount int
	for _, name := range targets {
		if err := runner.RunTarget(file, name, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			errCount++
			if !opts.KeepGoing {
				break
			}
		}
	}
	if errCount > 0 {
		os.Exit(1)
	}
}
