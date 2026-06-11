package lang

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grimdork/creo/internal/fiat"
)

func TestCommentStripping(t *testing.T) {
	content := []byte("# comment\nbuild: go\n\tcmd=echo hi\n# another\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
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
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
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
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	if len(f.Targets[0].Cmds) != 1 {
		t.Fatalf("expected 1 cmd, got %d", len(f.Targets[0].Cmds))
	}
}

func TestParenthesizedVars(t *testing.T) {
	vars := map[string]*fiat.Var{
		"bin": {Name: "bin", Value: "./creo"},
	}
	result := fiat.Expand("$(bin)-debug", vars, 0)
	expected := "./creo-debug"
	if result != expected {
		t.Fatalf("Expand($(bin)-debug): expected %q, got %q", expected, result)
	}

	result2 := fiat.Expand("$(bin)$(bin)", vars, 0)
	expected2 := "./creo./creo"
	if result2 != expected2 {
		t.Fatalf("Expand($(bin)$(bin)): expected %q, got %q", expected2, result2)
	}
}

func TestPlainVarStillWorks(t *testing.T) {
	vars := map[string]*fiat.Var{
		"bin": {Name: "bin", Value: "./creo"},
	}
	result := fiat.Expand("$bin", vars, 0)
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
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
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
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
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
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
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
	vars := map[string]*fiat.Var{
		"bin": {Name: "bin", Value: "./creo"},
	}
	trg := &fiat.Target{
		Name: "install",
		Vars: []*fiat.Var{{Name: "go", Value: "1"}},
	}
	result := fiat.ExpandWithTarget("$(bin)-debug", vars, trg)
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
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}

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
	if trg.Cmds[0] != "$GO $args -buildvcs=false -trimpath -ldflags=\"-s -w -buildid=reproducible -X main.version=$VERSION\" -o $bin ./cmd/server" {
		t.Fatalf("unexpected cmd: %s", trg.Cmds[0])
	}
}

func TestOciDefaults(t *testing.T) {
	content := []byte("image: oci\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}

	if len(f.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(f.Targets))
	}
	trg := f.Targets[0]
	if trg.Language != "oci" {
		t.Fatalf("expected oci language, got %q", trg.Language)
	}
	if trg.OCI == nil {
		t.Fatal("expected OCI config")
	}
	if trg.OCI.AppDir != "/app" {
		t.Fatalf("expected appdir /app, got %q", trg.OCI.AppDir)
	}
}

