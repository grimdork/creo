package targets

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/grimdork/climate/fx"
	"github.com/grimdork/creo/internal/fiat"
	"github.com/grimdork/creo/internal/semver"
	"github.com/grimdork/creo/internal/templates"
	"github.com/grimdork/creo/internal/util"
)

func absDirName(dir string) (string, string) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}
	return absDir, filepath.Base(absDir)
}

func ensureFiat(dir string) (*fiat.File, error) {
	fiatPath := filepath.Join(dir, "fiat")
	if _, err := os.Stat(fiatPath); err == nil {
		file, err := fiat.Parse(fiatPath)
		if err != nil {
			return nil, err
		}
		return file, nil
	}
	file := fiat.NewFile(fiatPath)
	return file, nil
}

func useBasicTemplate(langName string) bool {
	_, err := templates.ResolveTemplate(langName, "basic")
	return err == nil
}

// InitProject dispatches to the correct Init* function based on the requested language list.
// When a "basic" template exists for a language, it delegates to InitProjectWithTemplate
// instead of the hardcoded Init* function.
func InitProject(langs []string, extraVars map[string]string, force, verbose bool) error {
	if force {
		if _, err := os.Stat(".creo"); err == nil {
			if err := os.RemoveAll(".creo"); err != nil {
				return fmt.Errorf(errRemovingCreo, err)
			}
			if verbose {
				fx.Println("  {muted}Removed .creo/{@}")
			}
		}
	}

	if len(langs) == 0 {
		return fiat.WriteDefaultFile("fiat", force, verbose)
	}

	var allIgnores []string

	for _, spec := range langs {
		langName, ver := spec, ""
		if idx := strings.IndexByte(spec, ':'); idx >= 0 {
			langName, ver = spec[:idx], spec[idx+1:]
		}

		var ignores []string
		var err error

		switch langName {
		case LangGo:
			if ver == "" && useBasicTemplate(langName) {
				err = InitProjectWithTemplate(langName, "basic", extraVars, force, verbose)
			} else {
				ignores, err = Init(".", ver, force, verbose)
			}
		case LangTinyGo:
			if useBasicTemplate(langName) {
				err = InitProjectWithTemplate(langName, "basic", extraVars, force, verbose)
			} else {
				ignores, err = InitTinyGo(".", force, verbose)
			}
		case LangC:
			if useBasicTemplate(langName) {
				err = InitProjectWithTemplate(langName, "basic", extraVars, force, verbose)
			} else {
				ignores, err = InitC(".", force, verbose)
			}
		case LangCxx, LangCpp:
			tmplLang := langName
			if langName == LangCpp {
				tmplLang = LangCxx
			}
			if useBasicTemplate(tmplLang) {
				err = InitProjectWithTemplate(tmplLang, "basic", extraVars, force, verbose)
			} else {
				ignores, err = InitCxx(".", force, verbose)
			}
		case LangRust:
			if useBasicTemplate(langName) {
				err = InitProjectWithTemplate(langName, "basic", extraVars, force, verbose)
			} else {
				ignores, err = InitRust(".", force, verbose)
			}
		case LangPython:
			if useBasicTemplate(langName) {
				err = InitProjectWithTemplate(langName, "basic", extraVars, force, verbose)
			} else {
				ignores, err = InitPython(".", force, verbose)
			}
		case LangNode, LangTS:
			if useBasicTemplate("node") {
				err = InitProjectWithTemplate("node", "basic", extraVars, force, verbose)
			} else {
				ignores, err = InitNode(".", force, verbose)
			}
		case LangJava, LangKotlin, LangGradle:
			if useBasicTemplate("java") {
				err = InitProjectWithTemplate("java", "basic", extraVars, force, verbose)
			} else {
				ignores, err = InitJava(".", force, verbose)
			}
		case LangOCI:
			ignores, err = InitOci(".", force, verbose)
		case LangArchive:
			ignores, err = InitArchive(".", force, verbose)
		case LangDeb:
			ignores, err = InitDeb(".", force, verbose)
		case LangRpm:
			ignores, err = InitRpm(".", force, verbose)
		case LangBrew:
			ignores, err = InitBrew(".", force, verbose)
		default:
			return fmt.Errorf("unknown language: %s", langName)
		}
		if err != nil {
			return err
		}
		if ignores != nil {
			allIgnores = append(allIgnores, ignores...)
		}
	}

	if len(allIgnores) > 0 {
		if err := WriteIgnores(allIgnores, verbose); err != nil {
			return err
		}
	}
	return nil
}

