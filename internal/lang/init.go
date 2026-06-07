package lang

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Init(dir, ver string, force, verbose bool) ([]string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}
	name := filepath.Base(absDir)

	fiatPath := filepath.Join(dir, "fiat")
	if _, err := os.Stat(fiatPath); err == nil && !force {
		return nil, fmt.Errorf("fiat already exists")
	}

	fiatContent := "build: go\n\ndebug: go\n"
	if err := os.WriteFile(fiatPath, []byte(fiatContent), 0644); err != nil {
		return nil, err
	}
	if verbose {
		fmt.Println("  Created fiat")
	}

	mod := exec.Command("go", "mod", "init", name)
	mod.Dir = dir
	if out, err := mod.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("go mod init: %s", strings.TrimSpace(string(out)))
	}
	if verbose {
		fmt.Println("  Initialised Go module")
	}

	if ver != "" {
		modPath := filepath.Join(dir, "go.mod")
		data, err := os.ReadFile(modPath)
		if err == nil {
			lines := strings.Split(string(data), "\n")
			for i, line := range lines {
				if strings.HasPrefix(strings.TrimSpace(line), "module ") {
					toolchain := fmt.Sprintf("toolchain go%s", ver)
					lines = append(lines[:i+1], append([]string{toolchain}, lines[i+1:]...)...)
					break
				}
			}
			os.WriteFile(modPath, []byte(strings.Join(lines, "\n")), 0644)
			if verbose {
				fmt.Printf("  Added toolchain go%s\n", ver)
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
	mainPath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		return nil, err
	}

	verContent := fmt.Sprintf(`package main

var Name = "%s"

var version string
`, name)
	verPath := filepath.Join(dir, "version.go")
	if err := os.WriteFile(verPath, []byte(verContent), 0644); err != nil {
		return nil, err
	}

	if verbose {
		fmt.Println("  Created main.go, version.go")
	}

	exec.Command("gofmt", "-w", dir).Run()
	exec.Command("goimports", "-w", dir).Run()
	if verbose {
		fmt.Println("  Formatted with gofmt, goimports")
	}

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = dir
	tidy.Run()
	if verbose {
		fmt.Println("  Ran go mod tidy")
	}

	return []string{"/" + name, "/.creo"}, nil
}
