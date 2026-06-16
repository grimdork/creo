package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grimdork/climate/arg"
	"github.com/grimdork/creo/internal/cli"
	"github.com/grimdork/creo/internal/targets"
)

func TestGenerateCompletion(t *testing.T) {
	opt := arg.New("creo", "test")
	opt.SetDefaultHelp(true)
	opt.SetFlag(arg.GroupDefault, "i", "init", "Initialise")
	opt.SetFlag(arg.GroupDefault, "l", "list", "List targets")

	result := cli.GenerateCompletion(opt)

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
	if !strings.Contains(result, "go c cxx cpp rust python node typescript java kotlin gradle oci") {
		t.Fatal("expected language list in completion")
	}
}

func TestListTargets(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	fiatContent := "build: go\n\tdesc=Build the binary\n"
	if err := os.WriteFile("fiat", []byte(fiatContent), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := cli.ListTargets("")
	if err != nil {
		t.Fatalf("ListTargets returned error: %v", err)
	}

	if !strings.Contains(out, "build") {
		t.Fatalf("expected output to contain 'build', got %q", out)
	}
	if !strings.Contains(out, "Build the binary") {
		t.Fatalf("expected output to contain description, got %q", out)
	}
}

func TestListTargetsNoFiat(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	_, err = cli.ListTargets("")
	if err == nil {
		t.Fatal("expected error for missing fiat file")
	}
}

func TestListTargetsNotFound(t *testing.T) {
	_, err := cli.ListTargets("/nonexistent/path/fiat")
	if err == nil {
		t.Fatal("expected error for non-existent path")
	}
}

func TestInitProjectNoLangs(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	if err := targets.InitProject([]string{}, false, false); err != nil {
		t.Fatalf("targets.InitProject returned error: %v", err)
	}

	if _, err := os.Stat("fiat"); err != nil {
		t.Fatal("expected fiat file to be created")
	}
}

func TestInitProjectGo(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	if err := targets.InitProject([]string{"go"}, false, false); err != nil {
		t.Fatalf("targets.InitProject returned error: %v", err)
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
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	err = targets.InitProject([]string{"zig"}, false, false)
	if err == nil {
		t.Fatal("expected error for unknown language")
	}
	if !strings.Contains(err.Error(), "unknown language") {
		t.Fatalf("expected 'unknown language' error, got %v", err)
	}
}

func TestInitProjectForceRemovesCreo(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	os.MkdirAll(".creo", 0755)
	if err := os.WriteFile(filepath.Join(".creo", "cache"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := targets.InitProject([]string{}, true, false); err != nil {
		t.Fatalf("targets.InitProject returned error: %v", err)
	}

	if _, err := os.Stat(".creo"); err == nil {
		t.Fatal("expected .creo to be removed after force init")
	}
}
