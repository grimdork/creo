package targets

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/grimdork/climate/fx"
	"github.com/grimdork/creo/internal/fiat"
)

func initGoMod(dir, name string, force, verbose bool) error {
	modPath := filepath.Join(dir, "go.mod")
	if _, err := os.Stat(modPath); os.IsNotExist(err) {
		mod := exec.Command("go", "mod", "init", name)
		mod.Dir = dir
		if out, err := mod.CombinedOutput(); err != nil {
			return fmt.Errorf(errGoModInit, strings.TrimSpace(string(out)))
		}
		if verbose {
			fx.Println("  {success}Initialised Go module{@}")
		}
	} else if force {
		if err := os.Remove(modPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing go.mod: %w", err)
		}
		mod := exec.Command("go", "mod", "init", name)
		mod.Dir = dir
		if out, err := mod.CombinedOutput(); err != nil {
			return fmt.Errorf(errGoModInit, strings.TrimSpace(string(out)))
		}
		if verbose {
			fx.Println("  {success}Reinitialised Go module{@}")
		}
	} else if verbose {
		fx.Println("  {warning}Skipped go.mod (already exists){@}")
	}
	return nil
}

func runGofmt(dir string) error {
	if out, err := exec.Command("gofmt", "-w", dir).CombinedOutput(); err != nil {
		return fmt.Errorf(errGofmt, strings.TrimSpace(string(out)))
	}
	if out, err := exec.Command("goimports", "-w", dir).CombinedOutput(); err != nil {
		return fmt.Errorf(errGoImports, strings.TrimSpace(string(out)))
	}
	return nil
}

func runGoModTidy(dir string) error {
	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = dir
	if out, err := tidy.CombinedOutput(); err != nil {
		return fmt.Errorf(errGoModTidy, strings.TrimSpace(string(out)))
	}
	return nil
}

func writeGoSources(dir, name string, force, verbose bool, file *fiat.File) error {
	if fiat.FindTarget(file, "build") == nil {
		bt := &fiat.Target{
			Name:     "build",
			Language: "go",
			Desc:     "Build the Go binary",
		}
		file.AddTarget(bt)
	}

	if err := initGoMod(dir, name, force, verbose); err != nil {
		return err
	}

	mainContent := `package main

import (
	"fmt"
	"runtime"
)

func main() {
	fmt.Printf("%s %s %s/%s\n", Name, version, runtime.GOOS, runtime.GOARCH)
}
`
	if err := tryWrite(filepath.Join(dir, "main.go"), mainContent, force, verbose, "main.go"); err != nil {
		return err
	}

	verContent := fmt.Sprintf(`package main

var Name = "%s"

var version string
`, name)
	if err := tryWrite(filepath.Join(dir, "version.go"), verContent, force, verbose, "version.go"); err != nil {
		return err
	}
	return nil
}

// Init scaffolds a Go project with a basic main.go and fiat file.
func Init(dir, ver string, force, verbose bool) ([]string, error) {
	_, name := absDirName(dir)
	file, err := ensureFiat(dir)
	if err != nil {
		return nil, err
	}

	if err := writeGoSources(dir, name, force, verbose, file); err != nil {
		return nil, err
	}

	if ver != "" {
		modPath := filepath.Join(dir, "go.mod")
		data, err := os.ReadFile(modPath)
		if err == nil {
			content := string(data)
			if !strings.Contains(content, "toolchain go") {
				lines := strings.Split(content, "\n")
				for i, line := range lines {
					if strings.HasPrefix(strings.TrimSpace(line), "module ") {
						tc := fmt.Sprintf("toolchain go%s", ver)
						lines = append(lines[:i+1], append([]string{tc}, lines[i+1:]...)...)
						break
					}
				}
				if err := os.WriteFile(modPath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
					return nil, fmt.Errorf("writing toolchain to go.mod: %w", err)
				}
				if verbose {
					fx.Println("  {success}Added toolchain go{}{@}", ver)
				}
			}
		}
	}

	if err := runGofmt(dir); err != nil {
		return nil, err
	}
	if err := runGoModTidy(dir); err != nil {
		return nil, err
	}
	if err := file.Write(); err != nil {
		return nil, err
	}

	return []string{GitignoreBuild, GitignoreCreo}, nil
}

// InitTinyGo scaffolds a TinyGo project with a basic main.go and fiat file.
func InitTinyGo(dir string, force, verbose bool) ([]string, error) {
	_, name := absDirName(dir)
	file, err := ensureFiat(dir)
	if err != nil {
		return nil, err
	}

	if err := writeGoSources(dir, name, force, verbose, file); err != nil {
		return nil, err
	}

	if bt := fiat.FindTarget(file, "build"); bt != nil {
		bt.Language = LangTinyGo
		bt.Desc = "Build with TinyGo"
	}

	if err := runGofmt(dir); err != nil {
		return nil, err
	}
	if err := runGoModTidy(dir); err != nil {
		return nil, err
	}
	if err := file.Write(); err != nil {
		return nil, err
	}

	return []string{GitignoreBuild, GitignoreCreo}, nil
}

// InitOci scaffolds an OCI image target on top of a Go project.
func InitOci(dir string, force, verbose bool) ([]string, error) {
	file, err := ensureFiat(dir)
	if err != nil {
		return nil, err
	}

	if fiat.FindTarget(file, "build") == nil {
		_, name := absDirName(dir)
		if err := writeGoSources(dir, name, force, verbose, file); err != nil {
			return nil, err
		}
	}

	if fiat.FindTarget(file, "image") == nil {
		img := &fiat.Target{
			Name:     "image",
			Language: "oci",
			Desc:     "Package and push OCI image",
			Requires: []string{"build"},
		}
		file.AddTarget(img)
		if verbose {
			fx.Println("  {success}Added oci target{@}")
		}
	} else if verbose {
		fx.Println("  {warning}Skipped oci target (already exists){@}")
	}

	if fiat.FindTarget(file, "build") != nil {
		if err := runGofmt(dir); err != nil {
			return nil, err
		}
		if err := runGoModTidy(dir); err != nil {
			return nil, err
		}
	}

	if err := file.Write(); err != nil {
		return nil, err
	}

	return []string{"/" + filepath.Base(dir), GitignoreBuild, GitignoreCreo}, nil
}
