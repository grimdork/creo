package fiat

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/grimdork/creo/internal/util"
)

func TestParseEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fiat")
	if err := os.WriteFile(path, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(f.Targets))
	}
	if _, ok := f.Vars["DIR"]; !ok {
		t.Error("expected DIR var")
	}
}

func TestParseCommentsOnly(t *testing.T) {
	content := "# This is a comment\n# Another comment\n\n# After blank\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "fiat")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	f, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(f.Targets))
	}
	if _, ok := f.Vars["DIR"]; !ok {
		t.Error("expected DIR var")
	}
}

func TestParseGlobalVars(t *testing.T) {
	content := "$name=test\n$eager:=immediate\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "fiat")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	f, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	v, ok := f.Vars["name"]
	if !ok {
		t.Fatal("expected var name")
	}
	if v.Value != "test" {
		t.Errorf("name value = %q, want %q", v.Value, "test")
	}
	if v.Eager {
		t.Error("name should not be eager")
	}
	v, ok = f.Vars["eager"]
	if !ok {
		t.Fatal("expected var eager")
	}
	if v.Value != "immediate" {
		t.Errorf("eager value = %q, want %q", v.Value, "immediate")
	}
	if !v.Eager {
		t.Error("eager should be true")
	}
}

func TestParseTargetAllProperties(t *testing.T) {
	content := "build: go\n" +
		"\tcmd=echo hello\n" +
		"\tbin=myapp\n" +
		"\tsources=*.go\n" +
		"\ttmp=/tmp/build\n" +
		"\trequire=dep1\n" +
		"\tarch=amd64\n" +
		"\tos=linux\n" +
		"\tdesc=My app\n" +
		"\tinstall=cp myapp /usr/bin\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "fiat")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	f, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(f.Targets))
	}
	tg := f.Targets[0]
	if tg.Name != "build" {
		t.Errorf("name = %q, want %q", tg.Name, "build")
	}
	if tg.Language != "go" {
		t.Errorf("language = %q, want %q", tg.Language, "go")
	}
	if len(tg.Cmds) != 1 || tg.Cmds[0] != "echo hello" {
		t.Errorf("cmds = %v, want [echo hello]", tg.Cmds)
	}
	if tg.Bin != "myapp" {
		t.Errorf("bin = %q, want %q", tg.Bin, "myapp")
	}
	if tg.Sources != "*.go" {
		t.Errorf("sources = %q, want %q", tg.Sources, "*.go")
	}
	if len(tg.Tmp) != 1 || tg.Tmp[0] != "/tmp/build" {
		t.Errorf("tmp = %v, want [/tmp/build]", tg.Tmp)
	}
	if len(tg.Requires) != 1 || tg.Requires[0] != "dep1" {
		t.Errorf("requires = %v, want [dep1]", tg.Requires)
	}
	if len(tg.Arch) != 1 || tg.Arch[0] != "amd64" {
		t.Errorf("arch = %v, want [amd64]", tg.Arch)
	}
	if len(tg.OS) != 1 || tg.OS[0] != "linux" {
		t.Errorf("os = %v, want [linux]", tg.OS)
	}
	if tg.Desc != "My app" {
		t.Errorf("desc = %q, want %q", tg.Desc, "My app")
	}
	if len(tg.Install) != 1 || tg.Install[0] != "cp myapp /usr/bin" {
		t.Errorf("install = %v, want [cp myapp /usr/bin]", tg.Install)
	}
}

func TestParseMultiLineContinuation(t *testing.T) {
	content := "build: go\n" +
		"\tcmd=echo first\n" +
		"\t\tsecond\n" +
		"\tbin=myapp-\n" +
		"\t\tdebug\n" +
		"\tsources=*.go\n" +
		"\t\textra.go\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "fiat")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	f, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(f.Targets))
	}
	tg := f.Targets[0]
	if len(tg.Cmds) != 2 || tg.Cmds[0] != "echo first" || tg.Cmds[1] != "second" {
		t.Errorf("cmds = %v, want [echo first second]", tg.Cmds)
	}
	if tg.Bin != "myapp- debug" {
		t.Errorf("bin = %q, want %q", tg.Bin, "myapp- debug")
	}
	if tg.Sources != "*.go extra.go" {
		t.Errorf("sources = %q, want %q", tg.Sources, "*.go extra.go")
	}
}

