package fiat

import (
	"os"
	"path/filepath"
	"strings"
)

type segKind int

const (
	segBlank segKind = iota
	segComment
	segVar
	segTarget
)

type segment struct {
	kind      segKind
	raw       []string
	varName   string
	targetIdx int
}

func isIndented(line string) bool {
	return len(line) > 0 && (line[0] == ' ' || line[0] == '\t')
}

func parseVarLine(line string, f *File, t *Target) {
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
	if key == "" {
		return ""
	}
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

func Parse(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	f := &File{
		path: path,
		Vars: make(map[string]*Var),
	}

	var curSeg *segment
	var curTarget *Target
	var lastKey string
	rawLines := strings.Split(string(data), "\n")

	flushSeg := func() {
		if curSeg != nil {
			f.segs = append(f.segs, *curSeg)
			curSeg = nil
		}
	}

	for _, raw := range rawLines {
		stripped := raw
		if idx := strings.IndexByte(stripped, '#'); idx >= 0 {
			stripped = stripped[:idx]
		}
		line := strings.TrimSpace(stripped)

		isBlank := line == ""
		isHash := len(raw) > 0 && raw[0] == '#'
		isVar := strings.HasPrefix(line, "$")
		isTarget := !isIndented(raw) && strings.Contains(line, ":") && strings.IndexByte(line, ':') > 0

		if isBlank {
			if curSeg == nil || curSeg.kind != segBlank {
				flushSeg()
				curSeg = &segment{kind: segBlank}
			}
			curSeg.raw = append(curSeg.raw, raw)
			continue
		}

		if isHash {
			if curSeg == nil || curSeg.kind != segComment {
				flushSeg()
				curSeg = &segment{kind: segComment}
			}
			curSeg.raw = append(curSeg.raw, raw)
			continue
		}

		if isVar && curTarget == nil {
			flushSeg()
			curSeg = &segment{kind: segVar}
			curSeg.raw = append(curSeg.raw, raw)
			parseVarLine(line, f, nil)
			rest := line[1:]
			sep := "="
			if idx := strings.Index(rest, ":="); idx >= 0 {
				sep = ":="
			}
			curSeg.varName = strings.TrimSpace(strings.SplitN(rest, sep, 2)[0])
			continue
		}

		if isTarget {
			flushSeg()
			parts := strings.SplitN(line, ":", 2)
			name := strings.TrimSpace(parts[0])
			if name != "" {
				virtual := name[0] == '.'
				curTarget = &Target{Name: name, IsVirtual: virtual}
				if len(parts) > 1 {
					tokens := strings.Fields(parts[1])
					if len(tokens) > 0 {
						curTarget.Language = tokens[0]
						for _, token := range tokens[1:] {
							if kv := strings.SplitN(token, "=", 2); len(kv) == 2 {
								curTarget.Vars = append(curTarget.Vars, &Var{Name: kv[0], Value: kv[1]})
							}
						}
					}
				}
				f.Targets = append(f.Targets, curTarget)
				curSeg = &segment{
					kind:      segTarget,
					targetIdx: len(f.Targets) - 1,
				}
				curSeg.raw = append(curSeg.raw, raw)
				lastKey = ""
			}
			continue
		}

		if curTarget != nil && isIndented(raw) {
			if curSeg != nil && curSeg.kind == segTarget {
				curSeg.raw = append(curSeg.raw, raw)
			}
			if strings.HasPrefix(raw, "\t\t") {
				switch lastKey {
				case "cmd":
					curTarget.Cmds = append(curTarget.Cmds, line)
				case "bin":
					if curTarget.Bin != "" {
						curTarget.Bin += " "
					}
					curTarget.Bin += line
				case "sources":
					if curTarget.Sources != "" {
						curTarget.Sources += " "
					}
					curTarget.Sources += line
				case "tmp":
					curTarget.Tmp = append(curTarget.Tmp, strings.Fields(line)...)
				case "require":
					curTarget.Requires = append(curTarget.Requires, strings.Fields(line)...)
				case "arch":
					curTarget.Arch = append(curTarget.Arch, strings.Fields(line)...)
				case "os":
					curTarget.OS = append(curTarget.OS, strings.Fields(line)...)
				case "desc":
					if curTarget.Desc != "" {
						curTarget.Desc += " "
					}
					curTarget.Desc += line
				case "install":
					curTarget.Install = append(curTarget.Install, line)
				default:
					for _, v := range curTarget.Vars {
						if v.Name == lastKey {
							v.Value += " " + line
							break
						}
					}
				}
			} else {
				lastKey = parseProperty(line, curTarget)
			}
			continue
		}

		if curSeg == nil || curSeg.kind != segComment {
			flushSeg()
			curSeg = &segment{kind: segComment}
		}
		curSeg.raw = append(curSeg.raw, raw)
	}
	flushSeg()

	dir := filepath.Dir(path)
	absDir, err := filepath.Abs(dir)
	if err == nil {
		f.Vars["DIR"] = &Var{Name: "DIR", Value: absDir}
	} else {
		f.Vars["DIR"] = &Var{Name: "DIR", Value: dir}
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

func FindTarget(f *File, name string) *Target {
	for _, t := range f.Targets {
		if t.Name == name {
			return t
		}
	}
	return nil
}
