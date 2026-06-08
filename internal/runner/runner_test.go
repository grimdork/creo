package runner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/grimdork/creo/internal/lang"
)

func writeFiat(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "fiat"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func parseAndApply(t *testing.T, dir string) *lang.FiatFile {
	t.Helper()
	f, err := lang.ParseFiat(filepath.Join(dir, "fiat"))
	if err != nil {
		t.Fatal(err)
	}
	lang.Apply(f)
	return f
}

func TestTargetNotFound(t *testing.T) {
	dir := t.TempDir()
	writeFiat(t, dir, "build: go\n")
	f := parseAndApply(t, dir)
	err := RunTarget(f, "nonexistent", RunOpts{})
	if err == nil {
		t.Fatal("expected error for missing target")
	}
}

func runInDir(t *testing.T, dir string) func() {
	t.Helper()
	cwd, _ := os.Getwd()
	os.Chdir(dir)
	return func() { os.Chdir(cwd) }
}

func TestSimpleGoBuild(t *testing.T) {
	dir := t.TempDir()
	defer runInDir(t, dir)()
	writeFiat(t, dir, "build: go\n")
	writeFile(t, dir, "go.mod", "module test\n")
	writeFile(t, dir, "main.go", "package main; func main() {}\n")
	f := parseAndApply(t, dir)

	if err := RunTarget(f, "build", RunOpts{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat("./test"); os.IsNotExist(err) {
		t.Fatal("expected binary to be created")
	}
}

func TestVirtualTargetAlwaysRuns(t *testing.T) {
	dir := t.TempDir()
	writeFiat(t, dir, ".test:\n\tcmd=echo ran\n")
	f := parseAndApply(t, dir)

	if err := RunTarget(f, ".test", RunOpts{}); err != nil {
		t.Fatal(err)
	}

	if err := RunTarget(f, ".test", RunOpts{}); err != nil {
		t.Fatal("virtual target should not rely on freshness")
	}
}

func TestKeepGoing(t *testing.T) {
	dir := t.TempDir()
	writeFiat(t, dir, "build: go\n\tcmd=exit 1\n")
	writeFile(t, dir, "go.mod", "module test\n")
	writeFile(t, dir, "main.go", "package main; func main() {}\n")
	f := parseAndApply(t, dir)

	opts := RunOpts{KeepGoing: true}
	err := RunTarget(f, "build", opts)
	if err == nil {
		t.Fatal("expected error even with keep-going")
	}
}

func TestDryRunDoesNotCreateFiles(t *testing.T) {
	dir := t.TempDir()
	writeFiat(t, dir, "build: go\n")
	writeFile(t, dir, "go.mod", "module test\n")
	writeFile(t, dir, "main.go", "package main; func main() {}\n")
	f := parseAndApply(t, dir)

	opts := RunOpts{DryRun: true}
	if err := RunTarget(f, "build", opts); err != nil {
		t.Fatal(err)
	}

	matches, err := filepath.Glob(filepath.Join(dir, "test"))
	if err == nil && len(matches) > 0 {
		t.Fatal("dry run should not create binary")
	}
}

func TestInstall(t *testing.T) {
	dir := t.TempDir()
	defer runInDir(t, dir)()
	dest := filepath.Join(dir, "dest", "test")
	writeFiat(t, dir, "build: go\n\tinstall=$bin:"+dest+"\n")
	writeFile(t, dir, "go.mod", "module test\n")
	writeFile(t, dir, "main.go", "package main; func main() {}\n")
	f := parseAndApply(t, dir)

	if err := RunTarget(f, "build", RunOpts{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(dest); os.IsNotExist(err) {
		t.Fatal("install should have copied binary to", dest)
	}
}

func TestCleanOnlyRemovesTargetsOwnFiles(t *testing.T) {
	dir := t.TempDir()
	defer runInDir(t, dir)()
	writeFiat(t, dir, "build: go\n")
	writeFile(t, dir, "go.mod", "module test\n")
	writeFile(t, dir, "main.go", "package main; func main() {}\n")
	f := parseAndApply(t, dir)

	if err := RunTarget(f, "build", RunOpts{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat("./test"); os.IsNotExist(err) {
		t.Fatal("build should create binary")
	}

	if err := RunTarget(f, "build", RunOpts{Clean: true}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat("./test"); err == nil {
		t.Fatal("clean should remove binary")
	}
}

func TestCrossEnv(t *testing.T) {
	tests := []struct {
		lang, arch, osval string
		want              []string
	}{
		{"go", "arm64", "", []string{"GOARCH=arm64"}},
		{"go", "", "linux", []string{"GOOS=linux"}},
		{"go", "arm64", "linux", []string{"GOARCH=arm64", "GOOS=linux"}},
		{"c", "arm64", "linux", nil},
		{"cxx", "arm64", "linux", nil},
		{"cpp", "arm64", "linux", nil},
		{"", "arm64", "linux", nil},
	}

	for _, tt := range tests {
		got := lang.CrossEnv(tt.lang, tt.arch, tt.osval)
		if len(got) != len(tt.want) {
			t.Errorf("CrossEnv(%q, %q, %q) = %v, want %v", tt.lang, tt.arch, tt.osval, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("CrossEnv(%q, %q, %q) = %v, want %v", tt.lang, tt.arch, tt.osval, got, tt.want)
			}
		}
	}
}
