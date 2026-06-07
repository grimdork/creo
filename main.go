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
		lang := t.Language
		if lang == "" {
			lang = "-"
		}
		if t.Desc != "" {
			fmt.Printf("  %-15s (%s)   %s\n", t.Name, lang, t.Desc)
		} else {
			fmt.Printf("  %-15s (%s)\n", t.Name, lang)
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

	for _, name := range targets {
		if err := runner.RunTarget(file, name, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}
