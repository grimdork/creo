package lang

import (
	"path/filepath"
	"strings"

	"github.com/grimdork/creo/internal/fiat"
)

func isDebug(t *fiat.Target) bool {
	return t.Name == "debug" || strings.HasSuffix(t.Name, "-debug")
}

func setVarDefaults(f *fiat.File) {
	if _, ok := f.Vars["CC"]; !ok {
		f.Vars["CC"] = &fiat.Var{Name: "CC", Value: "cc"}
	}
	if _, ok := f.Vars["CFLAGS"]; !ok {
		f.Vars["CFLAGS"] = &fiat.Var{Name: "CFLAGS", Value: "-O2 -Wall"}
	}
	if _, ok := f.Vars["CDEBUGFLAGS"]; !ok {
		f.Vars["CDEBUGFLAGS"] = &fiat.Var{Name: "CDEBUGFLAGS", Value: "-O0 -g -Wall"}
	}
	if _, ok := f.Vars["LDFLAGS"]; !ok {
		f.Vars["LDFLAGS"] = &fiat.Var{Name: "LDFLAGS", Value: ""}
	}
	if _, ok := f.Vars["LIBS"]; !ok {
		f.Vars["LIBS"] = &fiat.Var{Name: "LIBS", Value: ""}
	}
}

func applyC(f *fiat.File, t *fiat.Target) {
	setVarDefaults(f)

	dir := filepath.Dir(f.Path())
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}
	if t.Bin == "" {
		t.Bin = "./" + filepath.Base(absDir)
	}
	if isDebug(t) {
		t.Bin += "-debug"
	}
	if t.Sources == "" {
		t.Sources = "*.c *.h"
	}
	if len(t.Cmds) == 0 && len(t.Install) == 0 {
		flags := "$CFLAGS"
		if isDebug(t) {
			flags = "$CDEBUGFLAGS"
		}
		t.Cmds = append(t.Cmds, "$CC $args "+flags+" $LDFLAGS -o $bin $sources $LIBS")
	}
}

func setCxxVarDefaults(f *fiat.File) {
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
	default:
		return nil
	}
}

func applyCxx(f *fiat.File, t *fiat.Target) {
	setCxxVarDefaults(f)

	dir := filepath.Dir(f.Path())
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}
	if t.Bin == "" {
		t.Bin = "./" + filepath.Base(absDir)
	}
	if isDebug(t) {
		t.Bin += "-debug"
	}
	if t.Sources == "" {
		t.Sources = "*.cpp *.hpp *.hxx *.hh *.cppm *.ixx *.mpp"
	}
	if len(t.Cmds) == 0 && len(t.Install) == 0 {
		flags := "$CXXFLAGS"
		if isDebug(t) {
			flags = "$CXXDEBUGFLAGS"
		}
		t.Cmds = append(t.Cmds, "$CXX $args "+flags+" $LDFLAGS -o $bin $sources $LIBS")
	}
}
