package targets

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/grimdork/creo/internal/fiat"
)

// ProjectName reads the project name from pyproject.toml.
func ProjectName(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	if err != nil {
		return filepath.Base(dir)
	}
	inProject := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "[project]" {
			inProject = true
			continue
		}
		if inProject {
			if strings.HasPrefix(line, "[") {
				break
			}
			if strings.HasPrefix(line, "name") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					name := strings.TrimSpace(parts[1])
					name = strings.Trim(name, "'\"")
					return name
				}
			}
		}
	}
	return filepath.Base(dir)
}

func applyPython(f *fiat.File, t *fiat.Target) {
	dir := filepath.Dir(f.Path())
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}

	proj := ProjectName(absDir)
	if _, ok := f.Vars["PROJECT"]; !ok {
		f.Vars["PROJECT"] = &fiat.Var{Name: "PROJECT", Value: proj}
	}
	if _, ok := f.Vars["PYTHON"]; !ok {
		f.Vars["PYTHON"] = &fiat.Var{Name: "PYTHON", Value: "python3"}
	}
	if _, ok := f.Vars["UV"]; !ok {
		f.Vars["UV"] = &fiat.Var{Name: "UV", Value: "uv"}
	}

	if t.Sources == "" {
		t.Sources = "*.py pyproject.toml setup.py setup.cfg"
	}

	t.Bin = expandBin(f, t, "src")

	if len(t.Cmds) == 0 {
		t.Cmds = append(t.Cmds, "$UV sync --frozen")
	}
}
