package lang

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCommentStripping(t *testing.T) {
	content := []byte("# comment\nbuild: go\n\tcmd=echo hi\n# another\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := ParseFiat(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Targets) != 1 || f.Targets[0].Name != "build" {
		t.Fatalf("expected 1 target 'build', got %d targets", len(f.Targets))
	}
}

func TestInlineCommentStripping(t *testing.T) {
	content := []byte("build: go\n\tcmd=echo hi # inline comment\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := ParseFiat(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Targets) != 1 || len(f.Targets[0].Cmds) != 1 {
		t.Fatalf("expected 1 target with 1 cmd, got %d targets, %d cmds",
			len(f.Targets), len(f.Targets[0].Cmds))
	}
	if f.Targets[0].Cmds[0] != "echo hi" {
		t.Fatalf("expected cmd 'echo hi', got %q", f.Targets[0].Cmds[0])
	}
}

func TestCommentLineStripping(t *testing.T) {
	content := []byte("build: go\n\t# this is a comment-only line\n\tcmd=echo hi\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := ParseFiat(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Targets[0].Cmds) != 1 {
		t.Fatalf("expected 1 cmd, got %d", len(f.Targets[0].Cmds))
	}
}

func TestParenthesizedVars(t *testing.T) {
	vars := map[string]*Var{
		"bin": {Name: "bin", Value: "./creo"},
	}
	result := Expand("$(bin)-debug", vars, 0)
	expected := "./creo-debug"
	if result != expected {
		t.Fatalf("Expand($(bin)-debug): expected %q, got %q", expected, result)
	}

	result2 := Expand("$(bin)$(bin)", vars, 0)
	expected2 := "./creo./creo"
	if result2 != expected2 {
		t.Fatalf("Expand($(bin)$(bin)): expected %q, got %q", expected2, result2)
	}
}

func TestPlainVarStillWorks(t *testing.T) {
	vars := map[string]*Var{
		"bin": {Name: "bin", Value: "./creo"},
	}
	result := Expand("$bin", vars, 0)
	expected := "./creo"
	if result != expected {
		t.Fatalf("Expand($bin): expected %q, got %q", expected, result)
	}
}

func TestInstallProperty(t *testing.T) {
	content := []byte("install: go\n\tinstall=$HOME/bin\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := ParseFiat(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Targets) != 1 {
		t.Fatal("expected 1 target")
	}
	if len(f.Targets[0].Install) != 1 {
		t.Fatalf("expected 1 install entry, got %d",
			len(f.Targets[0].Install))
	}
	if f.Targets[0].Install[0] != "$HOME/bin" {
		t.Fatalf("expected '$HOME/bin', got %q", f.Targets[0].Install[0])
	}
}

func TestInstallPropertyWithSource(t *testing.T) {
	content := []byte("install: go\n\tinstall=$bin:$HOME/bin\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := ParseFiat(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if f.Targets[0].Install[0] != "$bin:$HOME/bin" {
		t.Fatalf("expected '$bin:$HOME/bin', got %q",
			f.Targets[0].Install[0])
	}
}

func TestMultipleInstallLines(t *testing.T) {
	content := []byte("install: go\n\tinstall=$bin:$HOME/bin/\n\tinstall=$(bin)-debug:$HOME/bin/\n\trequire=build debug\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := ParseFiat(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Targets[0].Install) != 2 {
		t.Fatalf("expected 2 install entries, got %d",
			len(f.Targets[0].Install))
	}
	if f.Targets[0].Install[0] != "$bin:$HOME/bin/" {
		t.Fatalf("expected '$bin:$HOME/bin/', got %q",
			f.Targets[0].Install[0])
	}
	if f.Targets[0].Install[1] != "$(bin)-debug:$HOME/bin/" {
		t.Fatalf("expected '$(bin)-debug:$HOME/bin/', got %q",
			f.Targets[0].Install[1])
	}
}

func TestExpandWithParenthesizedInTarget(t *testing.T) {
	vars := map[string]*Var{
		"bin": {Name: "bin", Value: "./creo"},
	}
	trg := &Target{
		Name: "install",
		Vars: []*Var{{Name: "go", Value: "1"}},
	}
	result := ExpandWithTarget("$(bin)-debug", vars, trg)
	expected := "./creo-debug"
	if result != expected {
		t.Fatalf("ExpandWithTarget($(bin)-debug): expected %q, got %q",
			expected, result)
	}
}

func TestGoSRCDIR(t *testing.T) {
	content := []byte("build: go SRCDIR=./cmd/server\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := ParseFiat(fpath)
	if err != nil {
		t.Fatal(err)
	}
	Apply(f)

	if len(f.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(f.Targets))
	}
	trg := f.Targets[0]
	if trg.Sources != "./cmd/server/*.go" {
		t.Fatalf("expected sources './cmd/server/*.go', got %q", trg.Sources)
	}
	if len(trg.Cmds) == 0 {
		t.Fatal("expected a cmd")
	}
	if trg.Cmds[0] != "$GO -trimpath -ldflags=\"-s -w -X main.version=$VERSION\" -o $bin ./cmd/server" {
		t.Fatalf("unexpected cmd: %s", trg.Cmds[0])
	}
}

func TestKoDefaults(t *testing.T) {
	content := []byte("image: ko\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := ParseFiat(fpath)
	if err != nil {
		t.Fatal(err)
	}
	Apply(f)

	if len(f.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(f.Targets))
	}
	trg := f.Targets[0]
	if trg.Language != "ko" {
		t.Fatalf("expected ko language, got %q", trg.Language)
	}
	if len(trg.Cmds) == 0 {
		t.Fatal("expected a cmd")
	}
}

func TestGlobalVars(t *testing.T) {
	content := []byte("build: go\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := ParseFiat(fpath)
	if err != nil {
		t.Fatal(err)
	}
	Apply(f)

	if _, ok := f.Vars["COMMIT"]; !ok {
		t.Fatal("expected COMMIT var to be set")
	}
	if _, ok := f.Vars["DATE"]; !ok {
		t.Fatal("expected DATE var to be set")
	}
}
