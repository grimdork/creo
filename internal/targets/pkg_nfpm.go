package targets

import (
	"fmt"
	"strings"

	"github.com/grimdork/creo/internal/fiat"
)

type nfpmPkg struct {
	maintainer  string
	vendor      string
	homepage    string
	license     string
	section     string
	priority    string
	description string
	depends     []string
	recommends  []string
	suggests    []string
	files       []fiat.ManifestFile
	scripts     map[string]string
}

func readNfpmConfig(f *fiat.File, t *fiat.Target) *nfpmPkg {
	m := lookupManifest(f, t)

	p := &nfpmPkg{
		maintainer:  m.Maintainer,
		vendor:      m.Vendor,
		homepage:    m.Homepage,
		license:     m.License,
		section:     m.Section,
		priority:    m.Priority,
		description: m.Description,
		depends:     m.Depends,
		recommends:  m.Recommends,
		suggests:    m.Suggests,
		files:       m.Files,
		scripts:     m.Scripts,
	}

	for _, v := range t.Vars {
		val := fiat.Expand(v.Value, f.Vars, 0)
		switch v.Name {
		case "maintainer":
			p.maintainer = val
		case "vendor":
			p.vendor = val
		case "homepage":
			p.homepage = val
		case "license":
			p.license = val
		case "section":
			p.section = val
		case "priority":
			p.priority = val
		case "description", "desc":
			p.description = val
		}
	}

	if p.maintainer == "" {
		p.maintainer = DefMaintainer
	}
	if p.vendor == "" {
		p.vendor = projectName(f)
	}
	if p.license == "" {
		p.license = DefLicense
	}
	if p.section == "" {
		p.section = DefSection
	}
	if p.priority == "" {
		p.priority = DefPriority
	}

	return p
}

func applyDeb(f *fiat.File, t *fiat.Target) {
	applyNfpm(f, t, "deb", "deb")
}

func applyRpm(f *fiat.File, t *fiat.Target) {
	applyNfpm(f, t, "rpm", "rpm")
}

func applyNfpm(f *fiat.File, t *fiat.Target, packager, ext string) {
	p := readNfpmConfig(f, t)
	proj := projectName(f)
	ver := versionClean(f)
	bd := BuildDir(f)
	defBin := fmt.Sprintf("%s/%s_%s_$arch.%s", bd, proj, ver, ext)
	t.Bin = expandBin(f, t, defBin)

	yamlContent := buildNfpmYAML(t, proj, ver, p)

	cmds := []string{
		"mkdir -p .creo",
		"cat > .creo/$THIS.yaml << 'YAML'",
		yamlContent,
		"YAML",
		"nfpm pkg --config .creo/$THIS.yaml --packager " + packager + " --target $bin",
		"rm -f .creo/$THIS.yaml",
	}

	t.Cmds = cmds
	if len(t.Tmp) == 0 {
		t.Tmp = []string{".creo/$THIS.yaml"}
	}
}

func buildNfpmYAML(t *fiat.Target, proj, ver string, p *nfpmPkg) string {
	var b strings.Builder

	str := func(v string) string {
		if v == "" {
			return `""`
		}
		return fmt.Sprintf("%q", v)
	}

	fmt.Fprintf(&b, "name: %s\n", proj)
	fmt.Fprintf(&b, "arch: $arch\n")
	fmt.Fprintf(&b, "version: %s\n", str(ver))
	fmt.Fprintf(&b, "maintainer: %s\n", str(p.maintainer))
	if p.description != "" {
		fmt.Fprintf(&b, "description: %s\n", str(p.description))
	}
	if p.vendor != "" {
		fmt.Fprintf(&b, "vendor: %s\n", str(p.vendor))
	}
	if p.homepage != "" {
		fmt.Fprintf(&b, "homepage: %s\n", str(p.homepage))
	}
	fmt.Fprintf(&b, "license: %s\n", str(p.license))
	fmt.Fprintf(&b, "section: %s\n", p.section)
	fmt.Fprintf(&b, "priority: %s\n", p.priority)

	b.WriteString("contents:\n")

	writeEntry := func(src, dst string, mode string) {
		fmt.Fprintf(&b, "  - src: %s\n", src)
		fmt.Fprintf(&b, "    dst: %s\n", dst)
		if mode != "" {
			fmt.Fprintf(&b, "    file_info:\n")
			fmt.Fprintf(&b, "      mode: %s\n", mode)
		}
	}

	dep := firstDep(t)
	writeEntry("$OUTPUT_"+dep, "/usr/bin/"+proj, "0755")

	hasReadme := false
	hasLicense := false
	for _, ef := range p.files {
		dstLower := strings.ToLower(ef.Dst)
		if strings.HasSuffix(dstLower, "/readme") || strings.HasSuffix(dstLower, "/readme.md") {
			hasReadme = true
		}
		if strings.HasSuffix(dstLower, "/license") || strings.HasSuffix(dstLower, "/license.md") {
			hasLicense = true
		}
	}

	if !hasReadme {
		writeEntry("README.md", "/usr/share/doc/"+proj+"/README.md", "0644")
	}
	if !hasLicense {
		writeEntry("LICENSE", "/usr/share/doc/"+proj+"/LICENSE", "0644")
	}

	for _, ef := range p.files {
		writeEntry(ef.Src, ef.Dst, "")
	}

	if len(p.depends) > 0 {
		b.WriteString("depends:\n")
		for _, d := range p.depends {
			fmt.Fprintf(&b, "  - %s\n", d)
		}
	}
	if len(p.recommends) > 0 {
		b.WriteString("recommends:\n")
		for _, d := range p.recommends {
			fmt.Fprintf(&b, "  - %s\n", d)
		}
	}
	if len(p.suggests) > 0 {
		b.WriteString("suggests:\n")
		for _, d := range p.suggests {
			fmt.Fprintf(&b, "  - %s\n", d)
		}
	}
	if len(p.scripts) > 0 {
		b.WriteString("scripts:\n")
		for typ, scriptPath := range p.scripts {
			fmt.Fprintf(&b, "  %s: %s\n", typ, scriptPath)
		}
	}

	return b.String()
}
