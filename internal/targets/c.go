package targets

import (
	"path/filepath"

	"github.com/grimdork/creo/internal/fiat"
)

func setCVarDefaults(f *fiat.File) {
	setDefaultVar(f.Vars, "CC", "cc")
	setDefaultVar(f.Vars, "CFLAGS", "-O2 -Wall")
	setDefaultVar(f.Vars, "CDEBUGFLAGS", "-O0 -g -Wall")
	if _, ok := f.Vars["CXX"]; !ok {
		if _, ok2 := f.Vars["CPP"]; ok2 {
			f.Vars["CXX"] = f.Vars["CPP"]
		} else {
			setDefaultVar(f.Vars, "CXX", "c++")
		}
	}
	if _, ok := f.Vars["CPP"]; !ok {
		f.Vars["CPP"] = f.Vars["CXX"]
	}
	if _, ok := f.Vars["CXXFLAGS"]; !ok {
		if _, ok2 := f.Vars["CPPFLAGS"]; ok2 {
			f.Vars["CXXFLAGS"] = f.Vars["CPPFLAGS"]
		} else {
			setDefaultVar(f.Vars, "CXXFLAGS", "-O2 -Wall")
		}
	}
	if _, ok := f.Vars["CPPFLAGS"]; !ok {
		f.Vars["CPPFLAGS"] = f.Vars["CXXFLAGS"]
	}
	if _, ok := f.Vars["CXXDEBUGFLAGS"]; !ok {
		if _, ok2 := f.Vars["CPPDEBUGFLAGS"]; ok2 {
			f.Vars["CXXDEBUGFLAGS"] = f.Vars["CPPDEBUGFLAGS"]
		} else {
			setDefaultVar(f.Vars, "CXXDEBUGFLAGS", "-O0 -g -Wall")
		}
	}
	if _, ok := f.Vars["CPPDEBUGFLAGS"]; !ok {
		f.Vars["CPPDEBUGFLAGS"] = f.Vars["CXXDEBUGFLAGS"]
	}
	setDefaultVar(f.Vars, "LDFLAGS", "")
	setDefaultVar(f.Vars, "LIBS", "")
}

func applyC(f *fiat.File, t *fiat.Target) {
	setCVarDefaults(f)

	absDir := absDir(f)

	proj := filepath.Base(absDir)
	setDefaultVar(f.Vars, "PROJECT", proj)

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

// CrossEnv returns environment variable overrides for cross-compilation based on the language, architecture and OS.
func CrossEnv(lang, arch, osval string) []string {
	switch lang {
	case LangGo, LangTinyGo:
		var env []string
		if arch != "" {
			env = append(env, "GOARCH="+arch)
		}
		if osval != "" {
			env = append(env, "GOOS="+osval)
		}
		return env
	case LangRust:
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

	absDir := absDir(f)

	proj := filepath.Base(absDir)
	setDefaultVar(f.Vars, "PROJECT", proj)

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
