package oci

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/grimdork/climate/paths"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

const caCertPath = "etc/ssl/certs/ca-certificates.crt"

var caCertURL = "https://curl.se/ca/cacert.pem"

type Config struct {
	Binary    string
	AppDir    string
	Name      string
	CACert    string
	BaseImage string
	Arch      string
	OS        string
	SBOM      bool
}

func digestPath(path string) string {
	if len(path) < 4 || !strings.HasSuffix(path, ".tar") {
		return path + ".digest"
	}
	return path[:len(path)-4] + ".digest"
}

func Build(cfg Config) (v1.Image, error) {
	var img v1.Image
	if cfg.BaseImage != "" {
		var err error
		img, err = pullImage(cfg)
		if err != nil {
			return nil, fmt.Errorf("base image: %w", err)
		}
	} else {
		img = empty.Image
	}

	layer, err := binaryLayer(cfg.Binary, cfg.AppDir, cfg.Name)
	if err != nil {
		return nil, err
	}

	img, err = mutate.AppendLayers(img, layer)
	if err != nil {
		return nil, err
	}

	if cfg.CACert != "" {
		cl, err := certsLayer(cfg.CACert)
		if err != nil {
			return nil, fmt.Errorf("CA certs: %w", err)
		}
		img, err = mutate.AppendLayers(img, cl)
		if err != nil {
			return nil, err
		}
	}

	if cfg.SBOM {
		sbomData, err := generateSBOM(cfg.Binary, cfg.Name)
		if err != nil {
			return nil, fmt.Errorf("SBOM: %w", err)
		}
		sl, err := sbomLayer(sbomData)
		if err != nil {
			return nil, fmt.Errorf("SBOM layer: %w", err)
		}
		img, err = mutate.AppendLayers(img, sl)
		if err != nil {
			return nil, err
		}
	}

	img, err = mutate.Config(img, v1.Config{
		Entrypoint: []string{filepath.Join(cfg.AppDir, cfg.Name)},
	})
	if err != nil {
		return nil, err
	}

	img, err = mutate.Time(img, time.Time{})
	if err != nil {
		return nil, err
	}

	return img, nil
}

func pullImage(cfg Config) (v1.Image, error) {
	cacheDir, err := cacheDirectory()
	if err != nil {
		return nil, err
	}

	cacheKey := cacheKeyName(cfg.BaseImage, cfg.Arch, cfg.OS)
	cachePath := filepath.Join(cacheDir, cacheKey+".tar")

	ref, err := name.ParseReference(cfg.BaseImage)
	if err != nil {
		return nil, fmt.Errorf("invalid reference %q: %w", cfg.BaseImage, err)
	}

	auth, err := authn.DefaultKeychain.Resolve(ref.Context())
	if err != nil {
		return nil, fmt.Errorf("auth: %w", err)
	}

	var plat *v1.Platform
	if cfg.Arch != "" && cfg.OS != "" {
		plat = &v1.Platform{Architecture: cfg.Arch, OS: cfg.OS}
	}

	if img := loadFromCache(cachePath, ref, auth, plat); img != nil {
		return img, nil
	}

	opts := []remote.Option{remote.WithAuth(auth)}
	if plat != nil {
		opts = append(opts, remote.WithPlatform(*plat))
	}

	img, err := remote.Image(ref, opts...)
	if err != nil {
		return nil, fmt.Errorf("pulling %q: %w", cfg.BaseImage, err)
	}

	if err := saveToCache(img, cachePath); err != nil {
		fmt.Fprintf(os.Stderr, "  Warning: OCI cache write failed: %v ", err)
	}

	return img, nil
}

func OCICachePath() (string, error) {
	p, err := paths.New("creo")
	if err != nil {
		return "", err
	}
	return filepath.Join(p.UserBase, "oci"), nil
}

func cacheDirectory() (string, error) {
	p, err := paths.New("creo")
	if err != nil {
		return "", fmt.Errorf("paths: %w", err)
	}
	dir := filepath.Join(p.UserBase, "oci")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating cache dir %q: %w", dir, err)
	}
	return dir, nil
}

func cacheKeyName(ref, arch, os string) string {
	s := ref + "|" + arch + "|" + os
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h[:16])
}

func loadFromCache(path string, ref name.Reference, auth authn.Authenticator, plat *v1.Platform) v1.Image {
	if _, err := os.Stat(path); err != nil {
		return nil
	}

	dp := digestPath(path)
	data, err := os.ReadFile(dp)
	if err != nil {
		os.Remove(path)
		return nil
	}
	storedDigest := strings.TrimSpace(string(data))

	opts := []remote.Option{remote.WithAuth(auth)}
	if plat != nil {
		opts = append(opts, remote.WithPlatform(*plat))
	}
	desc, err := remote.Get(ref, opts...)
	if err != nil {
		return nil
	}

	if desc.Digest.String() != storedDigest {
		os.Remove(path)
		os.Remove(dp)
		return nil
	}

	img, err := tarball.ImageFromPath(path, nil)
	if err != nil {
		os.Remove(path)
		os.Remove(dp)
		return nil
	}
	return img
}

func saveToCache(img v1.Image, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	raw, err := img.RawManifest()
	if err != nil {
		return err
	}
	h := sha256.Sum256(raw)
	digest := "sha256:" + hex.EncodeToString(h[:])

	tmpPath := path + ".tmp"
	ref, err := name.NewTag("creo-cache:latest")
	if err != nil {
		return err
	}
	if err := tarball.WriteToFile(tmpPath, ref, img); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}

	if err := os.WriteFile(digestPath(path), []byte(digest+"\n"), 0644); err != nil {
		return err
	}
	return nil
}

func certsLayer(caCert string) (v1.Layer, error) {
	data, err := os.ReadFile(caCert)
	if err != nil {
		return nil, err
	}
	return layerFromBytes(caCertPath, data, 0644)
}

func FetchCACert() ([]byte, error) {
	resp, err := httpClient.Get(caCertURL)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", caCertURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading %s: %s", caCertURL, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	return data, nil
}

func WriteTarball(img v1.Image, path, tag string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	ref, err := name.NewTag(tag)
	if err != nil {
		return err
	}

	return tarball.WriteToFile(path, ref, img)
}

func binaryLayer(binary, appDir, name string) (v1.Layer, error) {
	data, err := os.ReadFile(binary)
	if err != nil {
		return nil, err
	}
	return layerFromBytes(filepath.Join(appDir, name), data, 0755)
}
