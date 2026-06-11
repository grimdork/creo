package lang

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/grimdork/creo/internal/fiat"
	"github.com/grimdork/creo/internal/semver"
)

func isDebug(t *fiat.Target) bool {
	return t.Name == "debug" || strings.HasSuffix(t.Name, "-debug")
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
		case "rust":
			applyRust(f, t)
		case "oci":
			applyOci(f, t)
		default:
			return fmt.Errorf("%s: unknown language %q", f.Path(), t.Language)
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
