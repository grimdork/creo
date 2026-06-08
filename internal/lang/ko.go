package lang

import (
	"path/filepath"
	"strings"
)

func applyKo(f *FiatFile, t *Target) {
	if _, ok := f.Vars["KO"]; !ok {
		f.Vars["KO"] = &Var{Name: "KO", Value: "ko build"}
	}

	dir := filepath.Dir(f.Path)
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}
	name := ModuleName(absDir)
	if name == "" {
		name = filepath.Base(absDir)
	}

	srcDir := ""
	for _, v := range t.Vars {
		if v.Name == "SRCDIR" {
			srcDir = v.Value
			break
		}
	}
	if v, ok := f.Vars["SRCDIR"]; ok && srcDir == "" {
		srcDir = v.Value
	}

	if t.Sources == "" {
		if srcDir != "" {
			t.Sources = srcDir + "/*.go"
		} else {
			t.Sources = "*.go go.mod go.sum"
		}
	}

	if len(t.Cmds) == 0 {
		platform := "linux/amd64"
		if len(t.Arch) > 0 || len(t.OS) > 0 {
			var parts []string
			archs := t.Arch
			if len(archs) == 0 {
				archs = []string{""}
			}
			oses := t.OS
			if len(oses) == 0 {
				oses = []string{""}
			}
			for _, arch := range archs {
				for _, osval := range oses {
					a := arch
					o := osval
					if a == "" {
						a = "amd64"
					}
					if o == "" {
						o = "linux"
					}
					parts = append(parts, o+"/"+a)
				}
			}
			platform = strings.Join(parts, ",")
		}

		tarball := "build/" + name + ".tar"
		for _, v := range t.Vars {
			if v.Name == "TARBALL" {
				tarball = v.Value
				break
			}
		}

		pkg := "."
		if srcDir != "" {
			pkg = srcDir
		}

		if t.Bin == "" {
			t.Bin = tarball
		}

		t.Cmds = append(t.Cmds, "mkdir -p build && $KO $args --platform="+platform+" --tarball "+tarball+" --push=false "+pkg)
		t.Arch = nil
		t.OS = nil
	}
}
