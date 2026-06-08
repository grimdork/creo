package lang

import (
	"runtime"
	"strings"
)

func IsIdent(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_'
}

func Expand(s string, vars map[string]*Var, depth int) string {
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
		if i+1 < len(s) && s[i+1] == '(' {
			j := i + 2
			for j < len(s) && s[j] != ')' {
				j++
			}
			if j < len(s) && j > i+2 {
				name := s[i+2 : j]
				if v, ok := vars[name]; ok {
					out.WriteString(Expand(v.Value, vars, depth+1))
				} else {
					out.WriteString(s[i : j+1])
				}
				i = j
				continue
			}
			out.WriteString("$(")
			i++
			continue
		}
		j := i + 1
		for j < len(s) && IsIdent(s[j]) {
			j++
		}
		if j > i+1 {
			name := s[i+1 : j]
			if v, ok := vars[name]; ok {
				out.WriteString(Expand(v.Value, vars, depth+1))
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

func ExpandWithTarget(s string, global map[string]*Var, t *Target) string {
	vars := make(map[string]*Var)
	for k, v := range global {
		vars[k] = v
	}
	for _, v := range t.Vars {
		vars[v.Name] = v
	}
	if t.Bin != "" {
		vars["bin"] = &Var{Name: "bin", Value: Expand(t.Bin, vars, 0)}
	}
	if t.Sources != "" {
		vars["sources"] = &Var{Name: "sources", Value: Expand(t.Sources, vars, 0)}
	}
	arch := runtime.GOARCH
	if len(t.Arch) > 0 {
		arch = t.Arch[0]
	}
	osval := runtime.GOOS
	if len(t.OS) > 0 {
		osval = t.OS[0]
	}
	vars["arch"] = &Var{Name: "arch", Value: arch}
	vars["os"] = &Var{Name: "os", Value: osval}
	vars["THIS"] = &Var{Name: "THIS", Value: t.Name}
	return Expand(s, vars, 0)
}
