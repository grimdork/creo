package lang

import (
	"os"
	"strings"
)

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

func parseProperty(line string, t *Target) string {
	eager := false
	sep := "="
	if idx := strings.Index(line, ":="); idx >= 0 {
		eager = true
		sep = ":="
	}

	parts := strings.SplitN(line, sep, 2)
	if len(parts) < 2 {
		return ""
	}
	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

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
	case "arch":
		t.Arch = strings.Fields(value)
	case "os":
		t.OS = strings.Fields(value)
	case "desc":
		t.Desc = value
	case "install":
		t.Install = append(t.Install, value)
	default:
		t.Vars = append(t.Vars, &Var{Name: key, Value: value, Eager: eager})
	}
	return key
}

func ParseFiat(path string) (*FiatFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	f := &FiatFile{
		Path: path,
		Vars: make(map[string]*Var),
	}

	var cur *Target
	var lastKey string
	lines := strings.Split(string(data), "\n")

	for i, raw := range lines {
		if idx := strings.IndexByte(raw, '#'); idx >= 0 {
			raw = raw[:idx]
		}
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
				lang := ""
				if len(parts) > 1 {
					lang = strings.TrimSpace(parts[1])
				}
				cur = &Target{Name: name, Language: lang, Line: i + 1}
				f.Targets = append(f.Targets, cur)
				lastKey = ""
				continue
			}
		}

		if cur != nil && isIndented(raw) {
			if strings.HasPrefix(raw, "\t\t") {
				switch lastKey {
				case "cmd":
					cur.Cmds = append(cur.Cmds, line)
				case "bin":
					cur.Bin += " " + line
				case "sources":
					cur.Sources += " " + line
				case "tmp":
					cur.Tmp = append(cur.Tmp, strings.Fields(line)...)
				case "require":
					cur.Requires = append(cur.Requires, strings.Fields(line)...)
				case "arch":
					cur.Arch = append(cur.Arch, strings.Fields(line)...)
				case "os":
					cur.OS = append(cur.OS, strings.Fields(line)...)
			case "desc":
				cur.Desc += " " + line
			case "install":
				cur.Install = append(cur.Install, line)
			default:
					for _, v := range cur.Vars {
						if v.Name == lastKey {
							v.Value += " " + line
							break
						}
					}
				}
			} else {
				lastKey = parseProperty(line, cur)
			}
			continue
		}
	}

	for _, v := range f.Vars {
		if v.Eager {
			v.Value = Expand(v.Value, f.Vars, 0)
		}
	}
	for _, t := range f.Targets {
		for _, v := range t.Vars {
			if v.Eager {
				v.Value = Expand(v.Value, f.Vars, 0)
			}
		}
	}

	return f, nil
}

func FindTarget(f *FiatFile, name string) *Target {
	for _, t := range f.Targets {
		if t.Name == name {
			return t
		}
	}
	return nil
}
