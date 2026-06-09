package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grimdork/climate/arg"
)

func TestGenerateCompletion(t *testing.T) {
	opt := arg.New("creo", "test")
	opt.SetDefaultHelp(true)
	opt.SetFlag(arg.GroupDefault, "i", "init", "Initialise")
	opt.SetFlag(arg.GroupDefault, "l", "list", "List targets")

	result := generateCompletion(opt)

	if !strings.Contains(result, "_creo()") {
		t.Fatal("expected _creo() function in completion output")
	}
	if !strings.Contains(result, "__creo_targets()") {
		t.Fatal("expected __creo_targets() helper")
	}
	if !strings.Contains(result, "__creo_langs()") {
		t.Fatal("expected __creo_langs() helper")
	}
	if !strings.Contains(result, "complete -F _creo") {
		t.Fatal("expected complete -F _creo line")
	}
	if !strings.Contains(result, "go c cxx cpp oci") {
		t.Fatal("expected language list in completion")
	}
}

func TestListTargets(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)

	fiatContent := "build: go\n\tdesc=Build the binary\n"
	if err := os.WriteFile("fiat", []byte(fiatContent), 0644); err != nil {
		t.Fatal(err)
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := listTargets("")

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("listTargets returned error: %v", err)
	}

	out, _ := io.ReadAll(r)
	output := string(out)
	if !strings.Contains(output, "build") {
		t.Fatalf("expected output to contain 'build', got %q", output)
	}
	if !strings.Contains(output, "Build the binary") {
		t.Fatalf("expected output to contain description, got %q", output)
	}
}

func TestListTargetsNoFiat(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)

	err := listTargets("")
	if err == nil {
		t.Fatal("expected error for missing fiat file")
	}
}

func TestListTargetsNotFound(t *testing.T) {
	err := listTargets("/nonexistent/path/fiat")
	if err == nil {
		t.Fatal("expected error for non-existent path")
	}
}

func TestInitProjectNoLangs(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)

	if err := initProject([]string{}, false, false); err != nil {
		t.Fatalf("initProject returned error: %v", err)
	}

	if _, err := os.Stat("fiat"); err != nil {
		t.Fatal("expected fiat file to be created")
	}
}

func TestInitProjectGo(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)

	if err := initProject([]string{"go"}, false, false); err != nil {
		t.Fatalf("initProject returned error: %v", err)
	}

	if _, err := os.Stat("fiat"); err != nil {
		t.Fatal("expected fiat file to be created")
	}
	if _, err := os.Stat("main.go"); err != nil {
		t.Fatal("expected main.go to be created")
	}
}

func TestInitProjectUnknown(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)

	err := initProject([]string{"rust"}, false, false)
	if err == nil {
		t.Fatal("expected error for unknown language")
	}
	if !strings.Contains(err.Error(), "unknown language") {
		t.Fatalf("expected 'unknown language' error, got %v", err)
	}
}

func TestInitProjectForceRemovesCreo(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)

	os.MkdirAll(".creo", 0755)
	if err := os.WriteFile(filepath.Join(".creo", "cache"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := initProject([]string{}, true, false); err != nil {
		t.Fatalf("initProject returned error: %v", err)
	}

	if _, err := os.Stat(".creo"); err == nil {
		t.Fatal("expected .creo to be removed after force init")
	}
}
