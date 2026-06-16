package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyGoBasic(t *testing.T) {
	dir := t.TempDir()
	tmpl, err := ResolveTemplate("go", "basic")
	if err != nil {
		t.Fatal(err)
	}
	extra := map[string]string{"PROJECT": "myapp", "VERSION": "1.0.0"}
	if err := ApplyTemplate(tmpl, dir, extra, false, false); err != nil {
		t.Fatal(err)
	}

	for _, f := range []string{"main.go", "version.go", "fiat"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("missing %s: %v", f, err)
		}
	}

	if _, err := os.Stat(filepath.Join(dir, "main.go.tmpl")); !os.IsNotExist(err) {
		t.Error(".tmpl file should not exist in output")
	}

	data, err := os.ReadFile(filepath.Join(dir, "version.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "$PROJECT") {
		t.Error("version.go still contains $PROJECT")
	}
	if !strings.Contains(string(data), "myapp") {
		t.Error("version.go should contain expanded project name")
	}
}

func TestApplyTemplateStripsTmplSuffix(t *testing.T) {
	dir := t.TempDir()
	tmpl, err := ResolveTemplate("c", "basic")
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyTemplate(tmpl, dir, nil, false, false); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "main.c")); err != nil {
		t.Errorf("main.c should exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "main.c.tmpl")); !os.IsNotExist(err) {
		t.Error("main.c.tmpl should not exist in output")
	}
}

func TestApplyTemplateWithExtraVars(t *testing.T) {
	dir := t.TempDir()
	tmpl, err := ResolveTemplate("go", "basic")
	if err != nil {
		t.Fatal(err)
	}
	extra := map[string]string{"PROJECT": "custom", "VERSION": "2.0.0"}
	if err := ApplyTemplate(tmpl, dir, extra, false, false); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "version.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "custom") {
		t.Errorf("version.go should contain 'custom', got: %s", string(data))
	}
}

func TestApplyTemplateForceOverwrites(t *testing.T) {
	dir := t.TempDir()
	tmpl, err := ResolveTemplate("c", "basic")
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyTemplate(tmpl, dir, nil, false, false); err != nil {
		t.Fatal(err)
	}

	oldContent := "modified"
	if err := os.WriteFile(filepath.Join(dir, "main.c"), []byte(oldContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := ApplyTemplate(tmpl, dir, nil, true, false); err != nil {
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

func TestApplyTemplateSkipsExisting(t *testing.T) {
	dir := t.TempDir()
	tmpl, err := ResolveTemplate("c", "basic")
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyTemplate(tmpl, dir, nil, false, false); err != nil {
		t.Fatal(err)
	}

	oldContent, err := os.ReadFile(filepath.Join(dir, "main.c"))
	if err != nil {
		t.Fatal(err)
	}

	if err := ApplyTemplate(tmpl, dir, nil, false, false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "main.c"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(oldContent) {
		t.Error("existing file should not be overwritten without force")
	}
}

func TestApplyTemplatePythonBasicCreatesSubdirs(t *testing.T) {
	dir := t.TempDir()
	tmpl, err := ResolveTemplate("python", "basic")
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyTemplate(tmpl, dir, nil, false, false); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "src", "app", "main.py")); err != nil {
		t.Errorf("src/app/main.py should exist: %v", err)
	}
}

func TestApplyTemplateAllTemplates(t *testing.T) {
	tmpls, err := ListTemplates("")
	if err != nil {
		t.Fatal(err)
	}
	for _, tmpl := range tmpls {
		t.Run(tmpl.Language+"/"+tmpl.Name, func(t *testing.T) {
			dir := t.TempDir()
			extra := map[string]string{"PROJECT": "testproj1", "VERSION": "1.0.0"}
			if err := ApplyTemplate(&tmpl, dir, extra, false, false); err != nil {
				t.Fatalf("applying %s/%s: %v", tmpl.Language, tmpl.Name, err)
			}
			for _, f := range tmpl.Files {
				dst := strings.TrimSuffix(f, ".tmpl")
				if _, err := os.Stat(filepath.Join(dir, dst)); err != nil {
					t.Errorf("file %s should exist: %v", dst, err)
				}
			}
		})
	}
}
