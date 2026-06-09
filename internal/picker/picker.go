package picker

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/unix"
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

	maxW := 0
	for _, idx := range p.filtered {
		if len(p.items[idx]) > maxW {
			maxW = len(p.items[idx])
		}
	}

	boxW := maxW + 6
	innerW := boxW - 2
	if boxW > p.width-4 {
		boxW = p.width - 4
	}
	if boxW < 20 {
		boxW = 20
		innerW = boxW - 2
	}

	visibleItems := p.height - 4
	if visibleItems < 1 {
		visibleItems = 1
	}

	start := 0
	if p.cursor >= visibleItems {
		start = p.cursor - visibleItems + 1
	}
	end := start + visibleItems
	if end > len(p.filtered) {
		end = len(p.filtered)
	}

	boxH := end - start + 4
	topRow := (p.height - boxH) / 2
	leftCol := (p.width - boxW) / 2
	if topRow < 0 {
		topRow = 0
	}
	if leftCol < 0 {
		leftCol = 0
	}

	lead := strings.Repeat(" ", leftCol)
	out.WriteString(fmt.Sprintf("\x1b[%d;1H", topRow+1))

	writeLine := func(content string) {
		out.WriteString(lead)
		out.WriteString(content)
		out.WriteString("\x1b[K\r\n")
	}

	drawBorder := func(left, mid, right string) {
		writeLine(left + strings.Repeat(mid, innerW) + right)
	}

	drawBorder("┌", "─", "┐")

	for i := start; i < end; i++ {
		idx := p.filtered[i]
		sel := "  "
		if i == p.cursor {
			sel = "> "
		}
		content := " " + sel + p.items[idx]
		pad := innerW - len(content)
		if pad > 0 {
			content += strings.Repeat(" ", pad)
		}
		writeLine("│" + content + "│")
	}

	drawBorder("├", "─", "┤")

	filterContent := " Filter: " + p.prefix
	pad := innerW - len(filterContent)
	if pad > 0 {
		filterContent += strings.Repeat(" ", pad)
	}
	writeLine("│" + filterContent + "│")

	drawBorder("└", "─", "┘")

	fmt.Fprint(os.Stdout, out.String())
}

func (p *picker) run() string {
	in := os.Stdin
	buf := make([]byte, 3)

	p.render()

	for {
		n, err := in.Read(buf[:1])
		if err != nil {
			return ""
		}
		if n == 0 {
			continue
		}

		switch buf[0] {
		case '\r', '\n':
			if len(p.filtered) > 0 {
				return p.items[p.filtered[p.cursor]]
			}

		case '\x1b':
			fd := int(in.Fd())
			oldFlags, fErr := unix.FcntlInt(uintptr(fd), unix.F_GETFL, 0)
			if fErr == nil {
				unix.FcntlInt(uintptr(fd), unix.F_SETFL, oldFlags|unix.O_NONBLOCK)
				n, _ = in.Read(buf[:1])
				unix.FcntlInt(uintptr(fd), unix.F_SETFL, oldFlags)
				if n == 0 {
					return ""
				}
			} else {
				_, _ = in.Read(buf[:1])
			}
			if buf[0] != '[' {
				return ""
			}
			n, _ = in.Read(buf[:1])
			if n == 0 {
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
			term.Restore(p.fd, p.oldState)
			fmt.Fprint(os.Stdout, "\x1b[?1049l")
			return ""

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
	defer term.Restore(fd, oldState)

	fmt.Fprint(os.Stdout, "\x1b[?1049h")
	defer fmt.Fprint(os.Stdout, "\x1b[?1049l")

	w, h, err := term.GetSize(fd)
	if err != nil {
		w, h = 80, 24
	}

	p := &picker{
		items:    items,
		filtered: make([]int, len(items)),
		width:    w,
		height:   h,
		fd:       fd,
		oldState: oldState,
	}
	for i := range items {
		p.filtered[i] = i
	}

	return p.run(), nil
}
