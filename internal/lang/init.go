package lang

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/grimdork/climate/fx"
	"github.com/grimdork/creo/internal/fiat"
	"github.com/grimdork/creo/internal/util"
)

func tryWrite(path, content string, force, verbose bool, label string) error {
	if _, err := os.Stat(path); err == nil {
		if force {
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return err
			}
			if verbose {
				fx.Println("  {success}Replaced {}{@}", label)
			}
		} else {
			if verbose {
				fx.Println("  {warning}Skipped {} (already exists){@}", label)
			}
		}
	} else {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
		if verbose {
			fx.Println("  {success}Created {}{@}", label)
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
			fx.Println("  {success}Initialised Go module{@}")
		}
	} else if force {
		if err := os.Remove(modPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing go.mod: %w", err)
		}
		mod := exec.Command("go", "mod", "init", name)
		mod.Dir = dir
		if out, err := mod.CombinedOutput(); err != nil {
			return fmt.Errorf("go mod init: %s", strings.TrimSpace(string(out)))
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

func ensureFiat(dir string) (*fiat.File, error) {
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
	fx.Println("{bold}{} {} {}/{}{@}", Name, version, runtime.GOOS, runtime.GOARCH)
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
	file, err := ensureFiat(dir)
	if err != nil {
		return nil, err
	}

	if err := writeGoSources(dir, name, force, verbose, file); err != nil {
		return nil, err
	}

	if bt := fiat.FindTarget(file, "build"); bt != nil {
		bt.Language = "tinygo"
		bt.Desc = "Build with TinyGo"
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

	return []string{"/build", "/.creo"}, nil
}

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
		bt.Language = "tinygo"
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

	return []string{"/build", "/.creo"}, nil
}

func InitC(dir string, force, verbose bool) ([]string, error) {
	file, err := ensureFiat(dir)
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
	file, err := ensureFiat(dir)
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
	file, err := ensureFiat(dir)
	if err != nil {
		return nil, err
	}

	// Ensure a Go build target exists
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

	return []string{"/" + filepath.Base(dir), "/build", "/.creo"}, nil
}

func InitRust(dir string, force, verbose bool) ([]string, error) {
	file, err := ensureFiat(dir)
	if err != nil {
		return nil, err
	}

	if fiat.FindTarget(file, "build") == nil {
		bt := &fiat.Target{
			Name:     "build",
			Language: "rust",
			Desc:     "Build the Rust binary",
		}
		file.AddTarget(bt)
	}

	cargoPath := filepath.Join(dir, "Cargo.toml")
	if _, err := os.Stat(cargoPath); os.IsNotExist(err) {
		cmd := exec.Command("cargo", "init", "--name", filepath.Base(dir))
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("cargo init: %s", strings.TrimSpace(string(out)))
		}
		if verbose {
			fx.Println("  {success}Initialised Cargo project{@}")
		}
	} else if force {
		if err := os.Remove(cargoPath); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("removing Cargo.toml: %w", err)
		}
		cmd := exec.Command("cargo", "init", "--name", filepath.Base(dir))
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("cargo init: %s", strings.TrimSpace(string(out)))
		}
		if verbose {
			fx.Println("  {success}Reinitialised Cargo project{@}")
		}
	} else if verbose {
		fx.Println("  {warning}Skipped Cargo.toml (already exists){@}")
	}

	if err := file.Write(); err != nil {
		return nil, err
	}

	return []string{"/.creo"}, nil
}

