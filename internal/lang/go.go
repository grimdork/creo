package lang

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/grimdork/creo/internal/fiat"
	"github.com/grimdork/creo/internal/semver"
)

func ModuleName(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return filepath.Base(dir)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			mod := strings.TrimSpace(strings.TrimPrefix(line, "module "))
			if idx := strings.LastIndexByte(mod, '/'); idx >= 0 {
				return mod[idx+1:]
			}
			return mod
		}
	}
	return filepath.Base(dir)
}

func Apply(f *fiat.File) error {
	if _, ok := f.Vars["VERSION"]; !ok {
		f.Vars["VERSION"] = &fiat.Var{Name: "VERSION", Value: semver.String()}
	}
	if _, ok := f.Vars["COMMIT"]; !ok {
		f.Vars["COMMIT"] = &fiat.Var{Name: "COMMIT", Value: semver.CommitString()}
	}
	if _, ok := f.Vars["DATE"]; !ok {
		f.Vars["DATE"] = &fiat.Var{Name: "DATE", Value: semver.DateString()}
	}

	for _, t := range f.Targets {
		if t.IsVirtual {
			continue
		}
		switch t.Language {
		case "":
			continue
		case "go":
			applyGo(f, t)
		case "c":
			applyC(f, t)
		case "cxx", "cpp":
			applyCxx(f, t)
		case "oci":
			applyOci(f, t)
		default:
			return fmt.Errorf("%s: unknown language %q", f.Path(), t.Language)
		}
	}
	return nil
}

func applyGo(f *fiat.File, t *fiat.Target) {
	dir := filepath.Dir(f.Path())
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}

	if _, ok := f.Vars["GO"]; !ok {
		f.Vars["GO"] = &fiat.Var{Name: "GO", Value: "go build"}
	}
	if _, ok := f.Vars["GODEBUGFLAGS"]; !ok {
		f.Vars["GODEBUGFLAGS"] = &fiat.Var{Name: "GODEBUGFLAGS", Value: `-gcflags="all=-N -l"`}
	}

	_, hasGoFlags := f.Vars["GOFLAGS"]

	defBin := "./" + ModuleName(absDir)
	if isDebug(t) {
		defBin += "-debug"
	}
	if t.Bin == "" {
		t.Bin = defBin
	} else {
		ev := make(map[string]*fiat.Var)
		for k, v := range f.Vars {
			ev[k] = v
		}
		for _, v := range t.Vars {
			ev[v.Name] = v
		}
		ev["bin"] = &fiat.Var{Name: "bin", Value: defBin}
		if len(t.Arch) > 1 || len(t.OS) > 1 {
		} else {
			if len(t.OS) > 0 {
				ev["os"] = &fiat.Var{Name: "os", Value: t.OS[0]}
			} else {
				ev["os"] = &fiat.Var{Name: "os", Value: runtime.GOOS}
			}
			if len(t.Arch) > 0 {
				ev["arch"] = &fiat.Var{Name: "arch", Value: t.Arch[0]}
			} else {
				ev["arch"] = &fiat.Var{Name: "arch", Value: runtime.GOARCH}
			}
		}
		t.Bin = fiat.Expand(t.Bin, ev, 0)
	}
	srcDir := ""
	for _, v := range t.Vars {
		if v.Name == "SRCDIR" {
			srcDir = v.Value
			break
		}
	}
	if v, ok := f.Vars["SRCDIR"]; ok && srcDir == "" {
		srcDir = v.Value
	}

	if t.Sources == "" {
		if srcDir != "" {
			t.Sources = srcDir + "/*.go"
		} else {
			t.Sources = "*.go go.mod go.sum"
		}
	}
	if len(t.Cmds) == 0 {
		flags := "$GOFLAGS"
		verPost := ""
		if !hasGoFlags {
			if isDebug(t) {
				flags = "$GODEBUGFLAGS"
				verPost = ` -ldflags="-X main.version=$VERSION"`
			} else {
				flags = `-trimpath -ldflags="-s -w -X main.version=$VERSION"`
			}
		}
		pkg := ""
		if srcDir != "" {
			pkg = " " + srcDir
		}
		t.Cmds = append(t.Cmds, "$GO $args "+flags+verPost+" -o $bin"+pkg)
	}
}