func TestParseInlineComment(t *testing.T) {
	content := "build: go\n" +
		"\tcmd=echo hi # inline comment\n" +
		"\tdesc=useful # end\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "fiat")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	f, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(f.Targets))
	}
	tg := f.Targets[0]
	if len(tg.Cmds) != 1 || tg.Cmds[0] != "echo hi" {
		t.Errorf("cmds = %v, want [echo hi]", tg.Cmds)
	}
	if tg.Desc != "useful" {
		t.Errorf("desc = %q, want %q", tg.Desc, "useful")
	}
}

func TestParseTargetLocalVars(t *testing.T) {
	content := "build: go NAME=myapp VERSION=1.0\n" +
		"\tcmd=echo $NAME\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "fiat")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	f, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(f.Targets))
	}
	tg := f.Targets[0]
	if tg.Name != "build" {
		t.Errorf("name = %q, want %q", tg.Name, "build")
	}
	if tg.Language != "go" {
		t.Errorf("language = %q, want %q", tg.Language, "go")
	}
	if len(tg.Vars) != 2 {
		t.Fatalf("expected 2 target vars, got %d", len(tg.Vars))
	}
	if tg.Vars[0].Name != "NAME" || tg.Vars[0].Value != "myapp" {
		t.Errorf("vars[0] = %+v, want {Name:NAME Value:myapp}", tg.Vars[0])
	}
	if tg.Vars[1].Name != "VERSION" || tg.Vars[1].Value != "1.0" {
		t.Errorf("vars[1] = %+v, want {Name:VERSION Value:1.0}", tg.Vars[1])
	}
	if tg.Vars[0].Eager || tg.Vars[1].Eager {
		t.Error("target-local vars should not be eager")
	}
	if len(tg.Cmds) != 1 || tg.Cmds[0] != "echo $NAME" {
		t.Errorf("cmds = %v, want [echo $NAME]", tg.Cmds)
	}
}

func TestParseNonExistentPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent")
	_, err := Parse(path)
	if err == nil {
		t.Fatal("expected error for non-existent path")
	}
}

func TestParseDIRVariable(t *testing.T) {
	content := "$name=value\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "fiat")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	f, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	dv, ok := f.Vars["DIR"]
	if !ok {
		t.Fatal("expected DIR var")
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if dv.Value != absDir {
		t.Errorf("DIR = %q, want %q", dv.Value, absDir)
	}
}

func TestParseUnknownPropertyAsVar(t *testing.T) {
	content := "build: go\n" +
		"\tMY_KEY=myvalue\n" +
		"\tOTHER=otherval\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "fiat")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	f, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(f.Targets))
	}
	tg := f.Targets[0]
	if len(tg.Vars) != 2 {
		t.Fatalf("expected 2 target vars, got %d", len(tg.Vars))
	}
	found := false
	for _, v := range tg.Vars {
		if v.Name == "MY_KEY" && v.Value == "myvalue" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected MY_KEY=myvalue in target vars, got %+v", tg.Vars)
	}
}

func TestParseVirtualTarget(t *testing.T) {
	content := ".phony: \n" +
		"\tcmd=echo virtual\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "fiat")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	f, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(f.Targets))
	}
	tg := f.Targets[0]
	if !tg.IsVirtual {
		t.Error("expected virtual target")
	}
	if tg.Name != ".phony" {
		t.Errorf("name = %q, want %q", tg.Name, ".phony")
	}
}

func TestParseTargetWithNoLanguage(t *testing.T) {
	content := "install:\n" +
		"\tcmd=cp myapp /usr/bin\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "fiat")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	f, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(f.Targets))
	}
	tg := f.Targets[0]
	if tg.Language != "" {
		t.Errorf("language = %q, want empty", tg.Language)
	}
	if tg.Name != "install" {
		t.Errorf("name = %q, want %q", tg.Name, "install")
	}
}

func TestExpandBasic(t *testing.T) {
	vars := map[string]*Var{
		"name": {Name: "name", Value: "world"},
	}
	got := Expand("hello $name", vars, 0)
	want := "hello world"
	if got != want {
		t.Errorf("dollar form = %q, want %q", got, want)
	}
	got = Expand("hello $(name)", vars, 0)
	if got != want {
		t.Errorf("parens form = %q, want %q", got, want)
	}
}

func TestExpandParenthesised(t *testing.T) {
	vars := map[string]*Var{
		"bin": {Name: "bin", Value: "myapp"},
	}
	got := Expand("$(bin)-debug", vars, 0)
	want := "myapp-debug"
	if got != want {
		t.Errorf("suffix = %q, want %q", got, want)
	}
}

func TestExpandNested(t *testing.T) {
	vars := map[string]*Var{
		"base":  {Name: "base", Value: "$app"},
		"app":   {Name: "app", Value: "myapp"},
		"final": {Name: "final", Value: "$base"},
	}
	got := Expand("$final", vars, 0)
	want := "myapp"
	if got != want {
		t.Errorf("nested = %q, want %q", got, want)
	}
}

