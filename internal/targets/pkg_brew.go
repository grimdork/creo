package targets

import (
	"strings"

	"github.com/grimdork/creo/internal/fiat"
)

func applyBrew(f *fiat.File, t *fiat.Target) {
	m := lookupManifest(f, t)

	proj := projectName(f)
	bd := BuildDir(f)

	cfg := &fiat.BrewConfig{
		ClassName: formulaClassName(proj),
		Desc:      t.Desc,
		License:   "MIT",
		Output:    bd + "/" + proj + ".rb",
	}

	if m.Homepage != "" {
		cfg.Homepage = m.Homepage
	}
	if m.License != "" {
		cfg.License = m.License
	}
	if t.Desc != "" {
		cfg.Desc = t.Desc
	} else if m.Description != "" {
		cfg.Desc = m.Description
	}

	for _, v := range t.Vars {
		val := fiat.Expand(v.Value, f.Vars, 0)
		switch v.Name {
		case "tap":
			cfg.Tap = val
		case "homepage":
			cfg.Homepage = val
		case "license":
			cfg.License = val
		case "output":
			cfg.Output = val
		case "repo":
			cfg.Repo = val
		case "token":
			cfg.Token = val
		}
	}

	t.Arch = []string{"arm64"}
	t.OS = []string{"darwin"}
	t.Bin = expandBin(f, t, cfg.Output)
	t.Brew = cfg
	if len(t.Tmp) == 0 {
		t.Tmp = []string{".creo/$THIS-tap"}
	}
}

func formulaClassName(name string) string {
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '-' || r == '_' || r == '.'
	})
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}
