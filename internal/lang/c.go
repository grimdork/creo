package lang

import (
	"path/filepath"

	"github.com/grimdork/creo/internal/fiat"
)

func setCVarDefaults(f *fiat.File) {
	if _, ok := f.Vars["CC"]; !ok {
		f.Vars["CC"] = &fiat.Var{Name: "CC", Value: "cc"}
	}
	if _, ok := f.Vars["CFLAGS"]; !ok {
		f.Vars["CFLAGS"] = &fiat.Var{Name: "CFLAGS", Value: "-O2 -Wall"}
	}
	if _, ok := f.Vars["CDEBUGFLAGS"]; !ok {
		f.Vars["CDEBUGFLAGS"] = &fiat.Var{Name: "CDEBUGFLAGS", Value: "-O0 -g -Wall"}
	}
	if _, ok := f.Vars["CXX"]; !ok {
		if _, ok2 := f.Vars["CPP"]; ok2 {
			f.Vars["CXX"] = f.Vars["CPP"]
		} else {
			f.Vars["CXX"] = &fiat.Var{Name: "CXX", Value: "c++"}
		}
	}
	if _, ok := f.Vars["CPP"]; !ok {
		f.Vars["CPP"] = f.Vars["CXX"]
	}
	if _, ok := f.Vars["CXXFLAGS"]; !ok {
		if _, ok2 := f.Vars["CPPFLAGS"]; ok2 {
			f.Vars["CXXFLAGS"] = f.Vars["CPPFLAGS"]
		} else {
			f.Vars["CXXFLAGS"] = &fiat.Var{Name: "CXXFLAGS", Value: "-O2 -Wall"}
		}
	}
	if _, ok := f.Vars["CPPFLAGS"]; !ok {
		f.Vars["CPPFLAGS"] = f.Vars["CXXFLAGS"]
	}
	if _, ok := f.Vars["CXXDEBUGFLAGS"]; !ok {
		if _, ok2 := f.Vars["CPPDEBUGFLAGS"]; ok2 {
			f.Vars["CXXDEBUGFLAGS"] = f.Vars["CPPDEBUGFLAGS"]
		} else {
			f.Vars["CXXDEBUGFLAGS"] = &fiat.Var{Name: "CXXDEBUGFLAGS", Value: "-O0 -g -Wall"}
		}
	}
	if _, ok := f.Vars["CPPDEBUGFLAGS"]; !ok {
		f.Vars["CPPDEBUGFLAGS"] = f.Vars["CXXDEBUGFLAGS"]
	}
	if _, ok := f.Vars["LDFLAGS"]; !ok {
		f.Vars["LDFLAGS"] = &fiat.Var{Name: "LDFLAGS", Value: ""}
	}
	if _, ok := f.Vars["LIBS"]; !ok {
		f.Vars["LIBS"] = &fiat.Var{Name: "LIBS", Value: ""}
	}
}

func applyC(f *fiat.File, t *fiat.Target) {
	setCVarDefaults(f)

	dir := filepath.Dir(f.Path())
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}

	proj := filepath.Base(absDir)
	if _, ok := f.Vars["PROJECT"]; !ok {
		f.Vars["PROJECT"] = &fiat.Var{Name: "PROJECT", Value: proj}
	}

	bd := BuildDir(f)
	defBin := bd + "/" + proj
	if isDebug(t) {
		defBin += "-debug"
	}
	t.Bin = expandBin(f, t, defBin)

	if t.Sources == "" {
		t.Sources = "*.c"
	}
	if len(t.Cmds) == 0 {
		flags := "$CFLAGS"
		if isDebug(t) {
			flags = "$CDEBUGFLAGS"
		}
		t.Cmds = append(t.Cmds, "mkdir -p $BUILDDIR && $CC $args "+flags+" $LDFLAGS -o $bin $sources $LIBS")
	}
}

func CrossEnv(lang, arch, osval string) []string {
	switch lang {
	case "go":
		var env []string
		if arch != "" {
			env = append(env, "GOARCH="+arch)
		}
		if osval != "" {
			env = append(env, "GOOS="+osval)
		}
		return env
	case "rust":
		if triple := rustTriple(arch, osval); triple != "" {
			return []string{"CARGO_BUILD_TARGET=" + triple}
		}
		return nil
	default:
		return nil
	}
}

func applyCxx(f *fiat.File, t *fiat.Target) {
	setCVarDefaults(f)

	dir := filepath.Dir(f.Path())
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}

	proj := filepath.Base(absDir)
	if _, ok := f.Vars["PROJECT"]; !ok {
		f.Vars["PROJECT"] = &fiat.Var{Name: "PROJECT", Value: proj}
	}

	bd := BuildDir(f)
	defBin := bd + "/" + proj
	if isDebug(t) {
		defBin += "-debug"
	}
	t.Bin = expandBin(f, t, defBin)

	if t.Sources == "" {
		t.Sources = "*.cpp"
	}
	if len(t.Cmds) == 0 {
		flags := "$CXXFLAGS"
		if isDebug(t) {
			flags = "$CXXDEBUGFLAGS"
		}
		t.Cmds = append(t.Cmds, "mkdir -p $BUILDDIR && $CXX $args "+flags+" $LDFLAGS -o $bin $sources $LIBS")
	}
}
