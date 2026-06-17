package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/grimdork/climate/fx"
)

// RunGitInit initialises a git repository, stages all files, and commits them.
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
