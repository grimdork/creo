package runner

import (
	"fmt"
	"strings"

	"github.com/grimdork/creo/internal/fiat"
)

func renderTree(f *fiat.File, dir string, checkStatus bool) (string, error) {
	roots := findRoots(f)
	if len(roots) == 0 {
		roots = f.Targets
	}

	var b strings.Builder
	for i, root := range roots {
		if i > 0 {
			b.WriteByte('\n')
		}
		renderTreeNode(f, dir, root, &b, "", "", "", checkStatus, map[string]bool{})
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