func TestExpandEscape(t *testing.T) {
	vars := map[string]*Var{}
	got := Expand("price is $$10", vars, 0)
	want := "price is $10"
	if got != want {
		t.Errorf("escape = %q, want %q", got, want)
	}
}

func TestExpandUnclosed(t *testing.T) {
	vars := map[string]*Var{}
	got := Expand("$(unclosed", vars, 0)
	want := "$(unclosed"
	if got != want {
		t.Errorf("unclosed = %q, want %q", got, want)
	}
}

func TestExpandDepth(t *testing.T) {
	vars := map[string]*Var{
		"a": {Name: "a", Value: "$b"},
		"b": {Name: "b", Value: "$c"},
		"c": {Name: "c", Value: "$d"},
		"d": {Name: "d", Value: "$e"},
		"e": {Name: "e", Value: "$f"},
		"f": {Name: "f", Value: "$g"},
		"g": {Name: "g", Value: "$h"},
		"h": {Name: "h", Value: "$i"},
		"i": {Name: "i", Value: "$j"},
		"j": {Name: "j", Value: "deep"},
	}
	got := Expand("$a", vars, 0)
	want := "deep"
	if got != want {
		t.Errorf("within limit = %q, want %q", got, want)
	}
	vars["j"].Value = "$k"
	vars["k"] = &Var{Name: "k", Value: "$l"}
	vars["l"] = &Var{Name: "l", Value: "too-deep"}
	got = Expand("$a", vars, 0)
	want = "$l"
	if got != want {
		t.Errorf("exceed limit = %q, want %q", got, want)
	}
}

func TestExpandUnknownVar(t *testing.T) {
	vars := map[string]*Var{}
	got := Expand("hello $name", vars, 0)
	want := "hello $name"
	if got != want {
		t.Errorf("unknown = %q, want %q", got, want)
	}
}

func TestExpandDollarAtEnd(t *testing.T) {
	vars := map[string]*Var{}
	got := Expand("trailing $", vars, 0)
	want := "trailing $"
	if got != want {
		t.Errorf("dollar at end = %q, want %q", got, want)
	}
}

func TestExpandNonIdentAfterDollar(t *testing.T) {
	vars := map[string]*Var{}
	got := Expand("hello $!", vars, 0)
	want := "hello $!"
	if got != want {
		t.Errorf("non-ident = %q, want %q", got, want)
	}
}

func TestExpandWithTarget(t *testing.T) {
	global := map[string]*Var{
		"prefix": {Name: "prefix", Value: "usr"},
	}
	tg := &Target{
		Name:    "testapp",
		Bin:     "myapp",
		Sources: "*.go",
		Arch:    []string{"arm64"},
		OS:      []string{"freebsd"},
	}
	got := ExpandWithTarget("$prefix/$bin-$arch-$os", global, tg)
	want := "usr/myapp-arm64-freebsd"
	if got != want {
		t.Errorf("ExpandWithTarget = %q, want %q", got, want)
	}
	got = ExpandWithTarget("$THIS", global, tg)
	if got != "testapp" {
		t.Errorf("THIS = %q, want %q", got, "testapp")
	}
	got = ExpandWithTarget("$sources", global, tg)
	if got != "*.go" {
		t.Errorf("sources = %q, want %q", got, "*.go")
	}
}

func TestExpandWithTargetDefaultArchOS(t *testing.T) {
	global := map[string]*Var{}
	tg := &Target{Name: "test"}
	got := ExpandWithTarget("$arch-$os", global, tg)
	want := runtime.GOARCH + "-" + runtime.GOOS
	if got != want {
		t.Errorf("defaults = %q, want %q", got, want)
	}
}

func TestExpandWithTargetLocalVars(t *testing.T) {
	global := map[string]*Var{}
	tg := &Target{
		Name: "test",
		Vars: []*Var{{Name: "OUT", Value: "bin/app"}},
	}
	got := ExpandWithTarget("$OUT", global, tg)
	want := "bin/app"
	if got != want {
		t.Errorf("local var = %q, want %q", got, want)
	}
}

