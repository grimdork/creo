package fiat

import (
	"fmt"
	"os"
	"strings"

	"github.com/grimdork/climate/fx"
)

// Write serialises the file back to disk, preserving existing structure where possible and appending new targets or vars.
func (f *File) Write() error {
	var b strings.Builder
	covered := make(map[int]bool)
	coveredVars := make(map[string]bool)

	for _, seg := range f.segs {
		switch seg.kind {
		case segBlank, segComment:
			for _, line := range seg.raw {
				b.WriteString(line)
				b.WriteByte('\n')
			}

		case segVar:
			coveredVars[seg.varName] = true
			// Write reconstructed var line if the var still exists
			if v, ok := f.Vars[seg.varName]; ok {
				sep := "="
				if v.Eager {
					sep = ":="
				}
				b.WriteByte('$')
				b.WriteString(v.Name)
				b.WriteString(sep)
				b.WriteString(v.Value)
				b.WriteByte('\n')
				if len(seg.raw) > 1 {
					for _, extra := range seg.raw[1:] {
						b.WriteString(extra)
						b.WriteByte('\n')
					}
				}
			}

		case segTarget:
			covered[seg.targetIdx] = true
			// Write raw lines verbatim
			for _, line := range seg.raw {
				b.WriteString(line)
				b.WriteByte('\n')
			}
		}
	}

	// Append new targets (no matching segment)
	for i, t := range f.Targets {
		if covered[i] {
			continue
		}
		serializeTarget(&b, t)
	}

	// Append vars not covered by any segment
	for _, v := range f.Vars {
		if coveredVars[v.Name] {
			continue
		}
		if v.Name == "DIR" {
			continue
		}
		sep := "="
		if v.Eager {
			sep = ":="
		}
		b.WriteByte('$')
		b.WriteString(v.Name)
		b.WriteString(sep)
		b.WriteString(v.Value)
		b.WriteByte('\n')
	}

	data := b.String()
	// Ensure trailing newline
	if !strings.HasSuffix(data, "\n") {
		data += "\n"
	}
	return os.WriteFile(f.path, []byte(data), 0644)
}

func serializeTarget(b *strings.Builder, t *Target) {
	if t.IsVirtual && !strings.HasPrefix(t.Name, ".") {
		b.WriteByte('.')
	}
	b.WriteString(t.Name)
	b.WriteByte(':')
	if t.Language != "" {
		b.WriteByte(' ')
		b.WriteString(t.Language)
		if t.LangAlias != "" {
			b.WriteByte(':')
			b.WriteString(t.LangAlias)
		}
	}
	for _, v := range t.Vars {
		b.WriteByte(' ')
		b.WriteString(v.Name)
		sep := "="
		if v.Eager {
			sep = ":="
		}
		b.WriteString(sep)
		b.WriteString(v.Value)
	}
	b.WriteByte('\n')

	props := targetProps(t)
	for _, p := range props {
		writeProp(b, p.key, p.value)
	}
}

type prop struct {
	key   string
	value string
}

func targetProps(t *Target) []prop {
	var props []prop
	add := func(k, v string) {
		if v != "" {
			props = append(props, prop{k, v})
		}
	}
	add("desc", t.Desc)
	for _, cmd := range t.Cmds {
		add("cmd", cmd)
	}
	add("bin", t.Bin)
	add("sources", t.Sources)
	for _, tmp := range t.Tmp {
		add("tmp", tmp)
	}
	for _, req := range t.Requires {
		add("require", req)
	}
	for _, a := range t.Arch {
		add("arch", a)
	}
	for _, o := range t.OS {
		add("os", o)
	}
	for _, inst := range t.Install {
		add("install", inst)
	}
	for _, v := range t.Vars {
		// Skip known fields that are already handled
		switch v.Name {
		case "arch", "os", "desc", "cmd", "bin", "sources", "tmp", "require", "install":
			continue
		}
		add(v.Name, v.Value)
	}
	return props
}

func writeProp(b *strings.Builder, key, value string) {
	lines := splitLines(value)
	for i, line := range lines {
		if i == 0 {
			b.WriteByte('\t')
		} else {
			b.WriteByte('\t')
			b.WriteByte('\t')
		}
		b.WriteString(key)
		b.WriteByte('=')
		b.WriteString(line)
		b.WriteByte('\n')
	}
}

func splitLines(s string) []string {
	if !strings.Contains(s, "\n") {
		return []string{s}
	}
	return strings.Split(s, "\n")
}

// AddTarget appends a target to the file's target list.
func (f *File) AddTarget(t *Target) {
	f.Targets = append(f.Targets, t)
}

// NewFile returns a new File with the given path and an initialised vars map.
func NewFile(path string) *File {
	return &File{
		path: path,
		Vars: make(map[string]*Var),
	}
}

// Path returns the filesystem path of the file.
func (f *File) Path() string {
	return f.path
}

// WriteDefaultFile writes a default "build: go" fiat file to path, skipping or replacing existing files per force.
func WriteDefaultFile(path string, force, verbose bool) error {
	if _, err := os.Stat(path); err == nil {
		if force {
			if err := os.WriteFile(path, []byte("build: go\n"), 0644); err != nil {
				return fmt.Errorf(errWriting, path, err)
			}
			if verbose {
				fx.Println("{success}  Replaced fiat{@}")
			}
		} else if verbose {
			fx.Println("{warning}  Skipped fiat (already exists){@}")
		}
	} else {
		if err := os.WriteFile(path, []byte("build: go\n"), 0644); err != nil {
			return fmt.Errorf(errWriting, path, err)
		}
		if verbose {
			fx.Println("{success}  Created fiat{@}")
		}
	}
	return nil
}
