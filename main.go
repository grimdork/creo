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

func listTargets(explicitPath string) error {
	fiatPath, ok := fiat.FindFiat(explicitPath)
	if !ok {
		return fmt.Errorf("no fiat file found")
	}
	file, err := fiat.Parse(fiatPath)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", fiatPath, err)
	}
	if err := lang.Apply(file); err != nil {
		return fmt.Errorf("applying defaults to %s: %w", fiatPath, err)
	}

	fmt.Println("Available targets:")
	for _, t := range file.Targets {
		ln := t.Language
		if ln == "" {
			ln = "-"
		}
		if t.Desc != "" {
			desc := fiat.ExpandWithTarget(t.Desc, file.Vars, t)
			fmt.Printf("  %-15s (%s)   %s\n", t.Name, ln, desc)
		} else {
			fmt.Printf("  %-15s (%s)\n", t.Name, ln)
		}
	}
	return nil
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

	if opt.GetBool("completion") {
		fmt.Print(generateCompletion(opt))
		return
	}

	if opt.GetBool("login") {
		if err := oci.Login(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Credentials stored")
		return
	}

	if ref := opt.GetString("inspect"); ref != "" {
		if err := oci.Inspect(ref); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if opt.GetBool("i") {
		langs := opt.GetPosStringSlice("targets")
		if err := initProject(langs, opt.GetBool("F"), opt.GetBool("v")); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

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

	if opt.GetBool("l") {
		if err := listTargets(opt.GetString("file")); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
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

const targetsHelper = `__creo_targets() {
	local fiat_file
	if [ -f "fiat" ]; then
		fiat_file="fiat"
	else
		for f in *.fiat; do
			if [ -f "$f" ]; then
				fiat_file="$f"
				break
			fi
		done
	fi
	if [ -n "$fiat_file" ]; then
		local targets
		targets=$(grep -E '^[a-zA-Z0-9._-]+:' "$fiat_file" | sed 's/:.*//' 2>/dev/null)
		COMPREPLY+=( $(compgen -W "$targets" -- "$cur") )
	fi
}`

const langsHelper = `__creo_langs() {
	COMPREPLY+=( $(compgen -W "go c cxx cpp oci" -- "$cur") )
}`

const completionFunc = `_creo() {
	COMPREPLY=()
	local cur prev
	_get_comp_words_by_ref cur prev

	if [ ${COMP_CWORD} -eq 1 ]; then
		if [[ ${cur} == -* ]]; then
			COMPREPLY=( $(compgen -W "${options}" -- $cur) )
			return 0
		fi

		__creo_targets
		return 0
	fi

	if [[ ${cur} == -* ]]; then
		case ${prev} in
		*)
			if [[ $(hasword ${prev} ${options}) == "1" ]]; then
				COMPREPLY=( $(compgen -W "${options}" -- $cur) )
				return 0
			fi
			;;
		esac
	fi

	if [[ $(hasword ${prev} ${options}) == "1" ]]; then
		case ${prev} in
		-i|--init)
			__creo_langs
			;;
		-f|--file)
			complete_files
			;;
		*)
			__creo_targets
			;;
		esac
		return 0
	fi

	__creo_targets
	return 0
}`
