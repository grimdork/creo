package lang

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func Init(dir, ver string, force, verbose bool) ([]string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}
	name := filepath.Base(absDir)

	if err := tryWrite(
		filepath.Join(dir, "fiat"),
		"build: go\n\ndebug: go\n",
		force, verbose, "fiat",
	); err != nil {
		return nil, err
	}

	modPath := filepath.Join(dir, "go.mod")
	if _, err := os.Stat(modPath); os.IsNotExist(err) {
		mod := exec.Command("go", "mod", "init", name)
		mod.Dir = dir
		if out, err := mod.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("go mod init: %s", strings.TrimSpace(string(out)))
		}
		if verbose {
			fmt.Println("  Initialised Go module")
		}
	} else if force {
		os.Remove(modPath)
		mod := exec.Command("go", "mod", "init", name)
		mod.Dir = dir
		if out, err := mod.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("go mod init: %s", strings.TrimSpace(string(out)))
		}
		if verbose {
			fmt.Println("  Reinitialised Go module")
		}
	} else if verbose {
		fmt.Println("  Skipped go.mod (already exists)")
	}

	if ver != "" {
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
		return nil, err
	}

	verContent := fmt.Sprintf(`package main

var Name = "%s"

var version string
`, name)
	if err := tryWrite(filepath.Join(dir, "version.go"), verContent, force, verbose, "version.go"); err != nil {
		return nil, err
	}

	exec.Command("gofmt", "-w", dir).Run()
	exec.Command("goimports", "-w", dir).Run()

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = dir
	tidy.Run()

	return []string{"/" + name, "/.creo"}, nil
}

func InitC(dir string, force, verbose bool) ([]string, error) {
	if err := tryWrite(filepath.Join(dir, "fiat"),
		"build: c\n",
		force, verbose, "fiat",
	); err != nil {
		return nil, err
	}

	mainContent := `#include <stdio.h>

int main(int argc, char **argv) {
	printf("hello\\n");
	return 0;
}
`
	if err := tryWrite(filepath.Join(dir, "main.c"), mainContent,
		force, verbose, "main.c"); err != nil {
		return nil, err
	}

	return []string{"/main", "/.creo"}, nil
}

func InitCxx(dir string, force, verbose bool) ([]string, error) {
	if err := tryWrite(filepath.Join(dir, "fiat"),
		"build: cxx\n",
		force, verbose, "fiat",
	); err != nil {
		return nil, err
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

	return []string{"/main", "/.creo"}, nil
}

func InitKo(dir string, force, verbose bool) ([]string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}
	name := filepath.Base(absDir)

	if err := tryWrite(filepath.Join(dir, "fiat"),
		"build: ko\n",
		force, verbose, "fiat",
	); err != nil {
		return nil, err
	}

	modPath := filepath.Join(dir, "go.mod")
	if _, err := os.Stat(modPath); os.IsNotExist(err) {
		mod := exec.Command("go", "mod", "init", name)
		mod.Dir = dir
		if out, err := mod.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("go mod init: %s", strings.TrimSpace(string(out)))
		}
		if verbose {
			fmt.Println("  Initialised Go module")
		}
	} else if force {
		os.Remove(modPath)
		mod := exec.Command("go", "mod", "init", name)
		mod.Dir = dir
		if out, err := mod.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("go mod init: %s", strings.TrimSpace(string(out)))
		}
		if verbose {
			fmt.Println("  Reinitialised Go module")
		}
	} else if verbose {
		fmt.Println("  Skipped go.mod (already exists)")
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
	if err := tryWrite(filepath.Join(dir, "main.go"), mainContent,
		force, verbose, "main.go"); err != nil {
		return nil, err
	}

	verContent := fmt.Sprintf(`package main

var Name = "%s"

var version string
`, name)
	if err := tryWrite(filepath.Join(dir, "version.go"), verContent,
		force, verbose, "version.go"); err != nil {
		return nil, err
	}

	exec.Command("gofmt", "-w", dir).Run()
	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = dir
	tidy.Run()

	return []string{"/build", "/.creo"}, nil
}