func TestWriteRoundTrip(t *testing.T) {
	content := "$name=testapp\n$eager:=immediate\n\nbuild: go\n\tcmd=echo hi\n\tbin=myapp\n\tsources=*.go\n\tdesc=Test app\n\tarch=amd64\n\tos=linux\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "fiat")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	f1, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f1.Write(); err != nil {
		t.Fatal(err)
	}
	f2, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(f1.Targets) != len(f2.Targets) {
		t.Fatalf("target count mismatch: %d vs %d", len(f1.Targets), len(f2.Targets))
	}
	for i := range f1.Targets {
		t1 := f1.Targets[i]
		t2 := f2.Targets[i]
		if t1.Name != t2.Name {
			t.Errorf("target[%d] name: %q vs %q", i, t1.Name, t2.Name)
		}
		if t1.Language != t2.Language {
			t.Errorf("target[%d] language: %q vs %q", i, t1.Language, t2.Language)
		}
		if t1.Bin != t2.Bin {
			t.Errorf("target[%d] bin: %q vs %q", i, t1.Bin, t2.Bin)
		}
		if t1.Sources != t2.Sources {
			t.Errorf("target[%d] sources: %q vs %q", i, t1.Sources, t2.Sources)
		}
		if t1.Desc != t2.Desc {
			t.Errorf("target[%d] desc: %q vs %q", i, t1.Desc, t2.Desc)
		}
	}
	for name, v1 := range f1.Vars {
		v2, ok := f2.Vars[name]
		if !ok {
			t.Errorf("var %q missing after round-trip", name)
			continue
		}
		if v1.Value != v2.Value {
			t.Errorf("var %q value: %q vs %q", name, v1.Value, v2.Value)
		}
		if v1.Eager != v2.Eager {
			t.Errorf("var %q eager: %v vs %v", name, v1.Eager, v2.Eager)
		}
	}
}

func TestWriteNewFileAddTarget(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fiat")
	f := NewFile(path)
	tg := &Target{
		Name:     "build",
		Language: "go",
		Bin:      "myapp",
		Cmds:     []string{"echo hello"},
	}
	f.AddTarget(tg)
	if err := f.Write(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	output := string(data)
	if !strings.Contains(output, "build: go") {
		t.Errorf("output should contain 'build: go', got %q", output)
	}
	if !strings.Contains(output, "bin=myapp") {
		t.Errorf("output should contain 'bin=myapp', got %q", output)
	}
	if !strings.Contains(output, "cmd=echo hello") {
		t.Errorf("output should contain 'cmd=echo hello', got %q", output)
	}
}

func TestWriteNewFileAddTargetVirtual(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fiat")
	f := NewFile(path)
	tg := &Target{
		Name:      ".clean",
		IsVirtual: true,
		Cmds:      []string{"rm -rf build"},
	}
	f.AddTarget(tg)
	if err := f.Write(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	output := string(data)
	if !strings.Contains(output, ".clean:") {
		t.Errorf("output should contain '.clean:', got %q", output)
	}
}

func TestWriteNewFileAddTargetEagerVar(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fiat")
	f := NewFile(path)
	f.Vars["ver"] = &Var{Name: "ver", Value: "1.0", Eager: true}
	tg := &Target{Name: "build", Language: "go"}
	f.AddTarget(tg)
	if err := f.Write(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	output := string(data)
	if !strings.Contains(output, "$ver:=1.0") {
		t.Errorf("output should contain '$ver:=1.0', got %q", output)
	}
}

func TestWriteDefaultFileCreate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fiat")
	if err := WriteDefaultFile(path, false, false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "build: go\n" {
		t.Errorf("content = %q, want %q", string(data), "build: go\n")
	}
}

func TestWriteDefaultFileSkip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fiat")
	if err := os.WriteFile(path, []byte("existing content\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := WriteDefaultFile(path, false, false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "existing content\n" {
		t.Errorf("content = %q, want %q", string(data), "existing content\n")
	}
}

func TestWriteDefaultFileOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fiat")
	if err := os.WriteFile(path, []byte("existing content\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := WriteDefaultFile(path, true, false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "build: go\n" {
		t.Errorf("content = %q, want %q", string(data), "build: go\n")
	}
}

func TestWriteDefaultFileError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent", "fiat")
	if err := WriteDefaultFile(path, false, false); err == nil {
		t.Fatal("expected error for invalid path")
	}
}

func TestFindFiatExplicitPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mybuild.fiat")
	if err := os.WriteFile(path, []byte("build: go\n"), 0644); err != nil {
		t.Fatal(err)
	}
	p, ok := FindFiat(path)
	if !ok {
		t.Fatal("expected to find explicit path")
	}
	if p != path {
		t.Errorf("path = %q, want %q", p, path)
	}
}

func TestFindFiatExplicitPathMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.fiat")
	p, ok := FindFiat(path)
	if ok {
		t.Fatalf("expected not found, got path %q", p)
	}
	if p != "" {
		t.Errorf("path should be empty, got %q", p)
	}
}

