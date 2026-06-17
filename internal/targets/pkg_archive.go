package targets

import (
	"fmt"
	"path/filepath"

	"github.com/grimdork/creo/internal/fiat"
)

func applyArchive(f *fiat.File, t *fiat.Target) {
	m := lookupManifest(f, t)

	format := "tar.gz"
	for _, v := range t.Vars {
		if v.Name == "format" && v.Value != "" {
			format = fiat.Expand(v.Value, f.Vars, 0)
		}
	}

	proj := projectName(f)
	ver := versionClean(f)
	bd := BuildDir(f)
	defBin := fmt.Sprintf("%s/%s_%s_$os_$arch.tar.gz", bd, proj, ver)
	if format == "zip" {
		defBin = fmt.Sprintf("%s/%s_%s_$os_$arch.zip", bd, proj, ver)
	}
	t.Bin = expandBin(f, t, defBin)

	dep := firstDep(t)
	cmds := []string{
		"mkdir -p .creo/$THIS-staging",
		"cp $OUTPUT_" + dep + " .creo/$THIS-staging/$PROJECT",
		"chmod +x .creo/$THIS-staging/$PROJECT",
	}

	for _, ef := range m.Files {
		dstName := filepath.Base(ef.Dst)
		cmds = append(cmds, "cp "+ef.Src+" .creo/$THIS-staging/"+dstName)
	}

	cmds = append(cmds,
		"test -f README.md && cp README.md .creo/$THIS-staging/ || true",
		"test -f LICENSE && cp LICENSE .creo/$THIS-staging/ || true",
	)

	if format == "zip" {
		cmds = append(cmds, "(cd .creo/$THIS-staging && zip -r ../../$bin .)")
	} else {
		cmds = append(cmds, "tar -czf $bin -C .creo/$THIS-staging .")
	}
	cmds = append(cmds, "rm -rf .creo/$THIS-staging")

	t.Cmds = cmds
	if len(t.Tmp) == 0 {
		t.Tmp = []string{".creo/$THIS-staging"}
	}
}
