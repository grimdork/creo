package targets

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/grimdork/creo/internal/fiat"
)

// PackageManifest holds metadata and file lists parsed from a manifest.ini
// file. It describes package metadata, dependencies, extra files, scripts,
// and architecture-specific overrides for deb/rpm/archive/brew targets.
type PackageManifest struct {
	Maintainer  string
	Vendor      string
	Homepage    string
	License     string
	Section     string
	Priority    string
	Description string

	Depends    []string
	Recommends []string
	Suggests   []string

	Files     []fiat.ManifestFile
	Downloads []fiat.ManifestFile
	Scripts   map[string]string

	ArchOverrides map[string]*PackageManifest
}

type rawSection struct {
	name   string
	fields map[string][]string // key → []value (preserves case)
	order  []string
}

func parseManifest(path string) (*PackageManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	raw := parseRawINI(string(data))
	m := &PackageManifest{
		Scripts:       make(map[string]string),
		ArchOverrides: make(map[string]*PackageManifest),
	}

	for _, sec := range raw {
		switch sec.name {
		case "package":
			for _, k := range sec.order {
				vals := sec.fields[k]
				if len(vals) == 0 {
					continue
				}
				v := vals[0]
				switch k {
				case "maintainer":
					m.Maintainer = v
				case "vendor":
					m.Vendor = v
				case "homepage":
					m.Homepage = v
				case "license":
					m.License = v
				case "section":
					m.Section = v
				case "priority":
					m.Priority = v
				case "description":
					m.Description = v
				default:
					// Unknown package keys are ignored.
				}
			}

		case "depends":
			for _, k := range sec.order {
				for _, v := range sec.fields[k] {
					if v == "" {
						m.Depends = append(m.Depends, k)
					} else {
						m.Depends = append(m.Depends, k+" "+v)
					}
				}
			}

		case "recommends":
			for _, k := range sec.order {
				for _, v := range sec.fields[k] {
					if v == "" {
						m.Recommends = append(m.Recommends, k)
					} else {
						m.Recommends = append(m.Recommends, k+" "+v)
					}
				}
			}

		case "suggests":
			for _, k := range sec.order {
				for _, v := range sec.fields[k] {
					if v == "" {
						m.Suggests = append(m.Suggests, k)
					} else {
						m.Suggests = append(m.Suggests, k+" "+v)
					}
				}
			}

		case "files":
			for _, k := range sec.order {
				for _, v := range sec.fields[k] {
					m.Files = append(m.Files, fiat.ManifestFile{Dst: k, Src: v})
				}
			}

		case "download":
			for _, k := range sec.order {
				for _, v := range sec.fields[k] {
					m.Downloads = append(m.Downloads, fiat.ManifestFile{Dst: v, Src: k})
				}
			}

		case "scripts":
			for _, k := range sec.order {
				vals := sec.fields[k]
				if len(vals) > 0 {
					m.Scripts[k] = vals[0]
				}
			}

		default:
			if strings.HasPrefix(sec.name, "arch:") {
				archName := strings.TrimPrefix(sec.name, "arch:")
				if archName == "" {
					continue
				}
				om := &PackageManifest{
					Scripts: make(map[string]string),
				}
				for _, k := range sec.order {
					vals := sec.fields[k]
					if len(vals) == 0 {
						continue
					}
					v := vals[0]
					switch k {
					case "maintainer":
						om.Maintainer = v
					case "vendor":
						om.Vendor = v
					case "homepage":
						om.Homepage = v
					case "license":
						om.License = v
					case "section":
						om.Section = v
					case "priority":
						om.Priority = v
					case "description":
						om.Description = v
					case "depends":
						om.Depends = append(om.Depends, v)
					case "recommends":
						om.Recommends = append(om.Recommends, v)
					case "suggests":
						om.Suggests = append(om.Suggests, v)
					default:
						if k == "preinstall" || k == "postinstall" || k == "preremove" || k == "postremove" {
							om.Scripts[k] = v
						} else {
							om.Files = append(om.Files, fiat.ManifestFile{Dst: k, Src: v})
						}
					}
				}
				m.ArchOverrides[archName] = om
			}
		}
	}

	return m, nil
}

func parseRawINI(data string) []rawSection {
	var sections []rawSection
	var cur *rawSection

	lines := strings.Split(data, "\n")
	for _, line := range lines {
		raw := strings.TrimRight(line, "\r")
		trimmed := strings.TrimSpace(raw)

		if trimmed == "" {
			continue
		}

		if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";") {
			continue
		}

		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			name := trimmed[1 : len(trimmed)-1]
			cur = &rawSection{name: name, fields: make(map[string][]string)}
			sections = append(sections, *cur)
			continue
		}

		if cur == nil {
			continue
		}

		idx := strings.IndexByte(raw, '=')
		if idx < 0 {
			continue
		}

		key := strings.TrimSpace(raw[:idx])
		value := strings.TrimSpace(raw[idx+1:])

		if key == "" {
			continue
		}

		cur.fields[key] = append(cur.fields[key], value)
		cur.order = append(cur.order, key)
	}

	return sections
}

func defaultManifest(proj string) *PackageManifest {
	return &PackageManifest{
		Vendor:   proj,
		License:  "MIT",
		Section:  "contrib",
		Priority: "extra",
		Scripts:  make(map[string]string),
	}
}

func lookupManifest(f *fiat.File, t *fiat.Target) *PackageManifest {
	proj := projectName(f)

	for _, v := range t.Vars {
		if v.Name == "manifest" && v.Value != "" {
			expanded := fiat.Expand(v.Value, f.Vars, 0)
			path := expanded
			if !filepath.IsAbs(path) {
				path = filepath.Join(filepath.Dir(f.Path()), path)
			}
			if _, err := os.Stat(path); err == nil {
				m, err := parseManifest(path)
				if err != nil {
					return defaultManifest(proj)
				}
				return m
			}
			return defaultManifest(proj)
		}
	}

	defaultPath := filepath.Join(filepath.Dir(f.Path()), "manifest.ini")
	if _, err := os.Stat(defaultPath); err == nil {
		m, err := parseManifest(defaultPath)
		if err != nil {
			return defaultManifest(proj)
		}
		return m
	}

	return defaultManifest(proj)
}

func projectName(f *fiat.File) string {
	if v, ok := f.Vars["PROJECT"]; ok && v.Value != "" {
		return v.Value
	}
	return filepath.Base(filepath.Dir(f.Path()))
}
