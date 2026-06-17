package targets

import (
	"strings"

	"github.com/grimdork/creo/internal/fiat"
)

func firstDep(t *fiat.Target) string {
	if t == nil {
		return "build"
	}
	if len(t.Requires) > 0 {
		return t.Requires[0]
	}
	return "build"
}

func versionClean(f *fiat.File) string {
	if v, ok := f.Vars["VERSION"]; ok && v.Value != "" {
		return strings.TrimPrefix(v.Value, "v")
	}
	return "0.0.0"
}
