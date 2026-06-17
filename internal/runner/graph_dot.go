package runner

import (
	"fmt"
	"strings"

	"github.com/grimdork/creo/internal/fiat"
)

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
