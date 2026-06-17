package cli

import (
	"github.com/grimdork/climate/fx"
	"github.com/grimdork/creo/internal/templates"
)

// RunListTemplates prints available project templates, optionally filtered by language.
func RunListTemplates(lang string) error {
	list, err := templates.ListTemplates(lang)
	if err != nil {
		return err
	}
	if len(list) == 0 {
		fx.Println("{warning}No templates found{@}")
		return nil
	}
	fx.Println("{bold}Available templates:{@}")
	for _, t := range list {
		fx.Println("  {cyan}{}/{}  {}{@}", t.Language, t.Name, t.Description)
	}
	return nil
}

// RunSaveTemplate extracts an embedded template to the user's template directory.
func RunSaveTemplate(spec string, force, verbose bool) error {
	return templates.SaveTemplate(spec, force, verbose)
}
