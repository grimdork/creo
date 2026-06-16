package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/grimdork/climate/arg"
	"github.com/grimdork/climate/fx"
	"github.com/grimdork/creo/internal/cli"
	"github.com/grimdork/creo/internal/runner"
)

func main() {
	opt := arg.New("creo", "A make-like build tool")
	opt.SetDefaultHelp(true)
	opt.SetFlag(arg.GroupDefault, "i", "init", "Initialise project with base files")
	opt.SetOption(arg.GroupDefault, "f", "file", "Alternative fiat file path", "", false, arg.VarString, nil)
	opt.SetFlag(arg.GroupDefault, "F", "force", "Force rebuild")
	opt.SetFlag(arg.GroupDefault, "r", "recursive", "Recurse into subdirectories")
	opt.SetFlag(arg.GroupDefault, "c", "clean", "Remove target binaries")
	opt.SetFlag(arg.GroupDefault, "v", "verbose", "Verbose diagnostic output")
	opt.SetFlag(arg.GroupDefault, "l", "list", "List available targets")
	opt.SetFlag(arg.GroupDefault, "w", "watch", "Watch sources and rebuild on change")
	opt.SetFlag(arg.GroupDefault, "k", "keep-going", "Continue despite errors")
	opt.SetFlag(arg.GroupDefault, "n", "dry-run", "Print commands without running them")
	opt.SetOption(arg.GroupDefault, "j", "jobs", "Parallel jobs (default: number of CPUs)", 0, false, arg.VarInt, nil)
	opt.SetFlag(arg.GroupDefault, "", "refresh-cacerts", "Re-download cached CA certificates")
	opt.SetFlag(arg.GroupDefault, "", "clean-cache", "Remove cached build artifacts")
	opt.SetFlag(arg.GroupDefault, "", "version", "Print version and exit")
	opt.SetFlag(arg.GroupDefault, "L", "login", "Store registry credentials in Docker config")
	opt.SetOption(arg.GroupDefault, "I", "inspect", "Inspect a remote image", "", false, arg.VarString, nil)
	opt.SetFlag(arg.GroupDefault, "", "completion", "Print shell completion script")
	opt.SetOption(arg.GroupDefault, "", "graph", "Show dependency graph (tree|dot|svg)", "", false, arg.VarString, nil)
	opt.SetFlag(arg.GroupDefault, "", "status", "Check cache state when showing graph")
	opt.SetFlag(arg.GroupDefault, "g", "git", "Initialise a git repository and commit")
	opt.SetOption(arg.GroupDefault, "o", "output", "Build output directory", "", false, arg.VarString, nil)
	opt.SetOption(arg.GroupDefault, "T", "template", "Project template name (use with -i)", "", false, arg.VarString, nil)
	opt.SetOption(arg.GroupDefault, "", "save-template", "Extract embedded template to user dir (lang/name)", "", false, arg.VarString, nil)
	opt.SetFlag(arg.GroupDefault, "", "list-templates", "List available project templates")
	opt.SetFlag(arg.GroupDefault, "", "no-color", "Disable coloured terminal output")
	opt.SetFlag(arg.GroupDefault, "", "no-colour", "Disable coloured terminal output")
	opt.SetOption(arg.GroupDefault, "", "cache-remote", "SSH remote cache URL (user@host:path)", "", false, arg.VarString, nil)
	opt.SetFlag(arg.GroupDefault, "", "cache-stats", "Print L1/L2 cache hit/miss statistics")
	opt.SetPositional("targets", "Targets to run or clean", nil, false, arg.VarStringSlice)

	if err := opt.Parse(os.Args[1:]); err != nil {
		if !errors.Is(err, arg.ErrNonFatal) {
			fail(err)
		}
	}

	switch {
	case opt.GetBool("version"):
		cli.RunVersion(version)
	case opt.GetBool("clean-cache"):
		if err := runner.CleanCache("."); err != nil {
			fail(err)
		}
		fx.Println("{success}Cache cleaned{@}")
	case opt.GetBool("completion"):
		fmt.Print(cli.GenerateCompletion(opt))
	case opt.GetBool("login"):
		if err := cli.RunLogin(); err != nil {
			fail(err)
		}
	case opt.GetString("inspect") != "":
		ref := opt.GetString("inspect")
		if err := cli.RunInspect(ref); err != nil {
			fail(err)
		}
	case opt.GetString("save-template") != "":
		if err := cli.RunSaveTemplate(opt.GetString("save-template"), opt.GetBool("F"), opt.GetBool("v")); err != nil {
			fail(err)
		}
	case opt.GetBool("list-templates"):
		lang := ""
		if names := opt.GetPosStringSlice("targets"); len(names) > 0 {
			lang = names[0]
		}
		if err := cli.RunListTemplates(lang); err != nil {
			fail(err)
		}
	case opt.GetBool("i"):
		if err := cli.RunInit(opt.GetPosStringSlice("targets"), opt.GetString("template"), opt.GetBool("F"), opt.GetBool("v")); err != nil {
			fail(err)
		}
		if opt.GetBool("g") {
			if err := cli.RunGitInit(opt.GetBool("v")); err != nil {
				fail(err)
			}
		}
	case opt.GetBool("g"):
		if err := cli.RunGitInit(opt.GetBool("v")); err != nil {
			fail(err)
		}
	case opt.GetBool("l"):
		if err := cli.RunList(opt.GetString("file")); err != nil {
			fail(err)
		}
	case opt.GetString("graph") != "":
		if err := cli.RunGraph(opt); err != nil {
			fail(err)
		}
	default:
		if err := cli.RunBuild(opt); err != nil {
			fail(err)
		}
	}
}
