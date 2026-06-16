package templates

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/grimdork/climate/fx"
	"github.com/grimdork/climate/paths"
	"github.com/grimdork/creo/internal/fiat"
)

type Template struct {
	Name        string
	Description string
	Language    string
	Files       []string
	Vars        map[string]string
	Dir         string
}

func userTemplateDir() (string, error) {
	p, err := paths.New("creo")
	if err != nil {
		return "", err
	}
	return filepath.Join(p.UserBase, "templates"), nil
}

func ResolveTemplate(lang, name string) (*Template, error) {
	ud, err := userTemplateDir()
	if err != nil {
		return nil, err
	}

	userPath := filepath.Join(ud, lang, name)
	if fi, err := os.Stat(userPath); err == nil && fi.IsDir() {
		return loadTemplate(userPath)
	}

	embedPath := filepath.Join("embedded", lang, name)
	if _, err := fs.Stat(embeddedTemplates, embedPath); err == nil {
		return loadEmbeddedTemplate(embedPath)
	}

	return nil, fmt.Errorf("template %q not found for language %q", name, lang)
}

func ListTemplates(lang string) ([]Template, error) {
	var list []Template
	seen := map[string]bool{}

	seenKey := func(t *Template) string {
		return t.Language + "/" + t.Name
	}

	addFromDir := func(root string) error {
		entries, err := os.ReadDir(root)
		if err != nil {
			return nil
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			t, err := loadTemplate(filepath.Join(root, e.Name()))
			if err != nil {
				continue
			}
			if seen[seenKey(t)] {
				continue
			}
			seen[seenKey(t)] = true
			list = append(list, *t)
		}
		return nil
	}

	addFromEmbed := func(prefix string) {
		entries, err := fs.ReadDir(embeddedTemplates, prefix)
		if err != nil {
			return
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			t, err := loadEmbeddedTemplate(filepath.Join(prefix, e.Name()))
			if err != nil {
				continue
			}
			if seen[seenKey(t)] {
				continue
			}
			seen[seenKey(t)] = true
			list = append(list, *t)
		}
	}

	ud, err := userTemplateDir()
	if err == nil {
		if lang == "" {
			langs, rerr := os.ReadDir(ud)
			if rerr == nil {
				for _, l := range langs {
					if l.IsDir() {
						addFromDir(filepath.Join(ud, l.Name()))
					}
				}
			}
		} else {
			addFromDir(filepath.Join(ud, lang))
		}
	}

	if lang == "" {
		langs, rerr := fs.ReadDir(embeddedTemplates, "embedded")
		if rerr == nil {
			for _, l := range langs {
				if l.IsDir() {
					addFromEmbed(filepath.Join("embedded", l.Name()))
				}
			}
		}
	} else {
		addFromEmbed(filepath.Join("embedded", lang))
	}

	return list, nil
}

func SaveTemplate(spec string, force, verbose bool) error {
	idx := strings.IndexByte(spec, '/')
	if idx < 0 {
		return fmt.Errorf("expected lang/name format, got %q", spec)
	}
	lang, name := spec[:idx], spec[idx+1:]
	if lang == "" || name == "" {
		return fmt.Errorf("expected lang/name format, got %q", spec)
	}

	embedPrefix := filepath.Join("embedded", lang, name)
	if _, err := fs.Stat(embeddedTemplates, embedPrefix); err != nil {
		return fmt.Errorf("template %q not found", spec)
	}

	ud, err := userTemplateDir()
	if err != nil {
		return err
	}
	destDir := filepath.Join(ud, lang, name)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("creating %s: %w", destDir, err)
	}

	t, err := loadEmbeddedTemplate(embedPrefix)
	if err != nil {
		return fmt.Errorf("loading template %q: %w", spec, err)
	}

	toCopy := []string{"template.ini"}
	toCopy = append(toCopy, t.Files...)

	entries, err := fs.ReadDir(embeddedTemplates, embedPrefix)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			entryName := e.Name()
			if entryName == "template.ini" {
				continue
			}
			isPlatform := false
			for _, f := range t.Files {
				base := strings.TrimSuffix(f, ".tmpl")
				if strings.HasPrefix(entryName, base+".") && strings.HasSuffix(entryName, ".tmpl") {
					isPlatform = true
					break
				}
			}
			if isPlatform {
				toCopy = append(toCopy, entryName)
			}
		}
	}

	seen := map[string]bool{}
	for _, file := range toCopy {
		if seen[file] {
			continue
		}
		seen[file] = true
		srcPath := filepath.Join(embedPrefix, file)
		dstPath := filepath.Join(destDir, file)

		if _, err := os.Stat(dstPath); err == nil {
			if !force {
				if verbose {
					fx.Println("  {warning}Skipped {} (already exists){@}", dstPath)
				}
				continue
			}
		}

		data, err := fs.ReadFile(embeddedTemplates, srcPath)
		if err != nil {
			return fmt.Errorf("reading %s: %w", srcPath, err)
		}
		if err := os.WriteFile(dstPath, data, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", dstPath, err)
		}
		if verbose {
			fx.Println("  {success}Created {}{@}", dstPath)
		}
	}
	return nil
}

