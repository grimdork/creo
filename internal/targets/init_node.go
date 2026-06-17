package targets

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/grimdork/creo/internal/fiat"
)

// InitNode scaffolds a Node/TypeScript project with package.json, tsconfig and a basic source file.
func InitNode(dir string, force, verbose bool) ([]string, error) {
	file, err := ensureFiat(dir)
	if err != nil {
		return nil, err
	}

	if fiat.FindTarget(file, "build") == nil {
		bt := &fiat.Target{
			Name:     "build",
			Language: "node",
			Desc:     "Build the Node/TypeScript project",
		}
		file.AddTarget(bt)
	}

	_, proj := absDirName(dir)

	pkgJSON := `{
  "name": "` + proj + `",
  "version": "0.1.0",
  "private": true,
  "main": "dist/index.js",
  "scripts": {
    "build": "tsc"
  },
  "devDependencies": {
    "typescript": "^5.0.0"
  }
}
`
	if err := tryWrite(filepath.Join(dir, "package.json"), pkgJSON,
		force, verbose, "package.json"); err != nil {
		return nil, err
	}

	tsconfig := `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "NodeNext",
    "moduleResolution": "nodenext",
    "outDir": "dist",
    "rootDir": "src",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true
  },
  "include": ["src/**/*"]
}
`
	if err := tryWrite(filepath.Join(dir, "tsconfig.json"), tsconfig,
		force, verbose, "tsconfig.json"); err != nil {
		return nil, err
	}

	srcDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		return nil, fmt.Errorf("creating src/: %w", err)
	}

	indexContent := `const greeting: string = "hello from ` + proj + `";
console.log(greeting);
`
	if err := tryWrite(filepath.Join(srcDir, "index.ts"), indexContent,
		force, verbose, "src/index.ts"); err != nil {
		return nil, err
	}

	if err := file.Write(); err != nil {
		return nil, err
	}

	return []string{"node_modules/", "dist/", "/.creo"}, nil
}
