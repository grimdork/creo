package lang

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/grimdork/creo/internal/fiat"
)

func tryWrite(path, content string, force, verbose bool, label string) error {
	if _, err := os.Stat(path); err == nil {
		if force {
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return err
			}
			if verbose {
				fmt.Printf("  Replaced %s\n", label)
			}
		} else {
			if verbose {
				fmt.Printf("  Skipped %s (already exists)\n", label)
			}
		}
	} else {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
		if verbose {
			fmt.Printf("  Created %s\n", label)
		}
	}
	return nil
}

func initGoMod(dir, name string, force, verbose bool) error {
	modPath := filepath.Join(dir, "go.mod")
	if _, err := os.Stat(modPath); os.IsNotExist(err) {
		mod := exec.Command("go", "mod", "init", name)
		mod.Dir = dir
		if out, err := mod.CombinedOutput(); err != nil {
			return fmt.Errorf("go mod init: %s", strings.TrimSpace(string(out)))
		}
		if verbose {
			fmt.Println("  Initialised Go module")
		}
	} else if force {
		os.Remove(modPath)
		mod := exec.Command("go", "mod", "init", name)
		mod.Dir = dir
		if out, err := mod.CombinedOutput(); err != nil {
			return fmt.Errorf("go mod init: %s", strings.TrimSpace(string(out)))
		}
		if verbose {
			fmt.Println("  Reinitialised Go module")
		}
	} else if verbose {
		fmt.Println("  Skipped go.mod (already exists)")
	}
	return nil
}

func runGofmt(dir string) error {
	if out, err := exec.Command("gofmt", "-w", dir).CombinedOutput(); err != nil {
		return fmt.Errorf("gofmt: %s", strings.TrimSpace(string(out)))
	}
	if out, err := exec.Command("goimports", "-w", dir).CombinedOutput(); err != nil {
		return fmt.Errorf("goimports: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func runGoModTidy(dir string) error {
	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = dir
	if out, err := tidy.CombinedOutput(); err != nil {
		return fmt.Errorf("go mod tidy: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func absDirName(dir string) (string, string) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}
	return absDir, filepath.Base(absDir)
}

func ensureFiat(dir string, _ bool) (*fiat.File, error) {
	fiatPath := filepath.Join(dir, "fiat")
	if _, err := os.Stat(fiatPath); err == nil {
		file, err := fiat.Parse(fiatPath)
		if err != nil {
			return nil, err
		}
		return file, nil
	}
	file := fiat.NewFile(fiatPath)
	return file, nil
}

func writeGoSources(dir, name, ver string, force, verbose bool, file *fiat.File) error {
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

func Init(dir, ver string, force, verbose bool) ([]string, error) {
	_, name := absDirName(dir)
	file, err := ensureFiat(dir, force)
	if err != nil {
		return nil, err
	}

	if err := writeGoSources(dir, name, ver, force, verbose, file); err != nil {
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
				os.WriteFile(modPath, []byte(strings.Join(lines, "\n")), 0644)
				if verbose {
					fmt.Printf("  Added toolchain go%s\n", ver)
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

	return []string{"/" + name, "/.creo"}, nil
}

func InitC(dir string, force, verbose bool) ([]string, error) {
	file, err := ensureFiat(dir, force)
	if err != nil {
		return nil, err
	}

	if fiat.FindTarget(file, "build") == nil {
		bt := &fiat.Target{
			Name:     "build",
			Language: "c",
			Desc:     "Build the C binary",
		}
		file.AddTarget(bt)
	}

	mainContent := `#include <stdio.h>

int main(int argc, char **argv) {
	printf("hello\n");
	return 0;
}
`
	if err := tryWrite(filepath.Join(dir, "main.c"), mainContent,
		force, verbose, "main.c"); err != nil {
		return nil, err
	}

	if err := file.Write(); err != nil {
		return nil, err
	}

	return []string{"/main", "/.creo"}, nil
}

func InitCxx(dir string, force, verbose bool) ([]string, error) {
	file, err := ensureFiat(dir, force)
	if err != nil {
		return nil, err
	}

	if fiat.FindTarget(file, "build") == nil {
		bt := &fiat.Target{
			Name:     "build",
			Language: "cxx",
			Desc:     "Build the C++ binary",
		}
		file.AddTarget(bt)
	}

	mainContent := `#include <iostream>

int main(int argc, char **argv) {
	std::cout << "hello" << std::endl;
	return 0;
}
`
	if err := tryWrite(filepath.Join(dir, "main.cpp"), mainContent,
		force, verbose, "main.cpp"); err != nil {
		return nil, err
	}

	if err := file.Write(); err != nil {
		return nil, err
	}

	return []string{"/main", "/.creo"}, nil
}

func InitOci(dir string, force, verbose bool) ([]string, error) {
	file, err := ensureFiat(dir, force)
	if err != nil {
		return nil, err
	}

	// Ensure a Go build target exists
	if fiat.FindTarget(file, "build") == nil {
		_, name := absDirName(dir)
		if err := writeGoSources(dir, name, "", force, verbose, file); err != nil {
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
			fmt.Println("  Added oci target")
		}
	} else if verbose {
		fmt.Println("  Skipped oci target (already exists)")
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

	return []string{"/" + filepath.Base(dir), "/build", "/.creo"}, nil
}
