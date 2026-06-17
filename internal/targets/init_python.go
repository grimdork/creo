package targets

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/grimdork/creo/internal/fiat"
)

// InitPython scaffolds a Python project with pyproject.toml and a basic package.
func InitPython(dir string, force, verbose bool) ([]string, error) {
	file, err := ensureFiat(dir)
	if err != nil {
		return nil, err
	}

	if fiat.FindTarget(file, "build") == nil {
		bt := &fiat.Target{
			Name:     "build",
			Language: "python",
			Desc:     "Sync Python dependencies and prepare source",
		}
		file.AddTarget(bt)
	}

	_, proj := absDirName(dir)

	pyproject := `[project]
name = "` + proj + `"
version = "0.1.0"
requires-python = ">=3.11"
dependencies = []

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
`
	if err := tryWrite(filepath.Join(dir, "pyproject.toml"), pyproject,
		force, verbose, "pyproject.toml"); err != nil {
		return nil, err
	}

	srcDir := filepath.Join(dir, "src", proj)
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		return nil, fmt.Errorf(errCreating, "src/"+proj, err)
	}

	initContent := `def main() -> None:
    print("hello from ` + proj + `")


if __name__ == "__main__":
    main()
`
	initPath := filepath.Join(srcDir, "main.py")
	if err := tryWrite(initPath, initContent, force, verbose, "src/"+proj+"/main.py"); err != nil {
		return nil, err
	}

	emptyInit := ""
	emptyInitPath := filepath.Join(srcDir, "__init__.py")
	if err := tryWrite(emptyInitPath, emptyInit, force, verbose, "src/"+proj+"/__init__.py"); err != nil {
		return nil, err
	}

	if err := file.Write(); err != nil {
		return nil, err
	}

	return []string{"__pycache__/", "*.pyc", "dist/", "*.egg-info/", ".venv/", "/.creo"}, nil
}
