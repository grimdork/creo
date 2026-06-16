package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/grimdork/climate/arg"
	"github.com/grimdork/climate/fx"
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
	opt.SetFlag(arg.GroupDefault, "", "no-color", "Disable coloured terminal output")
	opt.SetFlag(arg.GroupDefault, "", "no-colour", "Disable coloured terminal output")
	opt.SetPositional("targets", "Targets to run or clean", nil, false, arg.VarStringSlice)

	if err := opt.Parse(os.Args[1:]); err != nil {
		if !errors.Is(err, arg.ErrNonFatal) {
			fail(err)
		}
	}

	switch {
	case opt.GetBool("version"):
		printVersion()
	case opt.GetBool("clean-cache"):
		if err := runner.CleanCache("."); err != nil {
			fail(err)
		}
		fx.Println("{success}Cache cleaned{@}")
	case opt.GetBool("completion"):
		fmt.Print(generateCompletion(opt))
	case opt.GetBool("login"):
		runLogin()
	case opt.GetString("inspect") != "":
		ref := opt.GetString("inspect")
		runInspect(ref)
	case opt.GetBool("i"):
		runInit(opt.GetPosStringSlice("targets"), opt.GetBool("F"), opt.GetBool("v"))
		if opt.GetBool("g") {
			runGitInit(opt.GetBool("v"))
		}
	case opt.GetBool("g"):
		runGitInit(opt.GetBool("v"))
	case opt.GetBool("l"):
		runList(opt.GetString("file"))
	case opt.GetString("graph") != "":
		runGraph(opt)
	default:
		runBuild(opt)
	}
}

func generateCompletion(opt *arg.Options) string {
	base, err := opt.Completions()
	if err != nil {
		return ""
	}

	funcStart := strings.Index(base, "\n_creo() {")
	if funcStart < 0 {
		return base
	}

	completeLine := strings.Index(base, "\ncomplete -F _creo")
	if completeLine < 0 {
		return base
	}

	var sb strings.Builder
	sb.WriteString(base[:funcStart])
	sb.WriteString("\n\n")
	sb.WriteString(targetsHelper)
	sb.WriteString("\n\n")
	sb.WriteString(langsHelper)
	sb.WriteString("\n\n")
	sb.WriteString(completionFunc)
	sb.WriteString("\n")
	sb.WriteString(base[completeLine:])
	return sb.String()
}
