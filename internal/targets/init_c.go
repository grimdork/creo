package targets

import (
	"path/filepath"

	"github.com/grimdork/creo/internal/fiat"
)

// InitC scaffolds a C project with a basic main.c and fiat file.
func InitC(dir string, force, verbose bool) ([]string, error) {
	file, err := ensureFiat(dir)
	if err != nil {
		return nil, err
	}

	if fiat.FindTarget(file, "build") == nil {
		bt := &fiat.Target{
			Name:     "build",
			Language: "c",
			Desc:     "Build the C binary",
		}
		file.AddTarget(bt)
	}

	mainContent := `#include <stdio.h>

int main(int argc, char **argv) {
	printf("hello\n");
	return 0;
}
`
	if err := tryWrite(filepath.Join(dir, "main.c"), mainContent,
		force, verbose, "main.c"); err != nil {
		return nil, err
	}

	if err := file.Write(); err != nil {
		return nil, err
	}

	return []string{"/build", "/.creo"}, nil
}
