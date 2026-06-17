package targets

import (
	"os/exec"
	"strings"

	"github.com/grimdork/climate/fx"
	"github.com/grimdork/creo/internal/fiat"
)

func gitConfigUser() string {
	out, err := exec.Command("git", "config", "--global", "user.name").Output()
	if err != nil {
		return ""
	}
	name := strings.TrimSpace(string(out))
	out, err = exec.Command("git", "config", "--global", "user.email").Output()
	if err != nil {
		return name
	}
	email := strings.TrimSpace(string(out))
	if email != "" {
		return name + " <" + email + ">"
	}
	return name
}

// InitArchive adds an archive target to the fiat file.
func InitArchive(dir string, force, verbose bool) ([]string, error) {
	file, err := ensureFiat(dir)
	if err != nil {
		return nil, err
	}
	if fiat.FindTarget(file, "archive") == nil {
		at := &fiat.Target{
			Name:     "archive",
			Language: "archive",
			Desc:     "Create release archive",
			Requires: []string{"build"},
		}
		file.AddTarget(at)
		if verbose {
			fx.Println("  {success}Added archive target{@}")
		}
	} else if verbose {
		fx.Println("  {warning}Skipped archive target (already exists){@}")
	}
	if err := file.Write(); err != nil {
		return nil, err
	}
	return nil, nil
}

// InitDeb adds a deb packaging target to the fiat file.
func InitDeb(dir string, force, verbose bool) ([]string, error) {
	file, err := ensureFiat(dir)
	if err != nil {
		return nil, err
	}
	if fiat.FindTarget(file, "deb") == nil {
		maint := gitConfigUser()
		if maint == "" {
			maint = DefMaintainer
		}
		dt := &fiat.Target{
			Name:     "deb",
			Language: "deb",
			Desc:     "Create .deb package",
			Requires: []string{"build"},
		}
		dt.Vars = append(dt.Vars, &fiat.Var{Name: "maintainer", Value: maint})
		file.AddTarget(dt)
		if verbose {
			fx.Println("  {success}Added deb target{@}")
		}
	} else if verbose {
		fx.Println("  {warning}Skipped deb target (already exists){@}")
	}
	if err := file.Write(); err != nil {
		return nil, err
	}
	return nil, nil
}

// InitRpm adds an RPM packaging target to the fiat file.
func InitRpm(dir string, force, verbose bool) ([]string, error) {
	file, err := ensureFiat(dir)
	if err != nil {
		return nil, err
	}
	if fiat.FindTarget(file, "rpm") == nil {
		maint := gitConfigUser()
		if maint == "" {
			maint = DefMaintainer
		}
		rt := &fiat.Target{
			Name:     "rpm",
			Language: "rpm",
			Desc:     "Create .rpm package",
			Requires: []string{"build"},
		}
		rt.Vars = append(rt.Vars, &fiat.Var{Name: "maintainer", Value: maint})
		file.AddTarget(rt)
		if verbose {
			fx.Println("  {success}Added rpm target{@}")
		}
	} else if verbose {
		fx.Println("  {warning}Skipped rpm target (already exists){@}")
	}
	if err := file.Write(); err != nil {
		return nil, err
	}
	return nil, nil
}

// InitBrew adds a Homebrew formula target to the fiat file.
func InitBrew(dir string, force, verbose bool) ([]string, error) {
	file, err := ensureFiat(dir)
	if err != nil {
		return nil, err
	}
	if fiat.FindTarget(file, "brew") == nil {
		bt := &fiat.Target{
			Name:     "brew",
			Language: "brew",
			Desc:     "Create Homebrew formula",
			Requires: []string{"archive"},
		}
		bt.Vars = append(bt.Vars, &fiat.Var{Name: "tap", Value: "user/homebrew-tools"})
		file.AddTarget(bt)
		if verbose {
			fx.Println("  {success}Added brew target{@}")
		}
	} else if verbose {
		fx.Println("  {warning}Skipped brew target (already exists){@}")
	}
	if err := file.Write(); err != nil {
		return nil, err
	}
	return nil, nil
}