func InitPython(dir string, force, verbose bool) ([]string, error) {
	file, err := ensureFiat(dir)
	if err != nil {
		return nil, err
	}

	if fiat.FindTarget(file, "build") == nil {
		bt := &fiat.Target{
			Name:     "build",
			Language: "python",
			Desc:     "Sync Python dependencies and prepare source",
		}
		file.AddTarget(bt)
	}

	_, proj := absDirName(dir)

	pyproject := `[project]
name = "` + proj + `"
version = "0.1.0"
requires-python = ">=3.11"
dependencies = []

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
`
	if err := tryWrite(filepath.Join(dir, "pyproject.toml"), pyproject,
		force, verbose, "pyproject.toml"); err != nil {
		return nil, err
	}

	srcDir := filepath.Join(dir, "src", proj)
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		return nil, fmt.Errorf("creating src/%s: %w", proj, err)
	}

	initContent := `def main() -> None:
    print("hello from ` + proj + `")


if __name__ == "__main__":
    main()
`
	initPath := filepath.Join(srcDir, "main.py")
	if err := tryWrite(initPath, initContent, force, verbose, "src/"+proj+"/main.py"); err != nil {
		return nil, err
	}

	emptyInit := ""
	emptyInitPath := filepath.Join(srcDir, "__init__.py")
	if err := tryWrite(emptyInitPath, emptyInit, force, verbose, "src/"+proj+"/__init__.py"); err != nil {
		return nil, err
	}

	if err := file.Write(); err != nil {
		return nil, err
	}

	return []string{"__pycache__/", "*.pyc", "dist/", "*.egg-info/", ".venv/", "/.creo"}, nil
}

func InitNode(dir string, force, verbose bool) ([]string, error) {
	file, err := ensureFiat(dir)
	if err != nil {
		return nil, err
	}

	if fiat.FindTarget(file, "build") == nil {
		bt := &fiat.Target{
			Name:     "build",
			Language: "node",
			Desc:     "Build the Node/TypeScript project",
		}
		file.AddTarget(bt)
	}

	_, proj := absDirName(dir)

	pkgJSON := `{
  "name": "` + proj + `",
  "version": "0.1.0",
  "private": true,
  "main": "dist/index.js",
  "scripts": {
    "build": "tsc"
  },
  "devDependencies": {
    "typescript": "^5.0.0"
  }
}
`
	if err := tryWrite(filepath.Join(dir, "package.json"), pkgJSON,
		force, verbose, "package.json"); err != nil {
		return nil, err
	}

	tsconfig := `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "NodeNext",
    "moduleResolution": "nodenext",
    "outDir": "dist",
    "rootDir": "src",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true
  },
  "include": ["src/**/*"]
}
`
	if err := tryWrite(filepath.Join(dir, "tsconfig.json"), tsconfig,
		force, verbose, "tsconfig.json"); err != nil {
		return nil, err
	}

	srcDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		return nil, fmt.Errorf("creating src/: %w", err)
	}

	indexContent := `const greeting: string = "hello from ` + proj + `";
console.log(greeting);
`
	if err := tryWrite(filepath.Join(srcDir, "index.ts"), indexContent,
		force, verbose, "src/index.ts"); err != nil {
		return nil, err
	}

	if err := file.Write(); err != nil {
		return nil, err
	}

	return []string{"node_modules/", "dist/", "/.creo"}, nil
}

func InitJava(dir string, force, verbose bool) ([]string, error) {
	file, err := ensureFiat(dir)
	if err != nil {
		return nil, err
	}

	if fiat.FindTarget(file, "build") == nil {
		bt := &fiat.Target{
			Name:     "build",
			Language: "java",
			Desc:     "Build the Java/Kotlin project",
		}
		file.AddTarget(bt)
	}

	_, proj := absDirName(dir)
	pkg := "com." + strings.ToLower(proj)
	pkgPath := strings.ReplaceAll(pkg, ".", "/")

	settings := `rootProject.name = "` + proj + `"
`
	if err := tryWrite(filepath.Join(dir, "settings.gradle.kts"), settings,
		force, verbose, "settings.gradle.kts"); err != nil {
		return nil, err
	}

	buildGradle := `plugins {
    kotlin("jvm") version "2.0.0"
    application
}

application {
    mainClass = "` + pkg + `.AppKt"
}

repositories {
    mavenCentral()
}

dependencies {
    implementation(kotlin("stdlib"))
}
`
	if err := tryWrite(filepath.Join(dir, "build.gradle.kts"), buildGradle,
		force, verbose, "build.gradle.kts"); err != nil {
		return nil, err
	}

	klassDir := filepath.Join(dir, "src", "main", "kotlin", pkgPath)
	if err := os.MkdirAll(klassDir, 0755); err != nil {
		return nil, fmt.Errorf("creating src/main/kotlin/%s: %w", pkgPath, err)
	}

	appContent := `package ` + pkg + `

fun main() {
    println("hello from ` + proj + `")
}
`
	if err := tryWrite(filepath.Join(klassDir, "App.kt"), appContent,
		force, verbose, "src/main/kotlin/"+pkgPath+"/App.kt"); err != nil {
		return nil, err
	}

	if err := file.Write(); err != nil {
		return nil, err
	}

	return []string{"build/", ".gradle/", "*.jar", "/.creo"}, nil
}

