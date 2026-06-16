package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/grimdork/climate/arg"
	"github.com/grimdork/climate/fx"
	"github.com/grimdork/creo/internal/fiat"
	"github.com/grimdork/creo/internal/oci"
	"github.com/grimdork/creo/internal/runner"
	"github.com/grimdork/creo/internal/targets"
)

func InjectBuildDir(f *fiat.File, bd string) {
	f.Vars["BUILDDIR"] = &fiat.Var{Name: "BUILDDIR", Value: bd}
}

func RunLogin() error {
	if err := oci.Login(); err != nil {
		return err
	}
	fx.Println("{success}Credentials stored{@}")
	return nil
}

func RunInspect(ref string) error {
	return oci.Inspect(ref)
}

func RunInit(langs []string, force, verbose bool) error {
	return targets.InitProject(langs, force, verbose)
}

func RunGitInit(verbose bool) error {
	git := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	if err := git("init"); err != nil {
		return fmt.Errorf("git init: %w", err)
	}
	if verbose {
		fx.Println("  {success}Initialised git repository{@}")
	}

	if err := git("add", "-A"); err != nil {
		return fmt.Errorf("git add: %w", err)
	}

	out, err := exec.Command("git", "diff", "--cached", "--name-only").Output()
	if err != nil {
		return fmt.Errorf("listing staged files: %w", err)
	}

	files := strings.TrimSpace(string(out))
	if files == "" {
		if verbose {
			fx.Println("  {warning}Nothing to commit{@}")
		}
		return nil
	}

	body := ""
	for _, f := range strings.Split(files, "\n") {
		body += "\n- " + f
	}
	msg := "Initial scaffolding" + body
	if err := git("commit", "-m", msg); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}
	return nil
}

func RunGraph(opt *arg.Options) error {
	format := opt.GetString("graph")
	if !runner.ValidGraphFormat(format) {
		return fmt.Errorf("--graph must be 'tree', 'dot', or 'svg'")
	}

	fiatPath, ok := fiat.FindFiat(opt.GetString("file"))
	if !ok {
		return fmt.Errorf("no fiat file found or file inaccessible")
	}
	dir := filepath.Dir(fiatPath)

	file, err := fiat.Parse(fiatPath)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", fiatPath, err)
	}
	if bd := opt.GetString("output"); bd != "" {
		InjectBuildDir(file, bd)
	}
	if err := targets.Apply(file); err != nil {
		return fmt.Errorf("applying defaults to %s: %w", fiatPath, err)
	}

	out, err := runner.RenderGraph(file, dir, format, opt.GetBool("status"))
	if err != nil {
		return err
	}
	fmt.Print(out)
	return nil
}

func RunVersion(ver string) {
	if ver == "" {
		fx.Println("{bold}creo (dev){@}")
	} else {
		fx.Println("{bold}creo {}{@}", ver)
	}
}

func ListTargets(explicitPath string) (string, error) {
	fiatPath, ok := fiat.FindFiat(explicitPath)
	if !ok {
		return "", fmt.Errorf("no fiat file found")
	}
	file, err := fiat.Parse(fiatPath)
	if err != nil {
		return "", fmt.Errorf("parsing %s: %w", fiatPath, err)
	}
	if err := targets.Apply(file); err != nil {
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

func RunList(filePath string) error {
	out, err := ListTargets(filePath)
	if err != nil {
		return err
	}
	fmt.Print(out)
	return nil
}

func RunBuild(opt *arg.Options) error {
	if opt.GetBool("no-color") || opt.GetBool("no-colour") {
		os.Setenv("NO_COLOR", "1")
	}

	results := &runner.TargetResults{}
	cacheRemote := opt.GetString("cache-remote")
	if cacheRemote == "" {
		cacheRemote = os.Getenv("CREO_CACHE_REMOTE")
	}
	cacheStats := &runner.CacheStats{}
	opts := runner.RunOpts{
		Rebuild:        opt.GetBool("F"),
		Recursive:      opt.GetBool("r"),
		Clean:          opt.GetBool("c"),
		Verbose:        opt.GetBool("v"),
		Jobs:           opt.GetInt("j"),
		KeepGoing:      opt.GetBool("k"),
		DryRun:         opt.GetBool("n"),
		RefreshCACerts: opt.GetBool("refresh-cacerts"),
		BuildDir:       opt.GetString("output"),
		NoColor:        opt.GetBool("no-color") || opt.GetBool("no-colour"),
		CacheRemote:    cacheRemote,
		CacheStats:     cacheStats,
		Results:        results,
	}

	names := opt.GetPosStringSlice("targets")
	if len(names) == 0 {
		names = []string{"build"}
	}

	if opts.Recursive {
		if err := runner.RunRecursive(".", names[0], opts); err != nil {
			return fmt.Errorf("recursive build: %w", err)
		}
		results.Print()
		if opt.GetBool("cache-stats") {
			cacheStats.Print()
		}
		return nil
	}

	fiatPath, ok := fiat.FindFiat(opt.GetString("file"))
	if !ok {
		return fmt.Errorf("no fiat file found or file inaccessible")
	}

	file, err := fiat.Parse(fiatPath)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", fiatPath, err)
	}
	if bd := opt.GetString("output"); bd != "" {
		InjectBuildDir(file, bd)
	}
	if err := targets.Apply(file); err != nil {
		return fmt.Errorf("applying defaults to %s: %w", fiatPath, err)
	}

	if opt.GetBool("w") {
		runner.RunWatch(file, names[0], opts)
		return nil
	}

	var errCount int
	for _, name := range names {
		if err := runner.RunTarget(file, name, opts); err != nil {
			fx.Fprint(os.Stderr, "{red}Error: {}{}{@}\n", err)
			errCount++
			if !opts.KeepGoing {
				break
			}
		}
	}
	results.Print()
	if opt.GetBool("cache-stats") {
		cacheStats.Print()
	}
	if errCount > 0 {
		return fmt.Errorf("some targets failed")
	}
	return nil
}
