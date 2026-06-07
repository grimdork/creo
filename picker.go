package main

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

type picker struct {
	items    []string
	filtered []int
	cursor   int
	prefix   string
	width    int
	height   int
	fd       int
	oldState *term.State
}

func (p *picker) filter() {
	prefix := strings.ToLower(p.prefix)
	p.filtered = p.filtered[:0]
	for i, item := range p.items {
		if strings.HasPrefix(strings.ToLower(item), prefix) {
			p.filtered = append(p.filtered, i)
		}
	}
	if p.cursor >= len(p.filtered) {
		p.cursor = 0
	}
}

func (p *picker) render() {
	var out strings.Builder
	out.WriteString("\x1b[H\x1b[J")

	listHeight := p.height - 2
	if listHeight < 1 {
		listHeight = 1
	}

	start := 0
	if p.cursor >= listHeight {
		start = p.cursor - listHeight + 1
	}
	end := start + listHeight
	if end > len(p.filtered) {
		end = len(p.filtered)
	}

	for i := start; i < end; i++ {
		idx := p.filtered[i]
		if i == p.cursor {
			out.WriteString("> ")
		} else {
			out.WriteString("  ")
		}
		out.WriteString(p.items[idx])
		out.WriteString("\x1b[K\n")
	}

	out.WriteString("\x1b[KFilter: ")
	out.WriteString(p.prefix)

	fmt.Fprint(os.Stdout, out.String())
}

func (p *picker) clear() {
	fmt.Fprint(os.Stdout, "\x1b[H\x1b[J")
}

func (p *picker) run() string {
	in := os.Stdin
	buf := make([]byte, 3)

	p.render()

	for {
		n, err := in.Read(buf[:1])
		if err != nil || n == 0 {
			continue
		}

		switch buf[0] {
		case '\r', '\n':
			if len(p.filtered) > 0 {
				p.clear()
				return p.items[p.filtered[p.cursor]]
			}

		case '\x1b':
			n, _ = in.Read(buf[:1])
			if n == 0 || buf[0] != '[' {
				p.clear()
				return ""
			}
			n, _ = in.Read(buf[:1])
			if n == 0 {
				p.clear()
				return ""
			}
			switch buf[0] {
			case 'A':
				if p.cursor > 0 {
					p.cursor--
				}
			case 'B':
				if p.cursor < len(p.filtered)-1 {
					p.cursor++
				}
			}
			p.render()

		case '\x03':
			p.clear()
			term.Restore(p.fd, p.oldState)
			os.Exit(1)

		case '\b', '\x7f':
			if len(p.prefix) > 0 {
				p.prefix = p.prefix[:len(p.prefix)-1]
				p.filter()
				p.render()
			}

		default:
			if buf[0] >= ' ' {
				p.prefix += string(buf[0])
				p.filter()
				p.render()
			}
		}
	}
}

// Run displays a terminal-based picker for the given items and returns the
// selected item, or empty string if cancelled.
func Run(items []string) (string, error) {
	if len(items) == 0 {
		return "", nil
	}

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", err
	}

	w, h, err := term.GetSize(fd)
	if err != nil {
		w, h = 80, 24
	}

	p := &picker{
		items:    items,
		filtered: make([]int, len(items)),
		cursor:   0,
		prefix:   "",
		width:    w,
		height:   h,
		fd:       fd,
		oldState: oldState,
	}
	for i := range items {
		p.filtered[i] = i
	}

	selected := p.run()

	err = term.Restore(fd, oldState)
	if err != nil {
		return "", err
	}

	return selected, nil
}