func gitConfigUser() string {
	out, err := exec.Command("git", "config", "--global", "user.name").Output()
	if err != nil {
		return ""
	}
	name := strings.TrimSpace(string(out))
	out, err = exec.Command("git", "config", "--global", "user.email").Output()
	if err != nil {
		return name
	}
	email := strings.TrimSpace(string(out))
	if email != "" {
		return name + " <" + email + ">"
	}
	return name
}

func InitArchive(dir string, force, verbose bool) ([]string, error) {
	file, err := ensureFiat(dir)
	if err != nil {
		return nil, err
	}
	if fiat.FindTarget(file, "archive") == nil {
		at := &fiat.Target{
			Name:     "archive",
			Language: "archive",
			Desc:     "Create release archive",
			Requires: []string{"build"},
		}
		file.AddTarget(at)
		if verbose {
			fx.Println("  {success}Added archive target{@}")
		}
	} else if verbose {
		fx.Println("  {warning}Skipped archive target (already exists){@}")
	}
	if err := file.Write(); err != nil {
		return nil, err
	}
	return nil, nil
}

func InitDeb(dir string, force, verbose bool) ([]string, error) {
	file, err := ensureFiat(dir)
	if err != nil {
		return nil, err
	}
	if fiat.FindTarget(file, "deb") == nil {
		maint := gitConfigUser()
		if maint == "" {
			maint = "packager <root@localhost>"
		}
		dt := &fiat.Target{
			Name:     "deb",
			Language: "deb",
			Desc:     "Create .deb package",
			Requires: []string{"build"},
		}
		dt.Vars = append(dt.Vars, &fiat.Var{Name: "maintainer", Value: maint})
		file.AddTarget(dt)
		if verbose {
			fx.Println("  {success}Added deb target{@}")
		}
	} else if verbose {
		fx.Println("  {warning}Skipped deb target (already exists){@}")
	}
	if err := file.Write(); err != nil {
		return nil, err
	}
	return nil, nil
}

func InitRpm(dir string, force, verbose bool) ([]string, error) {
	file, err := ensureFiat(dir)
	if err != nil {
		return nil, err
	}
	if fiat.FindTarget(file, "rpm") == nil {
		maint := gitConfigUser()
		if maint == "" {
			maint = "packager <root@localhost>"
		}
		rt := &fiat.Target{
			Name:     "rpm",
			Language: "rpm",
			Desc:     "Create .rpm package",
			Requires: []string{"build"},
		}
		rt.Vars = append(rt.Vars, &fiat.Var{Name: "maintainer", Value: maint})
		file.AddTarget(rt)
		if verbose {
			fx.Println("  {success}Added rpm target{@}")
		}
	} else if verbose {
		fx.Println("  {warning}Skipped rpm target (already exists){@}")
	}
	if err := file.Write(); err != nil {
		return nil, err
	}
	return nil, nil
}

