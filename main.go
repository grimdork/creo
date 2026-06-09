package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/grimdork/climate/arg"
	"github.com/grimdork/creo/internal/fiat"
	"github.com/grimdork/creo/internal/lang"
	"github.com/grimdork/creo/internal/oci"
	"github.com/grimdork/creo/internal/runner"
)

var version string

func listTargets(explicitPath string) (string, error) {
	fiatPath, ok := fiat.FindFiat(explicitPath)
	if !ok {
		return "", fmt.Errorf("no fiat file found")
	}
	file, err := fiat.Parse(fiatPath)
	if err != nil {
		return "", fmt.Errorf("parsing %s: %w", fiatPath, err)
	}
	if err := lang.Apply(file); err != nil {
		return "", fmt.Errorf("applying defaults to %s: %w", fiatPath, err)
	}

	var b strings.Builder
	b.WriteString("Available targets:\n")
	for _, t := range file.Targets {
		ln := t.Language
		if ln == "" {
			ln = "-"
		}
		if t.Desc != "" {
			desc := fiat.ExpandWithTarget(t.Desc, file.Vars, t)
			fmt.Fprintf(&b, "  %-15s (%s)   %s\n", t.Name, ln, desc)
		} else {
			fmt.Fprintf(&b, "  %-15s (%s)\n", t.Name, ln)
		}
	}
	return b.String(), nil
}

func printVersion() {
	if version == "" {
		fmt.Println("creo (dev)")
	} else {
		fmt.Println("creo " + version)
	}
}

func runLogin() {
	if err := oci.Login(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Credentials stored")
}

func runInspect(ref string) {
	if err := oci.Inspect(ref); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runInit(langs []string, force, verbose bool) {
	if err := lang.InitProject(langs, force, verbose); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runList(filePath string) {
	out, err := listTargets(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(out)
}

func runBuild(opt *arg.Options) {
	opts := runner.RunOpts{
		Rebuild:        opt.GetBool("F"),
		Recursive:      opt.GetBool("r"),
		Clean:          opt.GetBool("c"),
		Verbose:        opt.GetBool("v"),
		Jobs:           opt.GetInt("j"),
		KeepGoing:      opt.GetBool("k"),
		DryRun:         opt.GetBool("n"),
		RefreshCACerts: opt.GetBool("refresh-cacerts"),
	}

	targets := opt.GetPosStringSlice("targets")
	if len(targets) == 0 {
		targets = []string{"build"}
	}

	if opts.Recursive {
		runner.RunRecursive(".", targets[0], opts)
		return
	}

	fiatPath, ok := fiat.FindFiat(opt.GetString("file"))
	if !ok {
		os.Exit(1)
	}

	file, err := fiat.Parse(fiatPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", fiatPath, err)
		os.Exit(1)
	}
	if err := lang.Apply(file); err != nil {
		fmt.Fprintf(os.Stderr, "Error applying defaults to %s: %v\n", fiatPath, err)
		os.Exit(1)
	}

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
	opt.SetFlag(arg.GroupDefault, "", "version", "Print version and exit")
	opt.SetFlag(arg.GroupDefault, "L", "login", "Store registry credentials in Docker config")
	opt.SetOption(arg.GroupDefault, "I", "inspect", "Inspect a remote image", "", false, arg.VarString, nil)
	opt.SetFlag(arg.GroupDefault, "", "completion", "Print shell completion script")
	opt.SetPositional("targets", "Targets to run or clean", nil, false, arg.VarStringSlice)

	if err := opt.Parse(os.Args[1:]); err != nil {
		if !errors.Is(err, arg.ErrNonFatal) {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	switch {
	case opt.GetBool("version"):
		printVersion()
	case opt.GetBool("completion"):
		fmt.Print(generateCompletion(opt))
	case opt.GetBool("login"):
		runLogin()
	case opt.GetString("inspect") != "":
		ref := opt.GetString("inspect")
		runInspect(ref)
	case opt.GetBool("i"):
		runInit(opt.GetPosStringSlice("targets"), opt.GetBool("F"), opt.GetBool("v"))
	case opt.GetBool("l"):
		runList(opt.GetString("file"))
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
