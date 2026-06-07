package lang

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/grimdork/creo/internal/semver"
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

func Apply(f *FiatFile) {
	dir := filepath.Dir(f.Path)
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}

	if _, ok := f.Vars["GO"]; !ok {
		f.Vars["GO"] = &Var{Name: "GO", Value: "go build"}
	}
	if _, ok := f.Vars["GODEBUGFLAGS"]; !ok {
		f.Vars["GODEBUGFLAGS"] = &Var{Name: "GODEBUGFLAGS", Value: `-gcflags="all=-N -l"`}
	}
	if _, ok := f.Vars["VERSION"]; !ok {
		f.Vars["VERSION"] = &Var{Name: "VERSION", Value: semver.String()}
	}

	_, hasGoFlags := f.Vars["GOFLAGS"]
	for _, t := range f.Targets {
		if t.Language != "go" {
			continue
		}
		isDebug := t.Name == "debug" || strings.HasSuffix(t.Name, "-debug")
		defBin := "./" + ModuleName(absDir)
		if isDebug {
			defBin += "-debug"
		}
		if t.Bin == "" {
			t.Bin = defBin
		} else {
			ev := make(map[string]*Var)
			for k, v := range f.Vars {
				ev[k] = v
			}
			for _, v := range t.Vars {
				ev[v.Name] = v
			}
			ev["bin"] = &Var{Name: "bin", Value: defBin}
			if len(t.Arch) > 1 || len(t.OS) > 1 {
				// Multi-arch/os: only $bin expanded; $os/$arch filled per combination in runner
			} else {
				if len(t.OS) > 0 {
					ev["os"] = &Var{Name: "os", Value: t.OS[0]}
				} else {
					ev["os"] = &Var{Name: "os", Value: runtime.GOOS}
				}
				if len(t.Arch) > 0 {
					ev["arch"] = &Var{Name: "arch", Value: t.Arch[0]}
				} else {
					ev["arch"] = &Var{Name: "arch", Value: runtime.GOARCH}
				}
			}
			t.Bin = Expand(t.Bin, ev, 0)
		}
		if t.Sources == "" {
			t.Sources = "*.go"
		}
		if len(t.Cmds) == 0 && len(t.Install) == 0 {
			flags := "$GOFLAGS"
			verPost := ""
			if !hasGoFlags {
				if isDebug {
					flags = "$GODEBUGFLAGS"
					verPost = ` -ldflags="-X main.version=$VERSION"`
				} else {
					flags = `-trimpath -ldflags="-s -w -X main.version=$VERSION"`
				}
			}
			t.Cmds = append(t.Cmds, "$GO "+flags+verPost+" -o $bin")
		}
	}
}
