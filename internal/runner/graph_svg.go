package runner

import (
	"fmt"
	"strings"

	"github.com/grimdork/creo/internal/fiat"
	"github.com/grimdork/creo/internal/targets"
)

func langFill(langName string) string {
	switch langName {
	case targets.LangGo:
		return "#e3f2fd"
	case targets.LangOCI:
		return "#fff3e0"
	case targets.LangC, targets.LangCxx, targets.LangCpp:
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
