package targets

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/grimdork/creo/internal/fiat"
)

func applyOci(f *fiat.File, t *fiat.Target) {
	m := lookupManifest(f, t)

	cfg := &fiat.OCIConfig{
		AppDir: DefAppDir,
		Tag:    "latest",
	}

	if len(m.Files) > 0 {
		cfg.Files = m.Files
	}
	if len(m.Downloads) > 0 {
		cfg.Downloads = m.Downloads
	}

	for _, v := range t.Vars {
		val := fiat.Expand(v.Value, f.Vars, 0)
		switch v.Name {
		case "tarball":
			cfg.Tarball = val
		case "repo":
			cfg.Repo = val
		case "tag":
			cfg.Tag = val
		case "appdir":
			cfg.AppDir = val
		case "ociuser":
			cfg.User = val
		case "ocipass":
			cfg.Pass = val
		case "ocicred":
			cfg.CredHelper = val
		case "region":
			cfg.Region = val
		case "cacert":
			cfg.CACert = val
		case "from":
			cfg.BaseImage = val
		case "sbom":
			cfg.SBOM = v.Value == "true" || v.Value == "1"
		case "entrypoint":
			cfg.Entrypoint = val
		default:
			// Unknown properties are silently ignored —
			// they may be consumed by other target plumbing.
		}
	}

	if t.LangAlias != "" {
		applyRegistryAlias(f, t, cfg)
	}

	if cfg.Tarball == "" && cfg.Repo == "" {
		bd := BuildDir(f)
		cfg.Tarball = bd + "/" + t.Name + ".tar"
	}

	t.OCI = cfg
	if cfg.Tarball != "" {
		t.Bin = cfg.Tarball
	}
}

func applyRegistryAlias(f *fiat.File, t *fiat.Target, cfg *fiat.OCIConfig) {
	if cfg.Repo == "" {
		owner := resolveOwner(f, t)
		cfg.Repo = aliasRepo(t.LangAlias, owner, cfg.Region, t.Name)
	}
	switch t.LangAlias {
	case "ecr":
		if cfg.User == "" {
			cfg.User = "AWS"
		}
		if cfg.CredHelper == "" {
			r := cfg.Region
			if r == "" {
				r = resolveRegion(f, t)
			}
			if r == "" {
				r = DefECRRegion
			}
			cfg.CredHelper = "aws ecr get-login-password --region " + r
		}
	}
}

func aliasRepo(alias, owner, region, name string) string {
	switch alias {
	case "ghcr":
		return "ghcr.io/" + owner + "/" + name
	case "docker", "dockerhub":
		return "docker.io/" + owner + "/" + name
	case "ecr":
		if region == "" {
			region = DefECRRegion
		}
		return owner + ".dkr.ecr." + region + ".amazonaws.com/" + name
	case "gcr":
		return "gcr.io/" + owner + "/" + name
	case "acr":
		return owner + ".azurecr.io/" + name
	case "scw":
		return "rg." + scwRegion(region) + ".scw.cloud/" + owner + "/" + name
	default:
		return ""
	}
}

func scwRegion(region string) string {
	if region == "" {
		return DefScwRegion
	}
	switch region {
	case "fr", "fr-par":
		return "fr-par"
	case "nl", "nl-ams":
		return "nl-ams"
	case "pl", "pl-waw":
		return "pl-waw"
	case "it", "it-mil":
		return "it-mil"
	default:
		return region
	}
}

func resolveOwner(f *fiat.File, t *fiat.Target) string {
	for _, v := range t.Vars {
		if v.Name == "OWNER" {
			return v.Value
		}
	}
	if v, ok := f.Vars["OWNER"]; ok {
		return v.Value
	}
	if s := os.Getenv("CREO_OWNER"); s != "" {
		return s
	}
	if t.LangAlias == "ghcr" {
		if owner := ownerFromGit(); owner != "" {
			return owner
		}
	}
	dir := filepath.Dir(f.Path())
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return filepath.Base(dir)
	}
	return filepath.Base(absDir)
}

func ownerFromGit() string {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}
	s := strings.TrimSpace(string(out))
	s = strings.TrimSuffix(s, ".git")

	if strings.Contains(s, "://") {
		parts := strings.SplitN(s, "/", 4)
		if len(parts) >= 4 {
			ownerAndRepo := parts[3]
			if idx := strings.IndexByte(ownerAndRepo, '/'); idx >= 0 {
				return ownerAndRepo[:idx]
			}
			return ownerAndRepo
		}
		return ""
	}

	if idx := strings.LastIndexByte(s, ':'); idx >= 0 {
		s = s[idx+1:]
	}
	if idx := strings.IndexByte(s, '/'); idx >= 0 {
		return s[:idx]
	}
	return ""
}

func resolveRegion(f *fiat.File, t *fiat.Target) string {
	if v, ok := f.Vars["REGION"]; ok {
		return v.Value
	}
	return os.Getenv("CREO_REGION")
}