func TestGlobalVars(t *testing.T) {
	content := []byte("build: go\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}

	if _, ok := f.Vars["COMMIT"]; !ok {
		t.Fatal("expected COMMIT var to be set")
	}
	if _, ok := f.Vars["DATE"]; !ok {
		t.Fatal("expected DATE var to be set")
	}
}

func chdir(t *testing.T, dir string) func() {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	return func() { os.Chdir(old) }
}

func TestApplyUnknownLanguage(t *testing.T) {
	content := []byte("build: zig\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	err = Apply(f)
	if err == nil {
		t.Fatal("expected error for unknown language")
	}
	if !strings.Contains(err.Error(), "unknown language") {
		t.Fatalf("expected error containing 'unknown language', got %q", err.Error())
	}
}

func TestApplyC(t *testing.T) {
	content := []byte("build: c\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	if len(f.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(f.Targets))
	}
	trg := f.Targets[0]
	if trg.Sources != "*.c" {
		t.Fatalf("expected sources '*.c', got %q", trg.Sources)
	}
	if !strings.HasPrefix(trg.Bin, "build/") {
		t.Fatalf("expected Bin to start with 'build/', got %q", trg.Bin)
	}
	if len(trg.Cmds) == 0 {
		t.Fatal("expected at least one cmd")
	}
	if !strings.Contains(trg.Cmds[0], "$CC") {
		t.Fatalf("expected cmd to contain '$CC', got %q", trg.Cmds[0])
	}
}

func TestApplyCxx(t *testing.T) {
	content := []byte("build: cxx\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	if len(f.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(f.Targets))
	}
	trg := f.Targets[0]
	if trg.Sources != "*.cpp" {
		t.Fatalf("expected sources '*.cpp', got %q", trg.Sources)
	}
}

func TestApplyCpp(t *testing.T) {
	content := []byte("build: cpp\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	if len(f.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(f.Targets))
	}
	trg := f.Targets[0]
	if trg.Sources != "*.cpp" {
		t.Fatalf("expected sources '*.cpp', got %q", trg.Sources)
	}
}

func TestApplyOci(t *testing.T) {
	content := []byte("image: oci\n\trepo=ghcr.io/u/r\n\ttag=v1\n\tappdir=/srv\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	if len(f.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(f.Targets))
	}
	trg := f.Targets[0]
	if trg.OCI == nil {
		t.Fatal("expected OCI config")
	}
	if trg.OCI.Repo != "ghcr.io/u/r" {
		t.Fatalf("expected repo 'ghcr.io/u/r', got %q", trg.OCI.Repo)
	}
	if trg.OCI.Tag != "v1" {
		t.Fatalf("expected tag 'v1', got %q", trg.OCI.Tag)
	}
	if trg.OCI.AppDir != "/srv" {
		t.Fatalf("expected appdir '/srv', got %q", trg.OCI.AppDir)
	}
	if trg.OCI.Tarball != "" {
		t.Fatalf("expected empty tarball (repo is set), got %q", trg.OCI.Tarball)
	}
}

func TestApplyOciCacert(t *testing.T) {
	content := []byte("image: oci\n\tcacert=auto\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	if len(f.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(f.Targets))
	}
	trg := f.Targets[0]
	if trg.OCI == nil {
		t.Fatal("expected OCI config")
	}
	if trg.OCI.CACert != "auto" {
		t.Fatalf("expected CACert 'auto', got %q", trg.OCI.CACert)
	}
}

func TestApplyOciDefaultTarball(t *testing.T) {
	content := []byte("img: oci\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	if len(f.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(f.Targets))
	}
	trg := f.Targets[0]
	if trg.OCI == nil {
		t.Fatal("expected OCI config")
	}
	if trg.OCI.Tarball != "build/img.tar" {
		t.Fatalf("expected tarball 'build/img.tar', got %q", trg.OCI.Tarball)
	}
	if trg.Bin != "build/img.tar" {
		t.Fatalf("expected Bin to be 'build/img.tar', got %q", trg.Bin)
	}
}

func TestApplyOciFrom(t *testing.T) {
	content := []byte("img: oci\n\tfrom=alpine:latest\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	if len(f.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(f.Targets))
	}
	trg := f.Targets[0]
	if trg.OCI == nil {
		t.Fatal("expected OCI config")
	}
	if trg.OCI.BaseImage != "alpine:latest" {
		t.Fatalf("expected BaseImage 'alpine:latest', got %q", trg.OCI.BaseImage)
	}
}

func TestApplyOciSBOM(t *testing.T) {
	content := []byte("img: oci\n\tsbom=true\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	if len(f.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(f.Targets))
	}
	trg := f.Targets[0]
	if trg.OCI == nil {
		t.Fatal("expected OCI config")
	}
	if !trg.OCI.SBOM {
		t.Fatal("expected SBOM to be true")
	}
}

func TestModuleName(t *testing.T) {
	dir1 := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir1, "go.mod"), []byte("module github.com/foo/bar\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := ModuleName(dir1); got != "bar" {
		t.Fatalf("ModuleName for github.com/foo/bar: expected 'bar', got %q", got)
	}

	dir2 := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir2, "go.mod"), []byte("module example.com/my/app\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := ModuleName(dir2); got != "app" {
		t.Fatalf("ModuleName for example.com/my/app: expected 'app', got %q", got)
	}

	dir3 := t.TempDir()
	expected := filepath.Base(dir3)
	if got := ModuleName(dir3); got != expected {
		t.Fatalf("ModuleName without go.mod: expected %q, got %q", expected, got)
	}
}

func TestApplyGoCustomGOFLAGS(t *testing.T) {
	content := []byte("$GOFLAGS=-tags=netgo\nbuild: go\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	if len(f.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(f.Targets))
	}
	trg := f.Targets[0]
	if len(trg.Cmds) == 0 {
		t.Fatal("expected at least one cmd")
	}
	if !strings.Contains(trg.Cmds[0], "$GOFLAGS") {
		t.Fatalf("expected cmd to contain '$GOFLAGS', got %q", trg.Cmds[0])
	}
}