func ApplyTemplate(t *Template, destDir string, extraVars map[string]string, force, verbose bool) error {
	vars := make(map[string]string)
	for k, v := range t.Vars {
		vars[k] = v
	}
	for k, v := range extraVars {
		vars[k] = v
	}

	fiatVars := make(map[string]*fiat.Var)
	for k, v := range vars {
		fiatVars[k] = &fiat.Var{Name: k, Value: v}
	}

	for _, file := range t.Files {
		srcName := file
		dstName := file
		shouldExpand := false
		if strings.HasSuffix(file, ".tmpl") {
			dstName = strings.TrimSuffix(file, ".tmpl")
			shouldExpand = true
			// Check for platform variant: file.darwin.tmpl, file.linux.tmpl
			platformFile := dstName + "." + runtime.GOOS + ".tmpl"
			platformPath := filepath.Join(t.Dir, platformFile)
			_, err := os.Stat(platformPath)
			if os.IsNotExist(err) {
				_, err = fs.Stat(embeddedTemplates, platformPath)
			}
			if err == nil {
				srcName = platformFile
			}
		}

		srcPath := filepath.Join(t.Dir, srcName)
		dstPath := filepath.Join(destDir, dstName)

		if _, err := os.Stat(dstPath); err == nil && !force {
			if verbose {
				fx.Println("  {warning}Skipped {} (already exists){@}", dstName)
			}
			continue
		}

		data, err := os.ReadFile(srcPath)
		if err != nil {
			srcPath = filepath.Join(t.Dir, srcName)
			edata, err2 := fs.ReadFile(embeddedTemplates, srcPath)
			if err2 != nil {
				return fmt.Errorf("reading template file %s: %w", srcName, err)
			}
			data = edata
		}

		if shouldExpand {
			expanded := fiat.Expand(string(data), fiatVars, 0)
			data = []byte(expanded)
		}

		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", dstName, err)
		}
		if err := os.WriteFile(dstPath, data, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", dstName, err)
		}
		if verbose {
			fx.Println("  {success}Created {}{@}", dstName)
		}
	}
	return nil
}

func loadTemplate(dir string) (*Template, error) {
	iniPath := filepath.Join(dir, "template.ini")
	data, err := os.ReadFile(iniPath)
	if err != nil {
		return nil, fmt.Errorf("reading template.ini: %w", err)
	}
	t, err := parseTemplateINI(string(data), dir)
	if err != nil {
		return nil, err
	}
	for _, f := range t.Files {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			return nil, fmt.Errorf("template file %q listed in template.ini not found", f)
		}
	}
	return t, nil
}

func loadEmbeddedTemplate(prefix string) (*Template, error) {
	iniPath := filepath.Join(prefix, "template.ini")
	data, err := fs.ReadFile(embeddedTemplates, iniPath)
	if err != nil {
		return nil, fmt.Errorf("reading embedded template.ini: %w", err)
	}
	t, err := parseTemplateINI(string(data), prefix)
	if err != nil {
		return nil, err
	}
	for _, f := range t.Files {
		if _, err := fs.Stat(embeddedTemplates, filepath.Join(prefix, f)); err != nil {
			return nil, fmt.Errorf("template file %q listed in template.ini not found", f)
		}
	}
	return t, nil
}

type iniSection struct {
	lines []string
}

func parseTemplateINI(data, dir string) (*Template, error) {
	t := &Template{
		Vars: make(map[string]string),
		Dir:  dir,
	}

	var section string
	for _, raw := range strings.Split(data, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(line[1 : len(line)-1])
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		switch section {
		case "template":
			switch key {
			case "name":
				t.Name = value
			case "description":
				t.Description = value
			case "language":
				t.Language = value
			case "files":
				for _, f := range strings.Split(value, ",") {
					f = strings.TrimSpace(f)
					if f != "" {
						t.Files = append(t.Files, f)
					}
				}
			}
		case "vars":
			t.Vars[key] = value
		}
	}

	if t.Name == "" {
		return nil, fmt.Errorf("template.ini missing 'name' in [template]")
	}
	if t.Language == "" {
		return nil, fmt.Errorf("template.ini missing 'language' in [template]")
	}
	return t, nil
}
