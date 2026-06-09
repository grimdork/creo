package runner

import (
	"fmt"
	"os"
	"strings"

	"github.com/grimdork/creo/internal/fiat"
)

func RenderGraph(f *fiat.File, dir string, format string, checkStatus bool) (string, error) {
	switch format {
	case "tree":
		return renderTree(f, dir, checkStatus)
	case "dot":
		return renderDOT(f, dir, checkStatus)
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
	sources := collectFilePaths(t, f, dir)
	if checkCache(dir, t.Name, sources, t.Cmds) {
		return "cached"
	}
	return "stale"
}

func findRoots(f *fiat.File) []*fiat.Target {
	required := map[string]bool{}
	for _, t := range f.Targets {
		for _, dep := range t.Requires {
			required[dep] = true
		}
	}
	var roots []*fiat.Target
	for _, t := range f.Targets {
		if !required[t.Name] {
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

func renderTree(f *fiat.File, dir string, checkStatus bool) (string, error) {
	roots := findRoots(f)
	if len(roots) == 0 {
		// all targets are in cycles or everyone has a dependant
		// show every target as a root
		roots = f.Targets
	}

	var b strings.Builder
	for i, root := range roots {
		if i > 0 {
			b.WriteByte('\n')
		}
		renderTreeNode(f, dir, root, &b, "", "", "", checkStatus, map[string]bool{})
		_ = i
	}
	return b.String(), nil
}

func renderTreeNode(f *fiat.File, dir string, t *fiat.Target, b *strings.Builder, indent, connector, childIndent string, checkStatus bool, ancestors map[string]bool) {
	if ancestors[t.Name] {
		fmt.Fprintf(b, "%s%s%s [circular]\n", indent, connector, targetLabel(t))
		return
	}
	ancestors[t.Name] = true
	defer delete(ancestors, t.Name)

	label := targetLabel(t)
	if checkStatus {
		switch targetStatus(f, t, dir) {
		case "cached":
			label += " [cached]"
		case "stale":
			label += " [stale]"
		}
	}
	fmt.Fprintf(b, "%s%s%s\n", indent, connector, label)

	for i, dep := range t.Requires {
		dt := fiat.FindTarget(f, dep)
		if dt == nil {
			fmt.Fprintf(b, "%s%s%s [missing]\n", childIndent, depConnector(i, len(t.Requires)), dep)
			continue
		}
		grandchildIndent := childIndent + "│   "
		if i == len(t.Requires)-1 {
			grandchildIndent = childIndent + "    "
		}
		renderTreeNode(f, dir, dt, b, childIndent, depConnector(i, len(t.Requires)), grandchildIndent, checkStatus, ancestors)
	}
}

func depConnector(i, total int) string {
	if total == 1 {
		return "└── "
	}
	if i == total-1 {
		return "└── "
	}
	return "├── "
}

func renderDOT(f *fiat.File, dir string, checkStatus bool) (string, error) {
	var b strings.Builder
	b.WriteString("digraph \"creo\" {\n")
	b.WriteString("\trankdir=LR;\n")
	b.WriteString("\tnode [shape=box];\n")

	for _, t := range f.Targets {
		extra := ""
		if checkStatus {
			switch targetStatus(f, t, dir) {
			case "cached":
				extra = ",color=green,fontcolor=green"
			case "stale":
				extra = ",color=darkorange,fontcolor=darkorange"
			}
		}
		fmt.Fprintf(&b, "\t\"%s\" [label=\"%s\"%s];\n", t.Name, targetLabel(t), extra)
	}

	for _, t := range f.Targets {
		for _, dep := range t.Requires {
			fmt.Fprintf(&b, "\t\"%s\" -> \"%s\";\n", t.Name, dep)
		}
	}

	b.WriteString("}\n")
	return b.String(), nil
}
