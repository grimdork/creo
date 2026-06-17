package cli

import (
	"os"
	"strings"

	"github.com/grimdork/climate/fx"
	"github.com/grimdork/creo/internal/targets"
)

// RunInit initialises a project with source files for the given languages, or
// applies a named template when tmplName and a single language are provided.
func RunInit(langs []string, tmplName, defineVars string, force, verbose bool) error {
	extra := parseDefine(defineVars)
	if tmplName != "" && len(langs) == 1 {
		return targets.InitProjectWithTemplate(langs[0], tmplName, extra, force, verbose)
	}
	if tmplName != "" && len(langs) != 1 {
		fx.Fprint(os.Stderr, "{warning}--template ignored: requires exactly one language{@}\n")
	}
	return targets.InitProject(langs, extra, force, verbose)
}

func parseDefine(raw string) map[string]string {
	if raw == "" {
		return nil
	}
	m := make(map[string]string)
	for _, pair := range strings.Split(raw, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		idx := strings.IndexByte(pair, '=')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(pair[:idx])
		val := strings.TrimSpace(pair[idx+1:])
		if key != "" {
			m[key] = val
		}
	}
	return m
}
