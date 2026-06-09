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
	case "svg":
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

func langFill(lang string) string {
	switch lang {
	case "go":
		return "#e3f2fd"
	case "oci":
		return "#fff3e0"
	case "c", "cxx", "cpp":
		return "#e8f5e9"
	default:
		return "#f5f5f5"
	}
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func renderSVG(f *fiat.File, dir string, checkStatus bool) (string, error) {
	const (
		nw = 160
		nh = 44
		lg = 80
		ng = 30
		pd = 40
	)

	layers := map[int][]string{}
	layerOf := map[string]int{}

	var compLayer func(name string, stack map[string]bool) int
	compLayer = func(name string, stack map[string]bool) int {
		if stack[name] {
			return 0
		}
		if l, ok := layerOf[name]; ok {
			return l
		}
		t := fiat.FindTarget(f, name)
		if t == nil || len(t.Requires) == 0 {
			layerOf[name] = 0
			return 0
		}
		stack[name] = true
		maxDep := 0
		for _, dep := range t.Requires {
			dl := compLayer(dep, stack) + 1
			if dl > maxDep {
				maxDep = dl
			}
		}
		delete(stack, name)
		layerOf[name] = maxDep
		return maxDep
	}

	for _, t := range f.Targets {
		compLayer(t.Name, map[string]bool{})
	}

	for name, l := range layerOf {
		layers[l] = append(layers[l], name)
	}

	maxW := 0
	numLayers := len(layers)
	for l := 0; l < numLayers; l++ {
		names := layers[l]
		if len(names) > maxW {
			maxW = len(names)
		}
	}
	totalW := maxW*(nw+ng) - ng + 2*pd
	totalH := numLayers*(nh+lg) - lg + 2*pd

	type lnode struct {
		x, y int
	}
	nodes := map[string]*lnode{}
	for l := 0; l < numLayers; l++ {
		names := layers[l]
		y := pd + l*(nh+lg)
		layerW := len(names)*(nw+ng) - ng
		startX := pd + (totalW-2*pd-layerW)/2
		for i, name := range names {
			x := startX + i*(nw+ng)
			nodes[name] = &lnode{x: x, y: y}
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, `<svg viewBox="0 0 %d %d" xmlns="http://www.w3.org/2000/svg">`+"\n", totalW, totalH)
	b.WriteString("<defs>\n")
	b.WriteString(`<marker id="a" viewBox="0 0 10 10" refX="10" refY="5" markerWidth="7" markerHeight="7" orient="auto"><path d="M0 0L10 5L0 10z" fill="#666"/></marker>` + "\n")
	b.WriteString("</defs>\n")

	for _, t := range f.Targets {
		pr := nodes[t.Name]
		if pr == nil {
			continue
		}
		x1 := pr.x + nw/2
		y1 := pr.y + nh
		for _, dep := range t.Requires {
			ch := nodes[dep]
			if ch == nil {
				continue
			}
			x2 := ch.x + nw/2
			y2 := ch.y
			fmt.Fprintf(&b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#666" stroke-width="1.5" marker-end="url(#a)"/>`+"\n", x1, y1, x2, y2)
		}
	}

	for _, t := range f.Targets {
		n, ok := nodes[t.Name]
		if !ok {
			continue
		}
		fill := langFill(t.Language)
		stroke := "#9e9e9e"
		if checkStatus {
			switch targetStatus(f, t, dir) {
			case "cached":
				stroke = "#4caf50"
			case "stale":
				stroke = "#ff9800"
			}
		}
		fmt.Fprintf(&b, `<rect x="%d" y="%d" width="%d" height="%d" rx="6" fill="%s" stroke="%s" stroke-width="2"/>`+"\n", n.x, n.y, nw, nh, fill, stroke)
		tx := n.x + nw/2
		ty := n.y + nh/2
		fmt.Fprintf(&b, `<text x="%d" y="%d" text-anchor="middle" dominant-baseline="central" font-family="sans-serif" font-size="12" fill="#333">%s</text>`+"\n", tx, ty, xmlEscape(targetLabel(t)))
	}

	b.WriteString("</svg>\n")
	return b.String(), nil
}

func dotEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
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
		fmt.Fprintf(&b, "\t\"%s\" [label=\"%s\"%s];\n", dotEscape(t.Name), dotEscape(targetLabel(t)), extra)
	}

	for _, t := range f.Targets {
		for _, dep := range t.Requires {
			fmt.Fprintf(&b, "\t\"%s\" -> \"%s\";\n", dotEscape(t.Name), dotEscape(dep))
		}
	}

	b.WriteString("}\n")
	return b.String(), nil
}
