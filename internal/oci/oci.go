package oci

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

type Config struct {
	Binary string
	AppDir string
	Name   string
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

