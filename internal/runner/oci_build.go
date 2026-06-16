package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/grimdork/climate/fx"
	"github.com/grimdork/creo/internal/fiat"
	"github.com/grimdork/creo/internal/oci"
)

func handleOCI(f *fiat.File, t *fiat.Target, c combo, comboVars map[string]*fiat.Var, comboEnv []string, dir string, activeArch, activeOS string, opts RunOpts, name string, outputs *Outputs, errCh chan<- error) {
	binSrc := ""
	for _, dep := range t.Requires {
		outVar := "OUTPUT_" + dep
		if v, ok := comboVars[outVar]; ok && v.Value != "" {
			binSrc = v.Value
			break
		}
	}
	if binSrc == "" {
		binSrc = c.bin
	}

	if binSrc != "" {
		absBin, err := filepath.Abs(binSrc)
		if err != nil {
			errCh <- fmt.Errorf("%s: resolving binary path: %w", f.Path(), err)
			return
		}
		if _, err := os.Stat(absBin); err != nil {
			errCh <- fmt.Errorf("%s: OCI binary %q not found: %w", f.Path(), absBin, err)
			return
		}
		binSrc = absBin
		binaryName := filepath.Base(binSrc)
		appDir := fiat.Expand(t.OCI.AppDir, comboVars, 0)
		if appDir == "" {
			appDir = "/app"
		}

		caCert := fiat.Expand(t.OCI.CACert, comboVars, 0)
		if caCert == "auto" {
			cacheDir := filepath.Join(filepath.Dir(f.Path()), ".creo")
			cachePath := filepath.Join(cacheDir, "cacert.pem")

			if opts.RefreshCACerts {
				if err := os.Remove(cachePath); err != nil && opts.Verbose {
					fx.Fprint(os.Stderr, "  {warning}removing stale CA cert: {}{@}\n", err)
				} else if opts.Verbose {
					fx.Println(`  {cyan}Refreshed cached CA certs{@}`)
				}
			}

			if _, err := os.Stat(cachePath); os.IsNotExist(err) {
				if err := os.MkdirAll(cacheDir, 0755); err != nil {
					errCh <- fmt.Errorf("%s: creating cache dir: %w", f.Path(), err)
					return
				}
				data, err := oci.FetchCACert()
				if err != nil {
					errCh <- fmt.Errorf("%s: %w", f.Path(), err)
					return
				}
				tmpPath := cachePath + ".tmp"
				if err := os.WriteFile(tmpPath, data, 0644); err != nil {
					errCh <- fmt.Errorf("%s: writing CA cert cache: %w", f.Path(), err)
					return
				}
				if err := os.Rename(tmpPath, cachePath); err != nil {
					os.Remove(tmpPath)
					errCh <- fmt.Errorf("%s: renaming CA cert cache: %w", f.Path(), err)
					return
				}
				if opts.Verbose {
					fx.Println(`  {cyan}Downloaded CA certs to .creo/cacert.pem{@}`)
				}
			}
			caCert = cachePath
		}

		entrypoint := strings.Fields(fiat.Expand(t.OCI.Entrypoint, comboVars, 0))

		var extraFiles []oci.ExtraFile
		for _, ef := range t.OCI.Files {
			src := fiat.Expand(ef.Src, comboVars, 0)
			dst := fiat.Expand(ef.Dst, comboVars, 0)
			extraFiles = append(extraFiles, oci.ExtraFile{Src: src, Dst: dst})
		}
		for _, df := range t.OCI.Downloads {
			src := fiat.Expand(df.Src, comboVars, 0)
			dst := fiat.Expand(df.Dst, comboVars, 0)
			extraFiles = append(extraFiles, oci.ExtraFile{Src: src, Dst: dst, IsURL: true})
		}

		img, err := oci.Build(oci.Config{
			Binary:     binSrc,
			AppDir:     appDir,
			Name:       binaryName,
			CACert:     caCert,
			BaseImage:  fiat.Expand(t.OCI.BaseImage, comboVars, 0),
			Arch:       activeArch,
			OS:         "linux",
			SBOM:       t.OCI.SBOM,
			Entrypoint: entrypoint,
			Files:      extraFiles,
		})
		if err != nil {
			errCh <- fmt.Errorf("%s: OCI build: %w", f.Path(), err)
			return
		}

		tarballPath := fiat.Expand(t.OCI.Tarball, comboVars, 0)
		if tarballPath != "" {
			tag := fiat.Expand(t.OCI.Tag, comboVars, 0)
			if tag == "" {
				tag = "latest"
			}
			if err := oci.WriteTarball(img, tarballPath, tag); err != nil {
				errCh <- fmt.Errorf("%s: OCI tarball: %w", f.Path(), err)
				return
			}
			fx.Println(`  {success}Wrote {}{@}`, tarballPath)
		}

		repo := fiat.Expand(t.OCI.Repo, comboVars, 0)
		if repo != "" {
			tag := fiat.Expand(t.OCI.Tag, comboVars, 0)
			user := os.ExpandEnv(fiat.Expand(t.OCI.User, comboVars, 0))
			pass := os.ExpandEnv(fiat.Expand(t.OCI.Pass, comboVars, 0))
			if pass == "" && t.OCI.CredHelper != "" {
				helper := os.ExpandEnv(fiat.Expand(t.OCI.CredHelper, comboVars, 0))
				hUser, hPass, helpErr := execCredHelper(helper, dir)
				if helpErr != nil {
					errCh <- fmt.Errorf("%s: credential helper: %w", f.Path(), helpErr)
					return
				}
				if user == "" {
					user = hUser
				}
				pass = hPass
			}
			if err := oci.Push(img, oci.PushConfig{
				Repo: repo,
				Tag:  tag,
				User: user,
				Pass: pass,
			}); err != nil {
				errCh <- fmt.Errorf("%s: OCI push: %w", f.Path(), err)
				return
			}
			ref := repo
			if tag != "" {
				ref += ":" + tag
			}
			fx.Println(`  {success}Pushed {}{@}`, ref)
		}
	}
}
