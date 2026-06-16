package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAllEmbeddedTemplatesLoad(t *testing.T) {
	langs := []string{"c", "cxx", "go", "tinygo", "node", "java", "python", "rust"}
	for _, lang := range langs {
		tmpls, err := ListTemplates(lang)
		if err != nil {
			t.Fatalf("listing %s templates: %v", lang, err)
		}
		if len(tmpls) == 0 {
			t.Fatalf("no templates found for %s", lang)
		}
		for _, tmpl := range tmpls {
			resolved, err := ResolveTemplate(tmpl.Language, tmpl.Name)
			if err != nil {
				t.Fatalf("resolving %s/%s: %v", tmpl.Language, tmpl.Name, err)
			}
			if resolved.Language != lang {
				t.Fatalf("%s/%s language mismatch: got %s", lang, tmpl.Name, resolved.Language)
			}
		}
	}
}

func TestGoBasicTemplate(t *testing.T) {
	tmpl, err := ResolveTemplate("go", "basic")
	if err != nil {
		t.Fatalf("resolving go/basic: %v", err)
	}
	if tmpl.Name != "basic" {
		t.Fatalf("expected name 'basic', got %q", tmpl.Name)
	}
	if tmpl.Language != "go" {
		t.Fatalf("expected language 'go', got %q", tmpl.Language)
	}
	if len(tmpl.Files) != 3 {
		t.Fatalf("expected 3 files, got %d: %v", len(tmpl.Files), tmpl.Files)
	}
}

func TestTinyGoBasicTemplate(t *testing.T) {
	tmpl, err := ResolveTemplate("tinygo", "basic")
	if err != nil {
		t.Fatalf("resolving tinygo/basic: %v", err)
	}
	if tmpl.Language != "tinygo" {
		t.Fatalf("expected language 'tinygo', got %q", tmpl.Language)
	}
}

func TestNodeBasicTemplate(t *testing.T) {
	tmpl, err := ResolveTemplate("node", "basic")
	if err != nil {
		t.Fatalf("resolving node/basic: %v", err)
	}
	if tmpl.Language != "node" {
		t.Fatalf("expected language 'node', got %q", tmpl.Language)
	}
}

func TestJavaBasicTemplate(t *testing.T) {
	tmpl, err := ResolveTemplate("java", "basic")
	if err != nil {
		t.Fatalf("resolving java/basic: %v", err)
	}
	if tmpl.Language != "java" {
		t.Fatalf("expected language 'java', got %q", tmpl.Language)
	}
}

func TestPythonBasicTemplate(t *testing.T) {
	tmpl, err := ResolveTemplate("python", "basic")
	if err != nil {
		t.Fatalf("resolving python/basic: %v", err)
	}
	if tmpl.Language != "python" {
		t.Fatalf("expected language 'python', got %q", tmpl.Language)
	}
}

func TestParseTemplateINIMissingName(t *testing.T) {
	ini := "[template]\nlanguage=go\nfiles=main.go.tmpl\n"
	_, err := parseTemplateINI(ini, "/tmp")
	if err == nil {
		t.Fatal("expected error for missing name")
	}
	if !strings.Contains(err.Error(), "missing 'name'") {
		t.Fatalf("expected 'missing name' error, got %q", err.Error())
	}
}

func TestParseTemplateINIMissingLanguage(t *testing.T) {
	ini := "[template]\nname=basic\nfiles=main.go.tmpl\n"
	_, err := parseTemplateINI(ini, "/tmp")
	if err == nil {
		t.Fatal("expected error for missing language")
	}
	if !strings.Contains(err.Error(), "missing 'language'") {
		t.Fatalf("expected 'missing language' error, got %q", err.Error())
	}
}

