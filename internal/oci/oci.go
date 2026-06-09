package oci

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

const caCertPath = "etc/ssl/certs/ca-certificates.crt"

var caCertURL = "https://curl.se/ca/cacert.pem"

type Config struct {
	Binary string
	AppDir string
	Name   string
	CACert string
}

func Build(cfg Config) (v1.Image, error) {
	layer, err := binaryLayer(cfg.Binary, cfg.AppDir, cfg.Name)
	if err != nil {
		return nil, err
	}

	img, err := mutate.AppendLayers(empty.Image, layer)
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

func certsLayer(caCert string) (v1.Layer, error) {
	data, err := os.ReadFile(caCert)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	if err := tw.WriteHeader(&tar.Header{
		Name:     caCertPath,
		Size:     int64(len(data)),
		Mode:     0644,
		ModTime:  time.Time{},
		Typeflag: tar.TypeReg,
	}); err != nil {
		return nil, err
	}
	if _, err := tw.Write(data); err != nil {
		return nil, err
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	return tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(buf.Bytes())), nil
	})
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

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	if err := tw.WriteHeader(&tar.Header{
		Name:     filepath.Join(appDir, name),
		Size:     int64(len(data)),
		Mode:     0755,
		ModTime:  time.Time{},
		Typeflag: tar.TypeReg,
	}); err != nil {
		return nil, err
	}
	if _, err := tw.Write(data); err != nil {
		return nil, err
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	return tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(buf.Bytes())), nil
	})
}

