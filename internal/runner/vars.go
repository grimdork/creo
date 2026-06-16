package runner

import (
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
