package targets

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/grimdork/creo/internal/fiat"
)

// NodeProjectName reads the project name from package.json.
func NodeProjectName(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return filepath.Base(dir)
	}
	var name string
	inKey := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, `"name"`) {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				name = strings.TrimSpace(parts[1])
				name = strings.Trim(name, `",`)
				if idx := strings.IndexByte(name, '@'); idx >= 0 && strings.HasPrefix(name, "@") {
					scopeParts := strings.SplitN(name, "/", 2)
					if len(scopeParts) == 2 {
						name = scopeParts[1]
					}
				}
				inKey = true
				break
			}
		}
	}
	if inKey && name != "" {
		return name
	}
	return filepath.Base(dir)
}

func detectPackageManager(dir string) string {
	if _, err := os.Stat(filepath.Join(dir, "pnpm-lock.yaml")); err == nil {
		return "pnpm"
	}
	if _, err := os.Stat(filepath.Join(dir, "yarn.lock")); err == nil {
		return "yarn"
	}
	return "npm"
}

func applyNode(f *fiat.File, t *fiat.Target) {
	absDir := absDir(f)

	proj := NodeProjectName(absDir)
	if _, ok := f.Vars["PROJECT"]; !ok {
		f.Vars["PROJECT"] = &fiat.Var{Name: "PROJECT", Value: proj}
	}

	pm := detectPackageManager(absDir)
	pmVar := strings.ToUpper(pm)
	if _, ok := f.Vars[pmVar]; !ok {
		f.Vars[pmVar] = &fiat.Var{Name: pmVar, Value: pm}
	}

	if t.Sources == "" {
		t.Sources = "*.js *.jsx *.ts *.tsx package.json tsconfig.json"
	}

	t.Bin = expandBin(f, t, "dist")

	if len(t.Cmds) == 0 {
		t.Cmds = append(t.Cmds, "$"+pmVar+" run build")
	}
}
