package targets

import (
	"path/filepath"

	"github.com/grimdork/creo/internal/fiat"
)

func applyTinyGo(f *fiat.File, t *fiat.Target) {
	dir := filepath.Dir(f.Path())
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}

	proj := ModuleName(absDir)
	if _, ok := f.Vars["PROJECT"]; !ok {
		f.Vars["PROJECT"] = &fiat.Var{Name: "PROJECT", Value: proj}
	}

	if _, ok := f.Vars["TINYGO"]; !ok {
		f.Vars["TINYGO"] = &fiat.Var{Name: "TINYGO", Value: "tinygo build"}
	}
	if _, ok := f.Vars["TINYGOFLAGS"]; !ok {
		f.Vars["TINYGOFLAGS"] = &fiat.Var{Name: "TINYGOFLAGS", Value: "-no-debug -panic=trap -scheduler=none"}
	}

	bd := BuildDir(f)
	defBin := bd + "/" + proj
	t.Bin = expandBin(f, t, defBin)

	if t.Sources == "" {
		t.Sources = "*.go go.mod go.sum"
	}

	if len(t.Cmds) == 0 {
		t.Cmds = append(t.Cmds, "$TINYGO $TINYGOFLAGS -o $bin")
	}
}
