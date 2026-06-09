package lang

import (
	"github.com/grimdork/creo/internal/fiat"
)

func applyOci(f *fiat.File, t *fiat.Target) {
	cfg := &fiat.OCIConfig{
		AppDir: "/app",
		Tag:    "latest",
	}

	for _, v := range t.Vars {
		switch v.Name {
		case "tarball":
			cfg.Tarball = v.Value
		case "repo":
			cfg.Repo = v.Value
		case "tag":
			cfg.Tag = v.Value
		case "appdir":
			cfg.AppDir = v.Value
		case "ociuser":
			cfg.User = v.Value
		case "ocipass":
			cfg.Pass = v.Value
		case "ocicred":
			cfg.CredHelper = v.Value
		case "cacert":
			cfg.CACert = v.Value
		case "from":
			cfg.BaseImage = v.Value
		case "sbom":
			cfg.SBOM = v.Value == "true" || v.Value == "1"
		}
	}

	if cfg.Tarball == "" && cfg.Repo == "" {
		cfg.Tarball = "build/" + t.Name + ".tar"
	}

	t.OCI = cfg
	if cfg.Tarball != "" {
		t.Bin = cfg.Tarball
	}
}
