package runner

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/grimdork/climate/fx"
	"github.com/grimdork/creo/internal/fiat"
)

func handleBrew(bt *buildTask) {
	f, t, c, comboVars, errCh, dir := bt.f, bt.t, bt.c, bt.comboVars, bt.errCh, bt.dir
	archivePath := ""
	for _, dep := range t.Requires {
		outVar := "OUTPUT_" + dep
		if v, ok := comboVars[outVar]; ok && v.Value != "" {
			archivePath = v.Value
			break
		}
	}
	if archivePath == "" {
		archivePath = c.bin
	}

	shaHex, shaErr := computeSHA256(archivePath)
	if shaErr != nil {
		errCh <- fmt.Errorf("%s: SHA256 of %s: %w", f.Path(), archivePath, shaErr)
		return
	}

	ver := strings.TrimPrefix(fiat.Expand("$VERSION", comboVars, 0), "v")
	archiveName := filepath.Base(archivePath)
	projName := fiat.Expand("$PROJECT", comboVars, 0)

	brewRepo := fiat.Expand(t.Brew.Repo, comboVars, 0)
	brewDesc := fiat.Expand(t.Brew.Desc, comboVars, 0)
	brewHomepage := fiat.Expand(t.Brew.Homepage, comboVars, 0)
	brewLicense := fiat.Expand(t.Brew.License, comboVars, 0)
	brewClassName := fiat.Expand(t.Brew.ClassName, comboVars, 0)

	url := fmt.Sprintf("https://github.com/%s/releases/download/v%s/%s",
		brewRepo, ver, archiveName)
	if brewRepo == "" {
		url = archiveName
	}

	className := brewClassName
	if className == "" {
		className = projName
	}

	var formulaBuf strings.Builder
	formulaBuf.WriteString(fmt.Sprintf(`class %s < Formula
  desc %q
  homepage %q
  url %q
  version %q
  sha256 %q
  license %q

  def install
    bin.install %q
  end
end
`, className, brewDesc, brewHomepage, url, ver, shaHex, brewLicense, projName))

	outputPath := t.Brew.Output
	if outputPath == "" {
		outputPath = filepath.Join(dir, t.Name+".rb")
	} else {
		outputPath = fiat.Expand(outputPath, comboVars, 0)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		errCh <- fmt.Errorf("%s: brew output dir: %w", f.Path(), err)
		return
	}
	if err := os.WriteFile(outputPath, []byte(formulaBuf.String()), 0644); err != nil {
		errCh <- fmt.Errorf("%s: writing brew formula: %w", f.Path(), err)
		return
	}
	fx.Println(`  {success}Wrote brew formula {}{@}`, outputPath)

	if t.Brew.Tap != "" {
		token := fiat.Expand(t.Brew.Token, comboVars, 0)
		token = os.ExpandEnv(token)
		if token == "" {
			token = os.Getenv("GH_TOKEN")
		}
		if token == "" {
			token = os.Getenv("GITHUB_TOKEN")
		}

		if token == "" {
			errCh <- fmt.Errorf("%s: GH_TOKEN required for brew tap push", f.Path())
			return
		}

		askPass, err := os.CreateTemp("", "creo-git-askpass-*")
		if err != nil {
			errCh <- fmt.Errorf("%s: creating askpass script: %w", f.Path(), err)
			return
		}
		askPassPath := askPass.Name()
		askPass.Close()
		defer os.Remove(askPassPath)
		if err := os.WriteFile(askPassPath, []byte("#!/bin/sh\necho \"${GIT_TOKEN}\"\n"), 0755); err != nil {
			errCh <- fmt.Errorf("%s: writing askpass script: %w", f.Path(), err)
			return
		}

		brewTap := fiat.Expand(t.Brew.Tap, comboVars, 0)
		tapDir := filepath.Join(filepath.Dir(f.Path()), ".creo", t.Name+"-tap")
		cloneURL := fmt.Sprintf("https://github.com/%s.git", brewTap)

		gitEnv := append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS="+askPassPath, "GIT_TOKEN="+token)

		cmd := exec.Command("git", "clone", cloneURL, tapDir)
		cmd.Env = gitEnv
		if out, err := cmd.CombinedOutput(); err != nil {
			errCh <- fmt.Errorf("%s: cloning tap %s: %w\n%s", f.Path(), brewTap, err, strings.TrimSpace(string(out)))
			return
		}

		formulaDir := filepath.Join(tapDir, "Formula")
		if err := os.MkdirAll(formulaDir, 0755); err != nil {
			errCh <- fmt.Errorf("%s: creating Formula dir: %w", f.Path(), err)
			return
		}
		formulaDest := filepath.Join(formulaDir, projName+".rb")
		if err := os.WriteFile(formulaDest, []byte(formulaBuf.String()), 0644); err != nil {
			errCh <- fmt.Errorf("%s: writing formula in tap: %w", f.Path(), err)
			return
		}

		gitCmds := [][]string{
			{"-C", tapDir, "config", "user.name", "creo"},
			{"-C", tapDir, "config", "user.email", "creo@localhost"},
			{"-C", tapDir, "add", "Formula/" + projName + ".rb"},
			{"-C", tapDir, "commit", "-m", fmt.Sprintf("Update %s to %s", projName, ver)},
			{"-C", tapDir, "push"},
		}
		for _, args := range gitCmds {
			c := exec.Command("git", args...)
			c.Env = gitEnv
			if out, err := c.CombinedOutput(); err != nil {
				errCh <- fmt.Errorf("%s: git %s: %w\n%s", f.Path(), args[0], err, strings.TrimSpace(string(out)))
				return
			}
		}
		fx.Println(`  {success}Pushed {} to {}{@}`, projName, brewTap)
	}
}

func computeSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), nil
}
