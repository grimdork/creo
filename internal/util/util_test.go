package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsIdent(t *testing.T) {
	tests := []struct {
		c    byte
		want bool
	}{
		{'a', true},
		{'z', true},
		{'A', true},
		{'Z', true},
		{'0', true},
		{'9', true},
		{'_', true},
		{'-', false},
		{'.', false},
		{'$', false},
		{'/', false},
		{'\n', false},
	}
	for _, tt := range tests {
		got := IsIdent(tt.c)
		if got != tt.want {
			t.Errorf("IsIdent(%q) = %v, want %v", tt.c, got, tt.want)
		}
	}
}

func TestUnique(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"empty", nil, nil},
		{"single", []string{"a"}, []string{"a"}},
		{"dedup", []string{"a", "b", "a", "c", "b"}, []string{"a", "b", "c"}},
		{"all same", []string{"x", "x", "x"}, []string{"x"}},
		{"preserves order", []string{"c", "a", "b", "a", "c"}, []string{"c", "a", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Unique(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("Unique() = %v (len %d), want %v (len %d)", got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("Unique()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestFmtSize(t *testing.T) {
	tests := []struct {
		size int64
		want string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{1023, "1023 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{1048576, "1.0 MiB"},
		{1073741824, "1.0 GiB"},
		{1610612736, "1.5 GiB"},
	}
	for _, tt := range tests {
		got := FmtSize(tt.size)
		if got != tt.want {
			t.Errorf("FmtSize(%d) = %q, want %q", tt.size, got, tt.want)
		}
	}
}

func TestGlobFilesNoGlob(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.txt")
	os.WriteFile(f1, []byte("a"), 0644)
	f2 := filepath.Join(dir, "b.go")
	os.WriteFile(f2, []byte("b"), 0644)

	matches, err := GlobFiles("*.txt", dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0] != f1 {
		t.Errorf("GlobFiles('*.txt') = %v, want [%s]", matches, f1)
	}
}

func TestGlobFilesDoubleStar(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	os.MkdirAll(sub, 0755)
	f1 := filepath.Join(dir, "root.txt")
	os.WriteFile(f1, []byte("r"), 0644)
	f2 := filepath.Join(sub, "nested.txt")
	os.WriteFile(f2, []byte("n"), 0644)

	matches, err := GlobFiles("**/*.txt", dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 {
		t.Errorf("GlobFiles('**/*.txt') = %v, want 2 matches", matches)
	}
}

func TestGlobFilesNoMatches(t *testing.T) {
	dir := t.TempDir()
	matches, err := GlobFiles("*.xyz", dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Errorf("GlobFiles('*.xyz') = %v, want empty", matches)
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	content := []byte("hello, copy test")
	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(dir, "sub", "dst.txt")
	if err := CopyFile(src, dst); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Errorf("CopyFile content = %q, want %q", got, content)
	}
}

func TestCopyFileSamePath(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f.txt")
	os.WriteFile(f, []byte("x"), 0644)

	if err := CopyFile(f, f); err != nil {
		t.Errorf("CopyFile same path: %v", err)
	}
}

func TestCopyFileSrcMissing(t *testing.T) {
	dir := t.TempDir()
	err := CopyFile(filepath.Join(dir, "nonexistent"), filepath.Join(dir, "out"))
	if err == nil {
		t.Error("CopyFile missing src: expected error, got nil")
	}
}

func TestCopyFilePreservesMode(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "exec.sh")
	os.WriteFile(src, []byte("#!/bin/sh\n"), 0755)

	dst := filepath.Join(dir, "bin", "exec.sh")
	if err := CopyFile(src, dst); err != nil {
		t.Fatal(err)
	}

	si, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if si.Mode()&os.ModePerm != 0755 {
		t.Errorf("CopyFile mode = %o, want 0755", si.Mode()&os.ModePerm)
	}
}
