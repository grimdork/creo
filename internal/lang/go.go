package lang

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/grimdork/creo/internal/fiat"
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

func applyGo(f *fiat.File, t *fiat.Target) {
	dir := filepath.Dir(f.Path())
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}

	proj := ModuleName(absDir)
	if _, ok := f.Vars["PROJECT"]; !ok {
		f.Vars["PROJECT"] = &fiat.Var{Name: "PROJECT", Value: proj}
	}

	if _, ok := f.Vars["GO"]; !ok {
		f.Vars["GO"] = &fiat.Var{Name: "GO", Value: "go build"}
	}
	if _, ok := f.Vars["GODEBUGFLAGS"]; !ok {
		f.Vars["GODEBUGFLAGS"] = &fiat.Var{Name: "GODEBUGFLAGS", Value: `-gcflags="all=-N -l"`}
	}

	_, hasGoFlags := f.Vars["GOFLAGS"]

	defBin := "./" + proj
	if isDebug(t) {
		defBin += "-debug"
	}
	t.Bin = expandBin(f, t, defBin)

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
				verPost = ` -ldflags="-buildid=reproducible -X main.version=$VERSION"`
			} else {
				flags = `-trimpath -ldflags="-s -w -buildid=reproducible -X main.version=$VERSION"`
			}
		}
		pkg := ""
		if srcDir != "" {
			pkg = " " + srcDir
		}
		t.Cmds = append(t.Cmds, "$GO $args -buildvcs=false "+flags+verPost+" -o $bin"+pkg)
	}
}
