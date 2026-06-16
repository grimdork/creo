package targets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type tmplTest struct {
	lang       string
	name       string
	files      []string
	gitignore  []string
	checkGoMod bool
}

func TestInitWithTemplateAll(t *testing.T) {
	tests := []tmplTest{
		{lang: "go", name: "basic", files: []string{"main.go", "version.go", "fiat"}, checkGoMod: true},
		{lang: "go", name: "arg", files: []string{"main.go", "fiat"}, checkGoMod: true},
		{lang: "go", name: "toolcmd", files: []string{"main.go", "fiat"}, checkGoMod: true},
		{lang: "go", name: "web", files: []string{"main.go", "fiat", "Dockerfile"}, checkGoMod: true},
		{lang: "tinygo", name: "basic", files: []string{"main.go", "version.go", "fiat"}, checkGoMod: true},
		{lang: "c", name: "basic", files: []string{"main.c", "fiat"}},
		{lang: "cxx", name: "basic", files: []string{"main.cpp", "fiat"}},
		{lang: "cxx", name: "arg", files: []string{"main.cpp", "fiat"}},
		{lang: "cxx", name: "boost", files: []string{"main.cpp", "fiat"}},
		{lang: "cxx", name: "toolcmd", files: []string{"main.cpp", "fiat"}},
		{lang: "rust", name: "basic", files: []string{"main.rs", "Cargo.toml", "fiat"}},
		{lang: "rust", name: "arg", files: []string{"main.rs", "Cargo.toml", "fiat"}},
		{lang: "rust", name: "toolcmd", files: []string{"main.rs", "Cargo.toml", "fiat"}},
		{lang: "python", name: "cli", files: []string{"main.py", "pyproject.toml", "fiat"}},
		{lang: "python", name: "basic", files: []string{"pyproject.toml", "fiat", "src/app/main.py"}},
		{lang: "node", name: "basic", files: []string{"package.json", "tsconfig.json", "fiat", "src/index.ts"}},
		{lang: "java", name: "basic", files: []string{"settings.gradle.kts", "build.gradle.kts", "fiat",
			"src/main/kotlin/com/example/App.kt"}},
	}

	gitignoreBase := []string{"/build", "/.creo", "/tmp"}
	langIgnores := map[string][]string{
		"rust":   {"/target"},
		"python": {"__pycache__/", "*.pyc"},
		"node":   {"node_modules/"},
		"java":   {".gradle/", "*.jar"},
	}

	for _, tt := range tests {
		t.Run(tt.lang+"/"+tt.name, func(t *testing.T) {
			dir := t.TempDir()
			restore := chdir(t, dir)
			defer restore()

			err := InitProjectWithTemplate(tt.lang, tt.name, false, false)
			if err != nil {
				t.Fatalf("InitProjectWithTemplate(%q, %q): %v", tt.lang, tt.name, err)
			}

			for _, f := range tt.files {
				if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
					t.Errorf("missing %s: %v", f, err)
				}
			}

			fileData, err := os.ReadFile(filepath.Join(dir, "fiat"))
			if err != nil {
				t.Fatal(err)
			}
			content := string(fileData)
			if !strings.Contains(content, tt.lang) && tt.lang != "node" {
				t.Errorf("fiat should contain language %q", tt.lang)
			}

			if tt.checkGoMod {
				if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
					t.Errorf("missing go.mod: %v", err)
				}
			}

			checkGitignore(t, dir, gitignoreBase)
			if ig, ok := langIgnores[tt.lang]; ok {
				checkGitignore(t, dir, ig)
			}

			if _, err := os.Stat(filepath.Join(dir, ".gitignore")); err != nil {
				t.Errorf("missing .gitignore: %v", err)
			}
		})
	}
}

func checkGitignore(t *testing.T, dir string, patterns []string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}
	content := string(data)
	for _, p := range patterns {
		if !strings.Contains(content, p) {
			t.Errorf(".gitignore should contain %q", p)
		}
	}
}

func TestInitWithTemplateNotFound(t *testing.T) {
	dir := t.TempDir()
	restore := chdir(t, dir)
	defer restore()

	err := InitProjectWithTemplate("go", "nonexistent", false, false)
	if err == nil {
		t.Fatal("expected error for unknown template")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected error containing 'not found', got %q", err.Error())
	}
}

func TestInitWithTemplateLanguageMismatch(t *testing.T) {
	dir := t.TempDir()
	restore := chdir(t, dir)
	defer restore()

	err := InitProjectWithTemplate("rust", "basic", false, false)
	if err != nil {
		t.Fatal(err)
	}

	err = InitProjectWithTemplate("go", "basic", false, false)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestInitWithTemplateGoVersion(t *testing.T) {
	dir := t.TempDir()
	restore := chdir(t, dir)
	defer restore()

	err := InitProjectWithTemplate("go", "basic", false, false)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "module ") {
		t.Errorf("go.mod should contain 'module' directive")
	}
}

func TestInitWithTemplateForceOverwrites(t *testing.T) {
	dir := t.TempDir()
	restore := chdir(t, dir)
	defer restore()

	err := InitProjectWithTemplate("c", "basic", false, false)
	if err != nil {
		t.Fatal(err)
	}

	oldContent := "modified main"
	if err := os.WriteFile(filepath.Join(dir, "main.c"), []byte(oldContent), 0644); err != nil {
		t.Fatal(err)
	}

	err = InitProjectWithTemplate("c", "basic", true, false)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "main.c"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == oldContent {
		t.Error("main.c should have been overwritten in force mode")
	}
}

func TestInitWithTemplateVarExpansion(t *testing.T) {
	dir := t.TempDir()
	restore := chdir(t, dir)
	defer restore()

	err := InitProjectWithTemplate("go", "basic", false, false)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "version.go"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if strings.Contains(content, "$PROJECT") {
		t.Error("$PROJECT should have been expanded")
	}
	base := filepath.Base(dir)
	if !strings.Contains(content, base) {
		t.Errorf("version.go should contain directory name %q, got: %s", base, content)
	}
}

func TestInitWithTemplateKnownTemplatesCoverAllLanguages(t *testing.T) {
	allLangs := []string{"go", "tinygo", "c", "cxx", "rust", "python", "node", "java"}
	for _, lang := range allLangs {
		t.Run(lang, func(t *testing.T) {
			dir := t.TempDir()
			restore := chdir(t, dir)
			defer restore()

			err := InitProjectWithTemplate(lang, "basic", false, false)
			if err != nil {
				t.Fatalf("InitProjectWithTemplate(%q, basic): %v", lang, err)
			}

			if _, err := os.Stat("fiat"); err != nil {
				t.Errorf("missing fiat for %s/basic: %v", lang, err)
			}
		})
	}
}