func TestInitC(t *testing.T) {
	dir := t.TempDir()
	ignores, err := InitC(dir, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "fiat")); os.IsNotExist(err) {
		t.Fatal("expected fiat file to exist")
	}
	if _, err := os.Stat(filepath.Join(dir, "main.c")); os.IsNotExist(err) {
		t.Fatal("expected main.c to exist")
	}
	if len(ignores) == 0 {
		t.Fatal("expected at least one ignore entry")
	}
}

func TestInitCxx(t *testing.T) {
	dir := t.TempDir()
	ignores, err := InitCxx(dir, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "fiat")); os.IsNotExist(err) {
		t.Fatal("expected fiat file to exist")
	}
	if _, err := os.Stat(filepath.Join(dir, "main.cpp")); os.IsNotExist(err) {
		t.Fatal("expected main.cpp to exist")
	}
	if len(ignores) == 0 {
		t.Fatal("expected at least one ignore entry")
	}
}

func TestInitOci(t *testing.T) {
	dir := t.TempDir()
	ignores, err := InitOci(dir, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "fiat")); os.IsNotExist(err) {
		t.Fatal("expected fiat file to exist")
	}
	f, err := fiat.Parse(filepath.Join(dir, "fiat"))
	if err != nil {
		t.Fatal(err)
	}
	build := fiat.FindTarget(f, "build")
	if build == nil {
		t.Fatal("expected 'build' target")
	}
	if build.Language != "go" {
		t.Fatalf("expected build target language 'go', got %q", build.Language)
	}
	image := fiat.FindTarget(f, "image")
	if image == nil {
		t.Fatal("expected 'image' target")
	}
	if image.Language != "oci" {
		t.Fatalf("expected image target language 'oci', got %q", image.Language)
	}
	if len(ignores) == 0 {
		t.Fatal("expected at least one ignore entry")
	}
}

func TestWriteIgnores(t *testing.T) {
	dir := t.TempDir()
	restore := chdir(t, dir)
	defer restore()

	WriteIgnores([]string{"/.creo"}, false)
	data, err := os.ReadFile(".gitignore")
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "/.creo") {
		t.Fatalf("expected .gitignore to contain '/.creo', got %q", content)
	}

	WriteIgnores([]string{"/.creo"}, false)
	data2, err := os.ReadFile(".gitignore")
	if err != nil {
		t.Fatal(err)
	}
	if string(data2) != string(data) {
		t.Fatal("expected .gitignore content to not change after duplicate call")
	}
}

