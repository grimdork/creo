package runner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/grimdork/creo/internal/fiat"
	"github.com/grimdork/creo/internal/lang"
	"github.com/grimdork/creo/internal/util"
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

func parseAndApply(t *testing.T, dir string) *fiat.File {
	t.Helper()
	f, err := fiat.Parse(filepath.Join(dir, "fiat"))
	if err != nil {
		t.Fatal(err)
	}
	if err := lang.Apply(f); err != nil {
		t.Fatal(err)
	}
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

	if _, err := os.Stat("build/test"); os.IsNotExist(err) {
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

	matches, err := filepath.Glob(filepath.Join(dir, "build", "test"))
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

	if _, err := os.Stat("build/test"); os.IsNotExist(err) {
		t.Fatal("build should create binary")
	}

	if err := RunTarget(f, "build", RunOpts{Clean: true}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat("build/test"); err == nil {
		t.Fatal("clean should remove binary")
	}
}

func TestSRCDIRBuild(t *testing.T) {
	dir := t.TempDir()
	defer runInDir(t, dir)()
	os.MkdirAll(dir+"/cmd/server", 0755)
	writeFiat(t, dir, "build: go SRCDIR=./cmd/server\n")
	writeFile(t, dir, "go.mod", "module test\n")
	writeFile(t, dir, "cmd/server/main.go", "package main; func main() {}\n")
	f := parseAndApply(t, dir)

	if err := RunTarget(f, "build", RunOpts{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat("build/test"); os.IsNotExist(err) {
		t.Fatal("expected binary to be created from sub-package")
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

func TestOutputsStoreLoad(t *testing.T) {
	o := Outputs{m: make(map[string]string)}
	o.Store("build", "amd64", "linux", "/tmp/mybin")
	if v := o.Load("build", "amd64", "linux"); v != "/tmp/mybin" {
		t.Fatalf("expected /tmp/mybin, got %q", v)
	}
	if v := o.Load("build", "arm64", "linux"); v != "" {
		t.Fatalf("expected empty, got %q", v)
	}
	if v := o.Load("other", "amd64", "linux"); v != "" {
		t.Fatalf("expected empty, got %q", v)
	}
}

func TestOutputsLoadAll(t *testing.T) {
	o := Outputs{m: make(map[string]string)}
	o.Store("build", "amd64", "linux", "b1")
	o.Store("build", "arm64", "linux", "b2")
	o.Store("other", "amd64", "linux", "o1")
	all := o.LoadAll("build")
	if len(all) != 2 {
		t.Fatalf("expected 2 keys, got %d: %v", len(all), all)
	}
	expected := map[string]bool{"amd64+linux": true, "arm64+linux": true}
	for _, v := range all {
		if !expected[v] {
			t.Fatalf("unexpected key %q", v)
		}
	}
	if got := o.LoadAll("nonexistent"); len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestHasCombo(t *testing.T) {
	if !hasCombo([]string{"amd64", "arm64"}, []string{"linux"}, "amd64", "linux") {
		t.Fatal("expected true for amd64+linux")
	}
	if !hasCombo([]string{"amd64", "arm64"}, []string{"linux"}, "arm64", "linux") {
		t.Fatal("expected true for arm64+linux")
	}
	if hasCombo([]string{"amd64", "arm64"}, []string{"linux"}, "amd64", "darwin") {
		t.Fatal("expected false for amd64+darwin")
	}
	if hasCombo([]string{}, []string{}, "amd64", "linux") {
		t.Fatal("expected false for empty archs/oses")
	}
}

func TestGlobFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "a.go", "")
	writeFile(t, dir, "b.go", "")
	writeFile(t, dir, "sub/c.go", "")
	writeFile(t, dir, "sub/d.h", "")

	got := util.GlobFiles("*.go", dir)
	if len(got) != 2 {
		t.Fatalf("expected 2 .go files, got %d: %v", len(got), got)
	}
	names := make(map[string]bool)
	for _, f := range got {
		names[filepath.Base(f)] = true
	}
	if !names["a.go"] || !names["b.go"] {
		t.Fatal("expected a.go and b.go")
	}

	got = util.GlobFiles("**/*.go", dir)
	if len(got) != 3 {
		t.Fatalf("expected 3 .go files with **, got %d: %v", len(got), got)
	}

	got = util.GlobFiles("nonexistent", dir)
	if len(got) != 0 {
		t.Fatalf("expected empty, got %d: %v", len(got), got)
	}
}

func TestAllTarget(t *testing.T) {
	dir := t.TempDir()
	defer runInDir(t, dir)()
	writeFiat(t, dir, "build: go\n\ntest:\n\tcmd=touch marker\n")
	writeFile(t, dir, "go.mod", "module test\n")
	writeFile(t, dir, "main.go", "package main; func main() {}\n")
	f := parseAndApply(t, dir)

	if err := RunTarget(f, "all", RunOpts{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat("build/test"); os.IsNotExist(err) {
		t.Fatal("expected binary from build target")
	}

	if _, err := os.Stat("./marker"); os.IsNotExist(err) {
		t.Fatal("expected marker file from test target")
	}
}

func TestCircularDependency(t *testing.T) {
	dir := t.TempDir()
	writeFiat(t, dir, ".a:\n\trequire=.b\n\n.b:\n\trequire=.a\n")
	f := parseAndApply(t, dir)

	err := RunTarget(f, ".a", RunOpts{})
	if err == nil {
		t.Fatal("expected error for circular dependency")
	}
}

func TestDependencyNotFound(t *testing.T) {
	dir := t.TempDir()
	writeFiat(t, dir, ".a:\n\trequire=nonexistent\n")
	f := parseAndApply(t, dir)

	err := RunTarget(f, ".a", RunOpts{})
	if err == nil {
		t.Fatal("expected error for missing dependency")
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	content := "hello world"
	if err := os.WriteFile(src, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(dir, "dest.txt")
	if err := util.CopyFile(src, dest); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != content {
		t.Fatalf("expected %q, got %q", content, string(data))
	}

	src2 := filepath.Join(dir, "src2.txt")
	if err := os.WriteFile(src2, []byte("perm"), 0755); err != nil {
		t.Fatal(err)
	}
	dest2 := filepath.Join(dir, "dest2.txt")
	if err := util.CopyFile(src2, dest2); err != nil {
		t.Fatal(err)
	}
	si, err := os.Stat(dest2)
	if err != nil {
		t.Fatal(err)
	}
	if si.Mode().Perm() != 0755 {
		t.Fatalf("expected 0755 permissions, got %#o", si.Mode().Perm())
	}

	if err := util.CopyFile(src, src); err != nil {
		t.Fatal("CopyFile(src, src) should be a no-op")
	}

	if err := util.CopyFile(filepath.Join(dir, "nonexistent"), dest); err == nil {
		t.Fatal("expected error for non-existent source")
	}
}

func TestFindFiatInDir(t *testing.T) {
	dir1 := t.TempDir()
	writeFiat(t, dir1, "build: go\n")
	path, ok := findFiatInDir(dir1, false)
	if !ok {
		t.Fatal("expected to find fiat file")
	}
	if path != filepath.Join(dir1, "fiat") {
		t.Fatalf("expected %q, got %q", filepath.Join(dir1, "fiat"), path)
	}

	dir2 := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir2, "project.fiat"), []byte("build: go\n"), 0644); err != nil {
		t.Fatal(err)
	}
	path, ok = findFiatInDir(dir2, false)
	if !ok {
		t.Fatal("expected to find .fiat file")
	}
	if path != filepath.Join(dir2, "project.fiat") {
		t.Fatalf("expected %q, got %q", filepath.Join(dir2, "project.fiat"), path)
	}

	dir3 := t.TempDir()
	path, ok = findFiatInDir(dir3, false)
	if ok {
		t.Fatal("expected not to find fiat file")
	}
	if path != "" {
		t.Fatalf("expected empty path, got %q", path)
	}

	dir4 := t.TempDir()
	writeFiat(t, dir4, "build: go\n")
	if err := os.WriteFile(filepath.Join(dir4, "other.fiat"), []byte("build: go\n"), 0644); err != nil {
		t.Fatal(err)
	}
	path, ok = findFiatInDir(dir4, false)
	if !ok {
		t.Fatal("expected to find fiat file when both exist")
	}
	if path != filepath.Join(dir4, "fiat") {
		t.Fatalf("expected %q (fiat preferred), got %q", filepath.Join(dir4, "fiat"), path)
	}
}

func TestExecShebangShell(t *testing.T) {
	err := execShebang("#!/bin/sh\necho hello", t.TempDir(), nil)
	if err != nil {
		t.Fatalf("execShebang: %v", err)
	}
}

func TestExecShebangFails(t *testing.T) {
	err := execShebang("#!/nonexistent/interpreter\nexit 0", t.TempDir(), nil)
	if err == nil {
		t.Fatal("execShebang with bad interpreter expected error, got nil")
	}
}

func TestExecCmdShebang(t *testing.T) {
	err := execCmd("#!/bin/sh\necho shebang-ok", t.TempDir(), nil)
	if err != nil {
		t.Fatalf("execCmd with shebang: %v", err)
	}
}

func TestExecCmdNormal(t *testing.T) {
	err := execCmd("echo normal-ok", t.TempDir(), nil)
	if err != nil {
		t.Fatalf("execCmd normal: %v", err)
	}
}

func TestComputeCacheKey(t *testing.T) {
	dir := t.TempDir()
	runInDir(t, dir)

	src := filepath.Join(dir, "main.go")
	if err := os.WriteFile(src, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	key1, err := computeCacheKey([]string{src}, []string{"go build"})
	if err != nil {
		t.Fatal(err)
	}
	key2, err := computeCacheKey([]string{src}, []string{"go build"})
	if err != nil {
		t.Fatal(err)
	}
	if key1 != key2 {
		t.Fatal("cache key not deterministic")
	}

	key3, err := computeCacheKey([]string{src}, []string{"go build -v"})
	if err != nil {
		t.Fatal(err)
	}
	if key1 == key3 {
		t.Fatal("cache key should change when command changes")
	}
}

func TestCheckCacheMiss(t *testing.T) {
	dir := t.TempDir()
	if checkCache(dir, "nonexistent", nil, nil) {
		t.Fatal("expected false for missing cache")
	}
}

func TestCheckCacheHitAndMiss(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "main.go")
	if err := os.WriteFile(src, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}
	sources := []string{src}
	cmds := []string{"go build"}

	if err := writeCache(dir, "test-target", sources, cmds); err != nil {
		t.Fatalf("writeCache: %v", err)
	}

	if !checkCache(dir, "test-target", sources, cmds) {
		t.Fatal("expected cache hit after write")
	}

	if cc := checkCache(dir, "test-target", sources, []string{"different cmd"}); cc {
		t.Fatal("expected cache miss after cmd change")
	}

	if err := os.WriteFile(src, []byte("package main\n// changed"), 0644); err != nil {
		t.Fatal(err)
	}
	if checkCache(dir, "test-target", sources, cmds) {
		t.Fatal("expected cache miss after source change")
	}
}

func TestCollectFilePaths(t *testing.T) {
	dir := t.TempDir()
	runInDir(t, dir)
	writeFiat(t, dir, "build: go\n\tsources=*.go\n")
	writeFile(t, dir, "main.go", "package main\n")
	writeFile(t, dir, "helper.go", "package main\n")
	f := parseAndApply(t, dir)
	target := fiat.FindTarget(f, "build")
	if target == nil {
		t.Fatal("target not found")
	}
	paths := collectFilePaths(target, f, dir)
	if len(paths) < 2 {
		t.Fatalf("expected at least 2 paths (main.go, helper.go), got %d", len(paths))
	}
}

func TestCachedBuildSkips(t *testing.T) {
	dir := t.TempDir()
	runInDir(t, dir)
	writeFiat(t, dir, "build: go\n\tsources=*.go\n")
	writeFile(t, dir, "go.mod", "module test\n")
	writeFile(t, dir, "main.go", "package main; func main() {}\n")
	f := parseAndApply(t, dir)

	// First build: should run
	if err := RunTarget(f, "build", RunOpts{}); err != nil {
		t.Fatal(err)
	}

	// Cache should now exist
	if _, err := os.Stat(".creo/cache/build.json"); os.IsNotExist(err) {
		t.Fatal("expected cache file after build")
	}

	// Second build: should skip (cached)
	if err := RunTarget(f, "build", RunOpts{}); err != nil {
		t.Fatal(err)
	}
	// We can't easily verify the skip message, but we can verify no error
}

func TestCachedBuildRebuildsAfterChange(t *testing.T) {
	dir := t.TempDir()
	runInDir(t, dir)
	writeFiat(t, dir, "build: go\n\tsources=*.go\n")
	writeFile(t, dir, "go.mod", "module test\n")
	writeFile(t, dir, "main.go", "package main; func main() {}\n")
	f := parseAndApply(t, dir)

	if err := RunTarget(f, "build", RunOpts{}); err != nil {
		t.Fatal(err)
	}

	// Modify source
	writeFile(t, dir, "main.go", "package main; func main() { println(\"v2\") }\n")

	// Rebuild: should run again (source changed)
	if err := RunTarget(f, "build", RunOpts{}); err != nil {
		t.Fatal(err)
	}
}

func TestMultiArchBuild(t *testing.T) {
	dir := t.TempDir()
	defer runInDir(t, dir)()
	writeFiat(t, dir, "build: go\n\tos=linux\n\tarch=amd64 arm64\n\tbin=$bin-$os-$arch\n")
	writeFile(t, dir, "go.mod", "module test\n")
	writeFile(t, dir, "main.go", "package main; func main() {}\n")
	f := parseAndApply(t, dir)

	if err := RunTarget(f, "build", RunOpts{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat("build/test-linux-amd64"); os.IsNotExist(err) {
		t.Fatal("expected test-linux-amd64 binary")
	}
	if _, err := os.Stat("build/test-linux-arm64"); os.IsNotExist(err) {
		t.Fatal("expected test-linux-arm64 binary")
	}
}

func TestExecCredHelperColonFormat(t *testing.T) {
	dir := t.TempDir()
	helper := filepath.Join(dir, "helper.sh")
	writeFile(t, dir, "helper.sh", `#!/bin/sh
echo "user:password"
`)
	if err := os.Chmod(helper, 0755); err != nil {
		t.Fatal(err)
	}
	user, pass, err := execCredHelper(helper, dir)
	if err != nil {
		t.Fatal(err)
	}
	if user != "user" {
		t.Fatalf("expected user 'user', got %q", user)
	}
	if pass != "password" {
		t.Fatalf("expected pass 'password', got %q", pass)
	}
}

func TestExecCredHelperNoColon(t *testing.T) {
	dir := t.TempDir()
	helper := filepath.Join(dir, "helper.sh")
	writeFile(t, dir, "helper.sh", `#!/bin/sh
echo "token123"
`)
	if err := os.Chmod(helper, 0755); err != nil {
		t.Fatal(err)
	}
	user, pass, err := execCredHelper(helper, dir)
	if err != nil {
		t.Fatal(err)
	}
	if user != "" {
		t.Fatalf("expected empty user for colon-less output, got %q", user)
	}
	if pass != "token123" {
		t.Fatalf("expected pass 'token123', got %q", pass)
	}
}

func TestExecCredHelperError(t *testing.T) {
	user, pass, err := execCredHelper("nonexistent-command-nonexistent", "/tmp")
	if err == nil {
		t.Fatal("expected error for missing helper")
	}
	if user != "" || pass != "" {
		t.Fatal("expected empty user/pass on error")
	}
}
