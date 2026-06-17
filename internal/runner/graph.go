package runner

import (
	"fmt"
	"os"

	"github.com/grimdork/creo/internal/fiat"
	"github.com/grimdork/creo/internal/util"
)

const (
	FormatGraphTree = "tree"
	FormatGraphDot  = "dot"
	FormatGraphSVG  = "svg"
)

func ValidGraphFormat(format string) bool {
	return format == FormatGraphTree || format == FormatGraphDot || format == FormatGraphSVG
}

func RenderGraph(f *fiat.File, dir string, format string, checkStatus bool) (string, error) {
	switch format {
	case FormatGraphTree:
		return renderTree(f, dir, checkStatus)
	case FormatGraphDot:
		return renderDOT(f, dir, checkStatus)
	case FormatGraphSVG:
		return renderSVG(f, dir, checkStatus)
	default:
		return "", fmt.Errorf("unknown graph format %q", format)
	}
}

func targetStatus(f *fiat.File, t *fiat.Target, dir string) string {
	if t.Sources == "" || t.Bin == "" {
		return ""
	}
	bin := fiat.ExpandWithTarget(t.Bin, f.Vars, t)
	if _, err := os.Stat(bin); err != nil {
		if _, err := os.Stat(cachePath(dir, t.Name)); err == nil {
			return "stale"
		}
		return ""
	}
	if _, err := os.Stat(cachePath(dir, t.Name)); err != nil {
		return ""
	}
	sources, err := collectFilePaths(t, f, dir)
	if err != nil {
		return "stale"
	}
	if checkCache(dir, t.Name, sources, t.Cmds) {
		return "cached"
	}
	return "stale"
}

func findRoots(f *fiat.File) []*fiat.Target {
	required := util.NewSet[string]()
	for _, t := range f.Targets {
		for _, dep := range t.Requires {
			required.Add(dep)
		}
	}
	var roots []*fiat.Target
	for _, t := range f.Targets {
		if !required.Has(t.Name) {
			roots = append(roots, t)
		}
	}
	return roots
}

func targetLabel(t *fiat.Target) string {
	if t.Language != "" {
		return t.Name + " (" + t.Language + ")"
	}
	return t.Name
}