func InitBrew(dir string, force, verbose bool) ([]string, error) {
	file, err := ensureFiat(dir)
	if err != nil {
		return nil, err
	}
	if fiat.FindTarget(file, "brew") == nil {
		bt := &fiat.Target{
			Name:     "brew",
			Language: "brew",
			Desc:     "Create Homebrew formula",
			Requires: []string{"archive"},
		}
		bt.Vars = append(bt.Vars, &fiat.Var{Name: "tap", Value: "user/homebrew-tools"})
		file.AddTarget(bt)
		if verbose {
			fx.Println("  {success}Added brew target{@}")
		}
	} else if verbose {
		fx.Println("  {warning}Skipped brew target (already exists){@}")
	}
	if err := file.Write(); err != nil {
		return nil, err
	}
	return nil, nil
}

func InitProject(langs []string, force, verbose bool) error {
	if force {
		if _, err := os.Stat(".creo"); err == nil {
			if err := os.RemoveAll(".creo"); err != nil {
				return fmt.Errorf("removing .creo: %w", err)
			}
			if verbose {
				fx.Println("  {muted}Removed .creo/{@}")
			}
		}
	}

	if len(langs) == 0 {
		return fiat.WriteDefaultFile("fiat", force, verbose)
	}

	var allIgnores []string

	for _, spec := range langs {
		langName, ver := spec, ""
		if idx := strings.IndexByte(spec, ':'); idx >= 0 {
			langName, ver = spec[:idx], spec[idx+1:]
		}

		var ignores []string
		var err error

		switch langName {
		case "go":
			ignores, err = Init(".", ver, force, verbose)
		case "tinygo":
			ignores, err = InitTinyGo(".", force, verbose)
		case "c":
			ignores, err = InitC(".", force, verbose)
		case "cxx", "cpp":
			ignores, err = InitCxx(".", force, verbose)
		case "oci":
			ignores, err = InitOci(".", force, verbose)
		case "rust":
			ignores, err = InitRust(".", force, verbose)
		case "python":
			ignores, err = InitPython(".", force, verbose)
		case "node", "typescript":
			ignores, err = InitNode(".", force, verbose)
		case "java", "kotlin", "gradle":
			ignores, err = InitJava(".", force, verbose)
		case "archive":
			ignores, err = InitArchive(".", force, verbose)
		case "deb":
			ignores, err = InitDeb(".", force, verbose)
		case "rpm":
			ignores, err = InitRpm(".", force, verbose)
		case "brew":
			ignores, err = InitBrew(".", force, verbose)
		default:
			return fmt.Errorf("unknown language: %s", langName)
		}
		if err != nil {
			return err
		}
		allIgnores = append(allIgnores, ignores...)
	}

	if err := WriteIgnores(allIgnores, verbose); err != nil {
		return err
	}
	return nil
}

func WriteIgnores(lines []string, verbose bool) error {
	lines = util.Unique(lines)
	if _, err := os.Stat(".gitignore"); err == nil {
		data, err := os.ReadFile(".gitignore")
		if err != nil {
			return fmt.Errorf("reading .gitignore: %w", err)
		}
		content := string(data)
		existing := strings.Split(content, "\n")
		added := false
		for _, line := range lines {
			found := false
			for _, el := range existing {
				if strings.TrimSpace(el) == line {
					found = true
					break
				}
			}
			if !found {
				f, err := os.OpenFile(".gitignore", os.O_APPEND|os.O_WRONLY, 0644)
				if err != nil {
					return fmt.Errorf("appending to .gitignore: %w", err)
				}
				if _, err := f.WriteString(line + "\n"); err != nil {
					f.Close()
					return fmt.Errorf("writing .gitignore: %w", err)
				}
				f.Close()
				added = true
			}
		}
		if added && verbose {
			fx.Println("  {success}Updated .gitignore{@}")
		} else if verbose {
			fx.Println("  {warning}Skipped .gitignore{@}")
		}
	} else {
		content := strings.Join(lines, "\n") + "\n"
		if err := os.WriteFile(".gitignore", []byte(content), 0644); err != nil {
			return fmt.Errorf("creating .gitignore: %w", err)
		}
		if verbose {
			fx.Println("  {success}Created .gitignore{@}")
		}
	}
	return nil
}