func TestParseTemplateINIWithVars(t *testing.T) {
	ini := "[template]\nname=basic\ndescription=test\nlanguage=go\nfiles=main.go.tmpl, fiat.tmpl\n\n[vars]\nPORT=8080\n"
	tmpl, err := parseTemplateINI(ini, "/tmp")
	if err != nil {
		t.Fatal(err)
	}
	if tmpl.Name != "basic" {
		t.Fatalf("expected name 'basic', got %q", tmpl.Name)
	}
	if tmpl.Language != "go" {
		t.Fatalf("expected language 'go', got %q", tmpl.Language)
	}
	if tmpl.Description != "test" {
		t.Fatalf("expected description 'test', got %q", tmpl.Description)
	}
	if len(tmpl.Files) != 2 || tmpl.Files[0] != "main.go.tmpl" || tmpl.Files[1] != "fiat.tmpl" {
		t.Fatalf("unexpected files: %v", tmpl.Files)
	}
	if tmpl.Vars["PORT"] != "8080" {
		t.Fatalf("expected PORT=8080, got %q", tmpl.Vars["PORT"])
	}
}

func TestParseTemplateINIComments(t *testing.T) {
	ini := "# comment\n[template]\n; also comment\nname=basic\nlanguage=go\nfiles=main.go.tmpl\n"
	tmpl, err := parseTemplateINI(ini, "/tmp")
	if err != nil {
		t.Fatal(err)
	}
	if tmpl.Name != "basic" {
		t.Fatalf("expected name 'basic', got %q", tmpl.Name)
	}
}

func TestLoadTemplateMissingFiles(t *testing.T) {
	dir := t.TempDir()
	ini := "[template]\nname=missing\nlanguage=go\nfiles=nonexistent.go.tmpl\n"
	if err := os.WriteFile(filepath.Join(dir, "template.ini"), []byte(ini), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := loadTemplate(dir)
	if err == nil {
		t.Fatal("expected error for missing template file")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected 'not found' error, got %q", err.Error())
	}
}

func TestSaveTemplateRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := SaveTemplate("go/basic", false, false); err != nil {
		t.Fatalf("SaveTemplate(go/basic): %v", err)
	}

	ud, err := userTemplateDir()
	if err != nil {
		t.Fatal(err)
	}
	destDir := filepath.Join(ud, "go", "basic")

	for _, f := range []string{"template.ini", "main.go.tmpl", "version.go.tmpl", "fiat.tmpl"} {
		if _, err := os.Stat(filepath.Join(destDir, f)); err != nil {
			t.Errorf("missing saved file %s: %v", f, err)
		}
	}

	tmpl, err := ResolveTemplate("go", "basic")
	if err != nil {
		t.Fatalf("resolving go/basic after save: %v", err)
	}
	if tmpl.Dir != destDir {
		t.Fatalf("expected user template dir %q, got %q", destDir, tmpl.Dir)
	}
}

func TestSaveTemplateNotFound(t *testing.T) {
	err := SaveTemplate("go/nonexistent", false, false)
	if err == nil {
		t.Fatal("expected error for nonexistent template")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected 'not found' error, got %q", err.Error())
	}
}

func TestSaveTemplateInvalidSpec(t *testing.T) {
	err := SaveTemplate("invalid", false, false)
	if err == nil {
		t.Fatal("expected error for invalid spec")
	}
	if !strings.Contains(err.Error(), "expected lang/name") {
		t.Fatalf("expected format error, got %q", err.Error())
	}
}

func TestListAllTemplates(t *testing.T) {
	tmpls, err := ListTemplates("")
	if err != nil {
		t.Fatalf("listing all templates: %v", err)
	}
	// We expect at least: c/basic, cxx/basic, go/basic, go/arg, go/toolcmd, go/web,
	// python/cli, python/basic, rust/basic, rust/arg, rust/toolcmd, tinygo/basic,
	// node/basic, java/basic = 14+ templates
	if len(tmpls) < 14 {
		t.Fatalf("expected at least 14 templates, got %d", len(tmpls))
	}
}
