package cli

import "github.com/grimdork/creo/internal/fiat"

// InjectBuildDir sets the BUILDDIR variable in the fiat file.
func InjectBuildDir(f *fiat.File, bd string) {
	f.Vars["BUILDDIR"] = &fiat.Var{Name: "BUILDDIR", Value: bd}
}
