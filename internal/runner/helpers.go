package runner

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/grimdork/creo/internal/fiat"
)

// baseComboVars merges file vars, target vars, arch/os vars, and dependency output paths into a single variable map.
func baseComboVars(f *fiat.File, t *fiat.Target, activeArch, activeOS string, outputs *Outputs) map[string]*fiat.Var {
	comboVars := make(map[string]*fiat.Var)
	for k, v := range f.Vars {
		comboVars[k] = v
	}
	for _, v := range t.Vars {
		comboVars[v.Name] = v
	}
	comboVars["arch"] = &fiat.Var{Name: "arch", Value: activeArch}
	comboVars["os"] = &fiat.Var{Name: "os", Value: activeOS}
	comboVars["THIS"] = &fiat.Var{Name: "THIS", Value: t.Name}
	for _, dep := range t.Requires {
		if binPath := outputs.Load(dep, activeArch, activeOS); binPath != "" {
			comboVars["OUTPUT_"+dep] = &fiat.Var{Name: "OUTPUT_" + dep, Value: binPath}
		}
	}
	return comboVars
}

// archOrEmpty returns the given arch slice or a slice containing the empty string if nil/empty.
func archOrEmpty(a []string) []string {
	if len(a) == 0 {
		return []string{""}
	}
	return a
}

// osOrEmpty returns the given OS slice or a slice containing the empty string if nil/empty.
func osOrEmpty(o []string) []string {
	if len(o) == 0 {
		return []string{""}
	}
	return o
}

// ensureArch returns runtime.GOARCH when a is empty, otherwise a.
func ensureArch(a string) string {
	if a == "" {
		return runtime.GOARCH
	}
	return a
}

// ensureOS returns runtime.GOOS when o is empty, otherwise o.
func ensureOS(o string) string {
	if o == "" {
		return runtime.GOOS
	}
	return o
}

// hasCombo reports whether the arch and os appear together in the cross-product of archs and oses.
func hasCombo(archs, oses []string, arch, os string) bool {
	for _, a := range archs {
		for _, o := range oses {
			if a == arch && o == os {
				return true
			}
		}
	}
	return false
}

// execCredHelper runs a credential helper command and returns the user:pass pair from its output.
func execCredHelper(helper, dir string) (user, pass string, err error) {
	parts := strings.Fields(helper)
	if len(parts) == 0 {
		return "", "", fmt.Errorf("empty credential helper")
	}
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = dir
	out, execErr := cmd.Output()
	if execErr != nil {
		return "", "", fmt.Errorf("%s: %w", helper, execErr)
	}
	line := strings.TrimSpace(string(out))
	idx := strings.IndexByte(line, ':')
	if idx < 0 {
		return "", line, nil
	}
	return line[:idx], line[idx+1:], nil
}
