package targets

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/grimdork/creo/internal/fiat"
	"github.com/grimdork/creo/internal/semver"
)

func BuildDir(f *fiat.File) string {
	if v, ok := f.Vars["BUILDDIR"]; ok && v.Value != "" {
		return v.Value
	}
	return "build"
}

func initBuildDir(f *fiat.File) {
	if _, ok := f.Vars["BUILDDIR"]; !ok {
		f.Vars["BUILDDIR"] = &fiat.Var{Name: "BUILDDIR", Value: "build"}
	}
}

func isDebug(t *fiat.Target) bool {
	return t.Name == "debug" || strings.HasSuffix(t.Name, "-debug")
}

func Apply(f *fiat.File) error {
	initBuildDir(f)
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
	ev := make(map[string]*fiat.Var)
	for k, v := range f.Vars {
		ev[k] = v
	}
	for _, v := range t.Vars {
		ev[v.Name] = v
	}
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
