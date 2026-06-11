package lang

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
		case "region":
			cfg.Region = v.Value
		case "cacert":
			cfg.CACert = v.Value
		case "from":
			cfg.BaseImage = v.Value
		case "sbom":
			cfg.SBOM = v.Value == "true" || v.Value == "1"
		case "entrypoint":
			cfg.Entrypoint = v.Value
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
				r = "us-east-1"
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
			region = "us-east-1"
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
		return "fr-par"
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
