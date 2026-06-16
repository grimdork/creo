package templates

import (
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