func TestInitProjectNoLangs(t *testing.T) {
	dir := t.TempDir()
	restore := chdir(t, dir)
	defer restore()

	err := InitProject([]string{}, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat("fiat"); os.IsNotExist(err) {
		t.Fatal("expected fiat file to exist")
	}
}

func TestInitProjectGo(t *testing.T) {
	dir := t.TempDir()
	restore := chdir(t, dir)
	defer restore()

	err := InitProject([]string{"go"}, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat("fiat"); os.IsNotExist(err) {
		t.Fatal("expected fiat file to exist")
	}
	if _, err := os.Stat("main.go"); os.IsNotExist(err) {
		t.Fatal("expected main.go to exist")
	}
}

func TestInitProjectUnknown(t *testing.T) {
	dir := t.TempDir()
	restore := chdir(t, dir)
	defer restore()

	err := InitProject([]string{"zig"}, false, false)
	if err == nil {
		t.Fatal("expected error for unknown language")
	}
	if !strings.Contains(err.Error(), "unknown language") {
		t.Fatalf("expected error containing 'unknown language', got %q", err.Error())
	}
}

func TestApplyGoDebug(t *testing.T) {
	content := []byte("debug: go\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	if len(f.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(f.Targets))
	}
	trg := f.Targets[0]
	if !strings.HasSuffix(trg.Bin, "-debug") {
		t.Fatalf("expected Bin to end with '-debug', got %q", trg.Bin)
	}
	if len(trg.Cmds) == 0 {
		t.Fatal("expected at least one cmd")
	}
	if !strings.Contains(trg.Cmds[0], "$GODEBUGFLAGS") {
		t.Fatalf("expected cmd to contain '$GODEBUGFLAGS', got %q", trg.Cmds[0])
	}
}

func TestOciGhcrAlias(t *testing.T) {
	content := []byte("deploy: oci:ghcr OWNER=myorg\n\ttag=latest\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	if len(f.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(f.Targets))
	}
	trg := f.Targets[0]
	if trg.LangAlias != "ghcr" {
		t.Fatalf("expected LangAlias 'ghcr', got %q", trg.LangAlias)
	}
	if trg.OCI.Repo != "ghcr.io/myorg/deploy" {
		t.Fatalf("expected repo 'ghcr.io/myorg/deploy', got %q", trg.OCI.Repo)
	}
	if trg.OCI.Tag != "latest" {
		t.Fatalf("expected tag 'latest', got %q", trg.OCI.Tag)
	}
}

func TestOciGhcrAliasOverride(t *testing.T) {
	content := []byte("deploy: oci:ghcr\n\trepo=ghcr.io/custom/repo\n\ttag=v1\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	trg := f.Targets[0]
	if trg.OCI.Repo != "ghcr.io/custom/repo" {
		t.Fatalf("expected overridden repo, got %q", trg.OCI.Repo)
	}
}

func TestOciDockerAlias(t *testing.T) {
	content := []byte("deploy: oci:docker OWNER=jdoe\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	trg := f.Targets[0]
	if trg.OCI.Repo != "docker.io/jdoe/deploy" {
		t.Fatalf("expected docker.io repo, got %q", trg.OCI.Repo)
	}
}

func TestOciEcrAlias(t *testing.T) {
	content := []byte("deploy: oci:ecr OWNER=123456789012 region=us-west-2\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	trg := f.Targets[0]
	expectedRepo := "123456789012.dkr.ecr.us-west-2.amazonaws.com/deploy"
	if trg.OCI.Repo != expectedRepo {
		t.Fatalf("expected repo %q, got %q", expectedRepo, trg.OCI.Repo)
	}
	if trg.OCI.User != "AWS" {
		t.Fatalf("expected ECR user 'AWS', got %q", trg.OCI.User)
	}
	if trg.OCI.CredHelper != "aws ecr get-login-password --region us-west-2" {
		t.Fatalf("unexpected cred helper: %q", trg.OCI.CredHelper)
	}
}

func TestOciScwAlias(t *testing.T) {
	content := []byte("deploy: oci:scw OWNER=myorg\n\tregion=nl-ams\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	trg := f.Targets[0]
	expectedRepo := "rg.nl-ams.scw.cloud/myorg/deploy"
	if trg.OCI.Repo != expectedRepo {
		t.Fatalf("expected repo %q, got %q", expectedRepo, trg.OCI.Repo)
	}
}

func TestOciScwCountryAlias(t *testing.T) {
	content := []byte("deploy: oci:scw OWNER=myorg\n\tregion=it\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	trg := f.Targets[0]
	expectedRepo := "rg.it-mil.scw.cloud/myorg/deploy"
	if trg.OCI.Repo != expectedRepo {
		t.Fatalf("expected repo %q, got %q", expectedRepo, trg.OCI.Repo)
	}
}

func TestOciGcrAlias(t *testing.T) {
	content := []byte("deploy: oci:gcr OWNER=my-project\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	trg := f.Targets[0]
	if trg.OCI.Repo != "gcr.io/my-project/deploy" {
		t.Fatalf("expected gcr.io repo, got %q", trg.OCI.Repo)
	}
}

func TestOciAcrAlias(t *testing.T) {
	content := []byte("deploy: oci:acr OWNER=myreg\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	trg := f.Targets[0]
	if trg.OCI.Repo != "myreg.azurecr.io/deploy" {
		t.Fatalf("expected azurecr.io repo, got %q", trg.OCI.Repo)
	}
}

func TestOciUnknownAlias(t *testing.T) {
	content := []byte("deploy: oci:unknown OWNER=myorg\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	trg := f.Targets[0]
	if trg.OCI.Repo != "" {
		t.Fatalf("expected empty repo for unknown alias, got %q", trg.OCI.Repo)
	}
}

func TestOwnerFromFileVar(t *testing.T) {
	content := []byte("$OWNER=myorg\ndeploy: oci:ghcr\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	trg := f.Targets[0]
	if trg.OCI.Repo != "ghcr.io/myorg/deploy" {
		t.Fatalf("expected ghcr.io/myorg/deploy, got %q", trg.OCI.Repo)
	}
}

func TestTestAliasWithRegionAndTag(t *testing.T) {
	content := []byte("deploy: oci:scw OWNER=myorg\n\ttag=latest\n\tos=linux\n\tarch=amd64\n\tregion=nl-ams\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	trg := f.Targets[0]
	if trg.OCI.Repo != "rg.nl-ams.scw.cloud/myorg/deploy" {
		t.Fatalf("expected scw repo, got %q", trg.OCI.Repo)
	}
	if trg.OCI.Tag != "latest" {
		t.Fatalf("expected tag 'latest', got %q", trg.OCI.Tag)
	}
}

func TestApplyWithArgs(t *testing.T) {
	content := []byte("build: go args=-v\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	if len(f.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(f.Targets))
	}
	trg := f.Targets[0]
	if len(trg.Cmds) == 0 {
		t.Fatal("expected at least one cmd")
	}
	if !strings.Contains(trg.Cmds[0], "$args") {
		t.Fatalf("expected cmd to contain '$args', got %q", trg.Cmds[0])
	}
}

func TestApplyRust(t *testing.T) {
	content := []byte("build: rust\n")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(`[package]
name = "myapp"
version = "0.1.0"
`), 0644); err != nil {
		t.Fatal(err)
	}
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	if len(f.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(f.Targets))
	}
	trg := f.Targets[0]
	if trg.Bin != "build/release/myapp" {
		t.Fatalf("expected bin 'build/release/myapp', got %q", trg.Bin)
	}
	if trg.Sources != "*.rs Cargo.toml Cargo.lock" {
		t.Fatalf("expected sources '*.rs Cargo.toml Cargo.lock', got %q", trg.Sources)
	}
	if len(trg.Cmds) == 0 {
		t.Fatal("expected at least one cmd")
	}
	if !strings.Contains(trg.Cmds[0], "--release") {
		t.Fatalf("expected --release in cmd, got %q", trg.Cmds[0])
	}
}

func TestApplyRustDebug(t *testing.T) {
	content := []byte("debug: rust\n")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(`[package]
name = "myapp"
version = "0.1.0"
`), 0644); err != nil {
		t.Fatal(err)
	}
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	if len(f.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(f.Targets))
	}
	trg := f.Targets[0]
	if trg.Bin != "build/debug/myapp" {
		t.Fatalf("expected bin 'build/debug/myapp', got %q", trg.Bin)
	}
	if len(trg.Cmds) == 0 {
		t.Fatal("expected at least one cmd")
	}
	if strings.Contains(trg.Cmds[0], "--release") {
		t.Fatalf("expected no --release in debug cmd, got %q", trg.Cmds[0])
	}
}

func TestCrateName(t *testing.T) {
	dir1 := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir1, "Cargo.toml"), []byte(`[package]
name = "my-crate"
version = "0.1.0"
`), 0644); err != nil {
		t.Fatal(err)
	}
	if got := CrateName(dir1); got != "my-crate" {
		t.Fatalf("CrateName: expected 'my-crate', got %q", got)
	}

	dir2 := t.TempDir()
	if got := CrateName(dir2); got != filepath.Base(dir2) {
		t.Fatalf("CrateName without Cargo.toml: expected %q, got %q", filepath.Base(dir2), got)
	}
}

func TestRustTriple(t *testing.T) {
	tests := []struct {
		arch, os, want string
	}{
		{"amd64", "linux", "x86_64-unknown-linux-gnu"},
		{"x86_64", "linux", "x86_64-unknown-linux-gnu"},
		{"arm64", "linux", "aarch64-unknown-linux-gnu"},
		{"aarch64", "linux", "aarch64-unknown-linux-gnu"},
		{"amd64", "darwin", "x86_64-apple-darwin"},
		{"amd64", "macos", "x86_64-apple-darwin"},
		{"arm64", "darwin", "aarch64-apple-darwin"},
		{"amd64", "freebsd", "x86_64-unknown-freebsd"},
		{"amd64", "windows", "x86_64-pc-windows-msvc"},
		{"arm", "linux", "armv7-unknown-linux-gnueabihf"},
		{"riscv64", "linux", ""},
	}
	for _, tt := range tests {
		got := rustTriple(tt.arch, tt.os)
		if got != tt.want {
			t.Errorf("rustTriple(%q, %q): expected %q, got %q", tt.arch, tt.os, tt.want, got)
		}
	}
}

func TestApplyRustCustomBin(t *testing.T) {
	content := []byte("build: rust\n\tbin=./target/release/$PROJECT\n")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(`[package]
name = "myapp"
version = "0.1.0"
`), 0644); err != nil {
		t.Fatal(err)
	}
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	trg := f.Targets[0]
	if trg.Bin != "./target/release/myapp" {
		t.Fatalf("expected bin './target/release/myapp', got %q", trg.Bin)
	}
}

func TestRustNoCargoToml(t *testing.T) {
	content := []byte("build: rust\n")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	trg := f.Targets[0]
	base := filepath.Base(dir)
	if trg.Bin != "build/release/"+base {
		t.Fatalf("expected bin 'build/release/%s', got %q", base, trg.Bin)
	}
}

func TestCrossEnvRust(t *testing.T) {
	got := CrossEnv("rust", "amd64", "linux")
	if len(got) != 1 || got[0] != "CARGO_BUILD_TARGET=x86_64-unknown-linux-gnu" {
		t.Fatalf("CrossEnv(rust, amd64, linux): got %v", got)
	}
	got2 := CrossEnv("rust", "arm64", "darwin")
	if len(got2) != 1 || got2[0] != "CARGO_BUILD_TARGET=aarch64-apple-darwin" {
		t.Fatalf("CrossEnv(rust, arm64, darwin): got %v", got2)
	}
	got3 := CrossEnv("rust", "riscv64", "linux")
	if got3 != nil {
		t.Fatalf("CrossEnv(rust, riscv64, linux): expected nil, got %v", got3)
	}
}

func TestApplyRustWithCARGO(t *testing.T) {
	content := []byte("$CARGO=cargo +nightly\nbuild: rust\n")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(`[package]
name = "x"
version = "0.1.0"
`), 0644); err != nil {
		t.Fatal(err)
	}
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	trg := f.Targets[0]
	if len(trg.Cmds) == 0 {
		t.Fatal("expected a cmd")
	}
	if !strings.Contains(trg.Cmds[0], "$CARGO") {
		t.Fatalf("expected cmd to contain '$CARGO', got %q", trg.Cmds[0])
	}
}

func TestProjectVarGo(t *testing.T) {
	content := []byte("build: go\n\tbin=./$PROJECT\n")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/foo/myapp\n"), 0644); err != nil {
		t.Fatal(err)
	}
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	if f.Vars["PROJECT"].Value != "myapp" {
		t.Fatalf("expected PROJECT 'myapp', got %q", f.Vars["PROJECT"].Value)
	}
	trg := f.Targets[0]
	if trg.Bin != "./myapp" {
		t.Fatalf("expected bin './myapp', got %q", trg.Bin)
	}
}

func TestProjectVarC(t *testing.T) {
	dir := t.TempDir()
	content := []byte("build: c\n\tbin=./$PROJECT\n")
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	base := filepath.Base(dir)
	if f.Vars["PROJECT"].Value != base {
		t.Fatalf("expected PROJECT %q, got %q", base, f.Vars["PROJECT"].Value)
	}
	trg := f.Targets[0]
	if trg.Bin != "./"+base {
		t.Fatalf("expected bin './%s', got %q", base, trg.Bin)
	}
}

func TestProjectVarCxx(t *testing.T) {
	dir := t.TempDir()
	content := []byte("build: cxx\n\tbin=./$PROJECT\n")
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	base := filepath.Base(dir)
	if f.Vars["PROJECT"].Value != base {
		t.Fatalf("expected PROJECT %q, got %q", base, f.Vars["PROJECT"].Value)
	}
	trg := f.Targets[0]
	if trg.Bin != "./"+base {
		t.Fatalf("expected bin './%s', got %q", base, trg.Bin)
	}
}

func TestProjectVarRust(t *testing.T) {
	content := []byte("build: rust\n\tbin=./target/release/$PROJECT\n")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(`[package]
name = "mycrate"
version = "0.1.0"
`), 0644); err != nil {
		t.Fatal(err)
	}
	fpath := filepath.Join(dir, "fiat")
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := fiat.Parse(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(f); err != nil {
		t.Fatal(err)
	}
	if f.Vars["PROJECT"].Value != "mycrate" {
		t.Fatalf("expected PROJECT 'mycrate', got %q", f.Vars["PROJECT"].Value)
	}
	trg := f.Targets[0]
	if trg.Bin != "./target/release/mycrate" {
		t.Fatalf("expected bin './target/release/mycrate', got %q", trg.Bin)
	}
}
