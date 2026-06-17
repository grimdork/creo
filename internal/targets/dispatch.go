package targets

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/grimdork/creo/internal/fiat"
	"github.com/grimdork/creo/internal/semver"
)

// BuildDir returns the effective build output directory from $BUILDDIR or the default "build".
func BuildDir(f *fiat.File) string {
	if v, ok := f.Vars["BUILDDIR"]; ok && v.Value != "" {
		return v.Value
	}
	return "build"
}

func initBuildDir(f *fiat.File) {
	setDefaultVar(f.Vars, "BUILDDIR", "build")
}

func isDebug(t *fiat.Target) bool {
	return t.Name == "debug" || strings.HasSuffix(t.Name, "-debug")
}

func absDir(f *fiat.File) string {
	dir := filepath.Dir(f.Path())
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return dir
	}
	return absDir
}

func setDefaultVar(vars map[string]*fiat.Var, name, value string) {
	if _, ok := vars[name]; !ok {
		vars[name] = &fiat.Var{Name: name, Value: value}
	}
}

// Apply runs language/target-type defaults on every target in a parsed fiat file.
func Apply(f *fiat.File) error {
	initBuildDir(f)
	setDefaultVar(f.Vars, "VERSION", semver.String())
	setDefaultVar(f.Vars, "COMMIT", semver.CommitString())
	setDefaultVar(f.Vars, "DATE", semver.DateString())

	for _, t := range f.Targets {
		if t.IsVirtual {
			continue
		}
		switch t.Language {
		case "":
			continue
		case LangGo:
			applyGo(f, t)
		case LangTinyGo:
			applyTinyGo(f, t)
		case LangC:
			applyC(f, t)
		case LangCxx, LangCpp:
			applyCxx(f, t)
		case LangRust:
			applyRust(f, t)
		case LangPython:
			applyPython(f, t)
		case LangNode, LangTS:
			applyNode(f, t)
		case LangJava, LangKotlin, LangGradle:
			applyJava(f, t)
		case LangOCI:
			applyOci(f, t)
		case LangArchive:
			applyArchive(f, t)
		case LangDeb:
			applyDeb(f, t)
		case LangRpm:
			applyRpm(f, t)
		case LangBrew:
			applyBrew(f, t)
		default:
			return fmt.Errorf(errUnknownLang, f.Path(), t.Language)
		}
	}
	return nil
}

func expandBin(f *fiat.File, t *fiat.Target, defBin string) string {
	if t.Bin == "" {
		return defBin
	}
	ev := fiat.MergeVars(f.Vars, t.Vars)
	ev["bin"] = &fiat.Var{Name: "bin", Value: defBin}
	if !(len(t.Arch) > 1 || len(t.OS) > 1) {
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
	return fiat.Expand(t.Bin, ev, 0)
}
