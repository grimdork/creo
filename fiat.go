package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Var struct {
	Name  string
	Value string
	Eager bool
}

type Target struct {
	Name     string
	Line     int
	Cmds     []string
	Bin      string
	Sources  string
	Tmp      []string
	Requires []string
	Vars     []*Var
}

type FiatFile struct {
	Path    string
	Vars    map[string]*Var
	Targets []*Target
}

func isIndented(line string) bool {
	return len(line) > 0 && (line[0] == ' ' || line[0] == '\t')
}

func parseVar(line string, f *FiatFile, t *Target) {
	rest := line[1:]

	eager := false
	sep := "="
	if idx := strings.Index(rest, ":="); idx >= 0 {
		eager = true
		sep = ":="
	}

	parts := strings.SplitN(rest, sep, 2)
	if len(parts) < 2 {
		return
	}
	name := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	v := &Var{Name: name, Value: value, Eager: eager}
	if t != nil {
		t.Vars = append(t.Vars, v)
	} else {
		f.Vars[name] = v
	}
}

func parseProperty(line string, t *Target) {
	eager := false
	sep := "="
	if idx := strings.Index(line, ":="); idx >= 0 {
		eager = true
		sep = ":="
	}

	parts := strings.SplitN(line, sep, 2)
	if len(parts) < 2 {
		return
	}
	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	if eager {
		value = expandVars(value, nil, nil)
	}

	switch key {
	case "cmd":
		t.Cmds = append(t.Cmds, value)
	case "bin":
		t.Bin = value
	case "sources":
		t.Sources = value
	case "tmp":
		t.Tmp = strings.Fields(value)
	case "require":
		t.Requires = strings.Fields(value)
	default:
		t.Vars = append(t.Vars, &Var{Name: key, Value: value, Eager: eager})
	}
}

func expandVars(s string, global map[string]*Var, target []*Var) string {
	vars := make(map[string]*Var)
	for k, v := range global {
		vars[k] = v
	}
	for _, v := range target {
		vars[v.Name] = v
	}
	return expand(s, vars, 0)
}

func expand(s string, vars map[string]*Var, depth int) string {
	if depth > 10 {
		return s
	}

	var out strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '$' {
			out.WriteByte(s[i])
			continue
		}
		if i+1 < len(s) && s[i+1] == '$' {
			out.WriteByte('$')
			i++
			continue
		}
		j := i + 1
		for j < len(s) && isIdent(s[j]) {
			j++
		}
		if j > i+1 {
			name := s[i+1 : j]
			if v, ok := vars[name]; ok {
				out.WriteString(expand(v.Value, vars, depth+1))
			} else {
				out.WriteString(s[i:j])
			}
			i = j - 1
		} else {
			out.WriteByte('$')
		}
	}
	return out.String()
}

func isIdent(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_'
}

func parseFiat(path string) (*FiatFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	f := &FiatFile{
		Path: path,
		Vars: make(map[string]*Var),
	}

	var cur *Target
	lines := strings.Split(string(data), "\n")

	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "$") {
			parseVar(line, f, cur)
			continue
		}

		if !isIndented(raw) && strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			name := strings.TrimSpace(parts[0])
			if name != "" {
				cur = &Target{Name: name, Line: i + 1}
				f.Targets = append(f.Targets, cur)
				continue
			}
		}

		if cur != nil && isIndented(raw) {
			parseProperty(line, cur)
			continue
		}
	}

	for _, v := range f.Vars {
		if v.Eager {
			v.Value = expand(v.Value, f.Vars, 0)
		}
	}
	for _, t := range f.Targets {
		for _, v := range t.Vars {
			if v.Eager {
				v.Value = expand(v.Value, f.Vars, 0)
			}
		}
	}

	return f, nil
}

func findTarget(f *FiatFile, name string) *Target {
	for _, t := range f.Targets {
		if t.Name == name {
			return t
		}
	}
	return nil
}

func findFiat() (string, bool) {
	if _, err := os.Stat("fiat"); err == nil {
		return "fiat", true
	}

	matches, err := filepath.Glob("*.fiat")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error scanning for .fiat files:", err)
		return "", false
	}

	if len(matches) == 0 {
		fmt.Fprintln(os.Stderr, "No .fiat files found")
		return "", false
	}

	sort.Strings(matches)

	if len(matches) == 1 {
		return matches[0], true
	}

	selected, err := Run(matches)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Picker error:", err)
		return "", false
	}
	if selected == "" {
		fmt.Fprintln(os.Stderr, "Cancelled")
		return "", false
	}
	return selected, true
}

func findFiatInDir(dir string) (string, bool) {
	path := filepath.Join(dir, "fiat")
	if _, err := os.Stat(path); err == nil {
		return path, true
	}

	matches, err := filepath.Glob(filepath.Join(dir, "*.fiat"))
	if err != nil {
		return "", false
	}

	if len(matches) == 1 {
		return matches[0], true
	}

	if len(matches) > 1 {
		fmt.Printf("  Skipped %s (multiple .fiat files)\n", dir)
	}
	return "", false
}
