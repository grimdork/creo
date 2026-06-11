package lang

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/grimdork/creo/internal/fiat"
)

func CrateName(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "Cargo.toml"))
	if err != nil {
		return filepath.Base(dir)
	}
	inPackage := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "[package]" {
			inPackage = true
			continue
		}
		if inPackage {
			if strings.HasPrefix(line, "[") {
				break
			}
			if strings.HasPrefix(line, "name") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					name := strings.TrimSpace(parts[1])
					name = strings.Trim(name, "'\"")
					return name
				}
			}
		}
	}
	return filepath.Base(dir)
}

func rustTriple(arch, os string) string {
	a := strings.ToLower(arch)
	o := strings.ToLower(os)

	var triple string
	switch a {
	case "amd64", "x86_64":
		triple = "x86_64"
	case "arm64", "aarch64":
		triple = "aarch64"
	case "arm":
		triple = "armv7"
	default:
		return ""
	}

	switch o {
	case "linux":
		if a == "arm" {
			triple += "-unknown-linux-gnueabihf"
		} else {
			triple += "-unknown-linux-gnu"
		}
	case "darwin", "macos":
		triple += "-apple-darwin"
	case "freebsd":
		triple += "-unknown-freebsd"
	case "windows":
		triple += "-pc-windows-msvc"
	default:
		return ""
	}

	return triple
}

func applyRust(f *fiat.File, t *fiat.Target) {
	dir := filepath.Dir(f.Path())
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}

	crateName := CrateName(absDir)

	if _, ok := f.Vars["CARGO"]; !ok {
		f.Vars["CARGO"] = &fiat.Var{Name: "CARGO", Value: "cargo"}
	}

	if _, ok := f.Vars["PROJECT"]; !ok {
		f.Vars["PROJECT"] = &fiat.Var{Name: "PROJECT", Value: crateName}
	}

	bd := buildDir(f)
	defBin := bd + "/release/" + crateName
	if isDebug(t) {
		defBin = bd + "/debug/" + crateName
	}
	t.Bin = expandBin(f, t, defBin)

	if t.Sources == "" {
		t.Sources = "*.rs Cargo.toml Cargo.lock"
	}

	if len(t.Tmp) == 0 {
		t.Tmp = []string{"target"}
	}

	if len(t.Cmds) == 0 {
		release := " --release"
		cargoDir := "target/release"
		if isDebug(t) {
			release = ""
			cargoDir = "target/debug"
		}
		t.Cmds = append(t.Cmds, "$CARGO build"+release+" $args 2>&1 && cp "+cargoDir+"/$PROJECT $bin")
	}
}