func InitProjectWithTemplate(lang, tmplName string, extraVars map[string]string, force, verbose bool) error {
	tmpl, err := templates.ResolveTemplate(lang, tmplName)
	if err != nil {
		return err
	}

	if tmpl.Language != lang {
		return fmt.Errorf("template %q targets language %q, not %q", tmplName, tmpl.Language, lang)
	}

	if extraVars == nil {
		extraVars = make(map[string]string)
	}
	if _, ok := extraVars["PROJECT"]; !ok {
		extraVars["PROJECT"] = dirProjectName()
	}
	if _, ok := extraVars["VERSION"]; !ok {
		extraVars["VERSION"] = versionFromGit()
	}
	if err := templates.ApplyTemplate(tmpl, ".", extraVars, force, verbose); err != nil {
		return err
	}

	switch lang {
	case LangGo, LangTinyGo:
		if err := initGoMod(".", dirProjectName(), force, verbose); err != nil {
			return err
		}
		if err := runGoModTidy("."); err != nil {
			fx.Fprint(os.Stderr, "{warning}go mod tidy: {}{@}\n", err)
		}
	case LangRust:
		// Cargo.toml.tmpl handles project metadata
		// No cargo init needed — template provides all files
	case LangC, LangCxx:
		// Template provides source files, no project init needed
	}
	gitignore := []string{"/build", "/.creo", "/tmp"}
	switch lang {
	case LangC, LangCxx:
		// Default gitignore (build, creo, tmp) is correct for C/C++
	case LangRust:
		gitignore = append(gitignore, "/target")
	case LangPython:
		gitignore = append(gitignore, "__pycache__/", "*.pyc", "dist/", "*.egg-info/", ".venv/")
	case LangNode, LangTS:
		gitignore = append(gitignore, "node_modules/", "dist/")
	case LangJava, LangKotlin, LangGradle:
		gitignore = append(gitignore, "build/", ".gradle/", "*.jar")
	}
	return WriteIgnores(gitignore, verbose)
}

// WriteIgnores writes or appends unique gitignore lines to .gitignore.
func WriteIgnores(lines []string, verbose bool) error {
	lines = util.Unique(lines)
	if _, err := os.Stat(".gitignore"); err == nil {
		data, err := os.ReadFile(".gitignore")
		if err != nil {
			return fmt.Errorf("reading .gitignore: %w", err)
		}
		content := string(data)
		existing := strings.Split(content, "\n")
		added := false
		for _, line := range lines {
			found := false
			for _, el := range existing {
				if strings.TrimSpace(el) == line {
					found = true
					break
				}
			}
			if !found {
				f, err := os.OpenFile(".gitignore", os.O_APPEND|os.O_WRONLY, 0644)
				if err != nil {
					return fmt.Errorf("appending to .gitignore: %w", err)
				}
				if _, err := f.WriteString(line + "\n"); err != nil {
					f.Close()
					return fmt.Errorf("writing .gitignore: %w", err)
				}
				f.Close()
				added = true
			}
		}
		if added && verbose {
			fx.Println("  {success}Updated .gitignore{@}")
		} else if verbose {
			fx.Println("  {warning}Skipped .gitignore{@}")
		}
	} else {
		content := strings.Join(lines, "\n") + "\n"
		if err := os.WriteFile(".gitignore", []byte(content), 0644); err != nil {
			return fmt.Errorf("creating .gitignore: %w", err)
		}
		if verbose {
			fx.Println("  {success}Created .gitignore{@}")
		}
	}
	return nil
}

func dirProjectName() string {
	_, name := absDirName(".")
	return name
}

func versionFromGit() string {
	return semver.String()
}
