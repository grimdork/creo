package targets

import (
	"path/filepath"

	"github.com/grimdork/creo/internal/fiat"
)

// InitCxx scaffolds a C++ project with a basic main.cpp and fiat file.
func InitCxx(dir string, force, verbose bool) ([]string, error) {
	file, err := ensureFiat(dir)
	if err != nil {
		return nil, err
	}

	if fiat.FindTarget(file, "build") == nil {
		bt := &fiat.Target{
			Name:     "build",
			Language: "cxx",
			Desc:     "Build the C++ binary",
		}
		file.AddTarget(bt)
	}

	mainContent := `#include <iostream>

int main(int argc, char **argv) {
	std::cout << "hello" << std::endl;
	return 0;
}
`
	if err := tryWrite(filepath.Join(dir, "main.cpp"), mainContent,
		force, verbose, "main.cpp"); err != nil {
		return nil, err
	}

	if err := file.Write(); err != nil {
		return nil, err
	}

	return []string{"/build", "/.creo"}, nil
}