func TestFindFiatDefaultName(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "fiat"), []byte("build: go\n"), 0644); err != nil {
		t.Fatal(err)
	}
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)
	p, ok := FindFiat("")
	if !ok {
		t.Fatal("expected to find fiat file")
	}
	if p != "fiat" {
		t.Errorf("path = %q, want %q", p, "fiat")
	}
}

func TestFindFiatGlobMatch(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "build.fiat"), []byte("build: go\n"), 0644); err != nil {
		t.Fatal(err)
	}
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)
	p, ok := FindFiat("")
	if !ok {
		t.Fatal("expected to find *.fiat file")
	}
	if p != "build.fiat" {
		t.Errorf("path = %q, want %q", p, "build.fiat")
	}
}

func TestFindFiatNoMatch(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)
	p, ok := FindFiat("")
	if ok {
		t.Fatalf("expected not found, got path %q", p)
	}
	if p != "" {
		t.Errorf("path should be empty, got %q", p)
	}
}

func TestFindTarget(t *testing.T) {
	f := &File{
		Targets: []*Target{
			{Name: "build"},
			{Name: "test"},
		},
	}
	tg := FindTarget(f, "build")
	if tg == nil {
		t.Fatal("expected to find target build")
	}
	if tg.Name != "build" {
		t.Errorf("name = %q, want %q", tg.Name, "build")
	}
	tg = FindTarget(f, "nonexistent")
	if tg != nil {
		t.Errorf("expected nil, got %+v", tg)
	}
}

func TestSplitLines(t *testing.T) {
	got := splitLines("hello")
	if len(got) != 1 || got[0] != "hello" {
		t.Errorf("no newline: got %v, want [hello]", got)
	}
	got = splitLines("hello\nworld")
	if len(got) != 2 || got[0] != "hello" || got[1] != "world" {
		t.Errorf("one newline: got %v, want [hello world]", got)
	}
	got = splitLines("a\nb\nc")
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Errorf("multiple newlines: got %v, want [a b c]", got)
	}
	got = splitLines("")
	if len(got) != 1 || got[0] != "" {
		t.Errorf("empty: got %v, want ['']", got)
	}
}

func TestIsIdent(t *testing.T) {
	if !util.IsIdent('a') {
		t.Error("expected 'a' to be ident")
	}
	if !util.IsIdent('Z') {
		t.Error("expected 'Z' to be ident")
	}
	if !util.IsIdent('0') {
		t.Error("expected '0' to be ident")
	}
	if !util.IsIdent('_') {
		t.Error("expected '_' to be ident")
	}
	if util.IsIdent('$') {
		t.Error("expected '$' not to be ident")
	}
	if util.IsIdent('-') {
		t.Error("expected '-' not to be ident")
	}
	if util.IsIdent(' ') {
		t.Error("expected ' ' not to be ident")
	}
	if util.IsIdent(0) {
		t.Error("expected null byte not to be ident")
	}
}

func TestPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fiat")
	f := NewFile(path)
	if f.Path() != path {
		t.Errorf("Path() = %q, want %q", f.Path(), path)
	}
}

func TestParseMalformedVar(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fiat")
	content := "$NOEQUALS\nbuild: go\n\tcmd=go build\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Parse(path)
	if err == nil {
		t.Fatal("expected error for malformed variable (no = sign)")
	}
}

func TestParseDeepNesting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fiat")
	content := "$a=$b\n$b=$c\n$c=$d\n$d=$e\n$e=$f\n$f=$g\n$g=$h\n$h=$i\n$i=$j\n$j=deep\nbuild: go\n\tcmd=go build\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	f, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	a := f.Vars["a"]
	if a == nil {
		t.Fatal("expected var a")
	}
	got := Expand(a.Value, f.Vars, 0)
	if got != "deep" {
		t.Errorf("nested = %q, want %q", got, "deep")
	}
}

func TestParseUnicodeIdent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fiat")
	content := "$café=yes\nbuild: go\n\tcmd=go build\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	f, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	c := f.Vars["café"]
	if c == nil {
		t.Fatal("expected var café")
	}
	if c.Value != "yes" {
		t.Errorf("café = %q, want %q", c.Value, "yes")
	}
}

func TestParseNumberedVar(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fiat")
	content := "$VER=1.0\nbuild: go\n\tcmd=go build\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	f, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	v := f.Vars["VER"]
	if v == nil {
		t.Fatal("expected var VER")
	}
	if v.Value != "1.0" {
		t.Errorf("VER = %q, want %q", v.Value, "1.0")
	}
}
