package main

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

func injectBuildDir(f *fiat.File, bd string) {
	f.Vars["BUILDDIR"] = &fiat.Var{Name: "BUILDDIR", Value: bd}
}

func runLogin() {
	if err := oci.Login(); err != nil {
		fail(err)
	}
	fx.Println("{success}Credentials stored{@}")
}

func runInspect(ref string) {
	if err := oci.Inspect(ref); err != nil {
		fail(err)
	}
}

func runInit(langs []string, force, verbose bool) {
	if err := targets.InitProject(langs, force, verbose); err != nil {
		fail(err)
	}
}

func runGitInit(verbose bool) {
	git := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	if err := git("init"); err != nil {
		failf("git init: %v", err)
	}
	if verbose {
		fx.Println("  {success}Initialised git repository{@}")
	}

	if err := git("add", "-A"); err != nil {
		failf("git add: %v", err)
	}

	out, err := exec.Command("git", "diff", "--cached", "--name-only").Output()
	if err != nil {
		failf("listing staged files: %v", err)
	}

	files := strings.TrimSpace(string(out))
	if files == "" {
		if verbose {
			fx.Println("  {warning}Nothing to commit{@}")
		}
		return
	}

	body := ""
	for _, f := range strings.Split(files, "\n") {
		body += "\n- " + f
	}
	msg := "Initial scaffolding" + body
	if err := git("commit", "-m", msg); err != nil {
		failf("git commit: %v", err)
	}
}

func runGraph(opt *arg.Options) {
	format := opt.GetString("graph")
	if format != "tree" && format != "dot" && format != "svg" {
		failf("--graph must be 'tree', 'dot', or 'svg'")
	}

	fiatPath, ok := fiat.FindFiat(opt.GetString("file"))
	if !ok {
		failf("no fiat file found or file inaccessible")
	}
	dir := filepath.Dir(fiatPath)

	file, err := fiat.Parse(fiatPath)
	if err != nil {
		failf("parsing %s: %v", fiatPath, err)
	}
	if bd := opt.GetString("output"); bd != "" {
		injectBuildDir(file, bd)
	}
	if err := targets.Apply(file); err != nil {
		failf("applying defaults to %s: %v", fiatPath, err)
	}

	out, err := runner.RenderGraph(file, dir, format, opt.GetBool("status"))
	if err != nil {
		fail(err)
	}
	fmt.Print(out)
}

func runBuild(opt *arg.Options) {
	if opt.GetBool("no-color") || opt.GetBool("no-colour") {
		os.Setenv("NO_COLOR", "1")
	}

	results := &runner.TargetResults{}
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
		Results:        results,
	}

	names := opt.GetPosStringSlice("targets")
	if len(names) == 0 {
		names = []string{"build"}
	}

	if opts.Recursive {
		if err := runner.RunRecursive(".", names[0], opts); err != nil {
			failf("recursive build: %v", err)
		}
		results.Print()
		return
	}

	fiatPath, ok := fiat.FindFiat(opt.GetString("file"))
	if !ok {
		failf("no fiat file found or file inaccessible")
	}

	file, err := fiat.Parse(fiatPath)
	if err != nil {
		failf("parsing %s: %v", fiatPath, err)
	}
	if bd := opt.GetString("output"); bd != "" {
		injectBuildDir(file, bd)
	}
	if err := targets.Apply(file); err != nil {
		failf("applying defaults to %s: %v", fiatPath, err)
	}

	if opt.GetBool("w") {
		runner.RunWatch(file, names[0], opts)
		return
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
	if errCount > 0 {
		os.Exit(1)
	}
}
