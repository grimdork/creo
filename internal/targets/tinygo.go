package targets

import (
	"github.com/grimdork/creo/internal/fiat"
)

func applyTinyGo(f *fiat.File, t *fiat.Target) {
	absDir := absDir(f)

	proj := ModuleName(absDir)
	setDefaultVar(f.Vars, "PROJECT", proj)
	setDefaultVar(f.Vars, "TINYGO", "tinygo build")
	setDefaultVar(f.Vars, "TINYGOFLAGS", "-no-debug -panic=trap -scheduler=none")

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
