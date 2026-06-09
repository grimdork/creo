package picker

import (
	"os"
	"testing"
)

func TestFilter(t *testing.T) {
	items := []string{"apple", "banana", "cherry", "avocado"}
	p := &picker{
		items:    items,
		filtered: make([]int, len(items)),
		cursor:   0,
		width:    80,
		height:   24,
	}
	for i := range items {
		p.filtered[i] = i
	}

	// Filter with "a" — should select ["apple", "avocado"]
	p.prefix = "a"
	p.filter()
	if len(p.filtered) != 2 {
		t.Fatalf("expected 2 filtered items for 'a', got %d", len(p.filtered))
	}
	if p.items[p.filtered[0]] != "apple" {
		t.Fatalf("expected first filtered item 'apple', got %q", p.items[p.filtered[0]])
	}
	if p.items[p.filtered[1]] != "avocado" {
		t.Fatalf("expected second filtered item 'avocado', got %q", p.items[p.filtered[1]])
	}
	if p.cursor != 0 {
		t.Fatalf("expected cursor 0 after filter, got %d", p.cursor)
	}

	// Filter with "b" — should select ["banana"]
	p.prefix = "b"
	p.filter()
	if len(p.filtered) != 1 {
		t.Fatalf("expected 1 filtered item for 'b', got %d", len(p.filtered))
	}
	if p.items[p.filtered[0]] != "banana" {
		t.Fatalf("expected filtered item 'banana', got %q", p.items[p.filtered[0]])
	}

	// Filter with "z" — should select nothing
	p.prefix = "z"
	p.filter()
	if len(p.filtered) != 0 {
		t.Fatalf("expected 0 filtered items for 'z', got %d", len(p.filtered))
	}

	// Filter with "" — should select all items
	p.prefix = ""
	p.filter()
	if len(p.filtered) != 4 {
		t.Fatalf("expected 4 filtered items for empty prefix, got %d", len(p.filtered))
	}

	// Case insensitive: "A" matches "apple" and "avocado"
	p.prefix = "A"
	p.filter()
	if len(p.filtered) != 2 {
		t.Fatalf("expected 2 filtered items for 'A', got %d", len(p.filtered))
	}
	if p.items[p.filtered[0]] != "apple" {
		t.Fatalf("expected first filtered item 'apple', got %q", p.items[p.filtered[0]])
	}
	if p.items[p.filtered[1]] != "avocado" {
		t.Fatalf("expected second filtered item 'avocado', got %q", p.items[p.filtered[1]])
	}

	// Cursor reset: cursor out of bounds after filter should be set to 0
	p.cursor = 5
	p.filter()
	if p.cursor != 0 {
		t.Fatalf("expected cursor reset to 0, got %d", p.cursor)
	}
}

func TestPickerRunEmptyItems(t *testing.T) {
	result, err := Run([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Fatalf("expected empty string, got %q", result)
	}
}

func TestPickerRenderNoPanic(t *testing.T) {
	items := []string{"hello", "world", "test"}
	p := &picker{
		items:    items,
		filtered: make([]int, len(items)),
		cursor:   0,
		width:    80,
		height:   24,
	}
	for i := range items {
		p.filtered[i] = i
	}
	p.render()
}

func TestPickerRunEscape(t *testing.T) {
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	go func() {
		w.Write([]byte{0x1b})
		w.Close()
	}()

	p := &picker{
		items:    []string{"hello"},
		filtered: []int{0},
		cursor:   0,
		width:    80,
		height:   24,
	}
	result := p.run()
	if result != "" {
		t.Fatalf("expected empty string for ESC, got %q", result)
	}
}

func TestPickerRunEnter(t *testing.T) {
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	go func() {
		w.Write([]byte{0x0d})
		w.Close()
	}()

	p := &picker{
		items:    []string{"hello"},
		filtered: []int{0},
		cursor:   0,
		width:    80,
		height:   24,
	}
	result := p.run()
	if result != "hello" {
		t.Fatalf("expected 'hello', got %q", result)
	}
}
