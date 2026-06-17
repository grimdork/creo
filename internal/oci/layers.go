package oci

import (
	"archive/tar"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

func certsLayer(caCert string) (v1.Layer, error) {
	data, err := os.ReadFile(caCert)
	if err != nil {
		return nil, err
	}
	return layerFromBytes(caCertPath, data, 0644)
}

func binaryLayer(binary, appDir, name string) (v1.Layer, error) {
	data, err := os.ReadFile(binary)
	if err != nil {
		return nil, err
	}
	return layerFromBytes(filepath.Join(appDir, name), data, 0755)
}

func directoryLayer(srcDir, appDir string) (v1.Layer, error) {
	if appDir == "" {
		appDir = "/app"
	}

	tmp, err := os.CreateTemp("", "creo-layer-*.tar")
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()

	tw := tar.NewWriter(tmp)

	walkErr := filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(appDir, rel)

		fi, err := d.Info()
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return err
		}
		header.Name = target
		header.ModTime = time.Time{}

		if fi.IsDir() {
			header.Name += "/"
		}

		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if !fi.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if _, err := tw.Write(data); err != nil {
				return err
			}
		}
		return nil
	})
	tw.Close()
	tmp.Close()
	if walkErr != nil {
		os.Remove(tmpPath)
		return nil, fmt.Errorf("taring %q: %w", srcDir, walkErr)
	}

	return tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return os.Open(tmpPath)
	})
}

func extraFileLayer(src, dst string, isURL bool) (v1.Layer, error) {
	var data []byte
	var mode int64 = 0644

	if isURL {
		resp, err := httpClient.Get(src)
		if err != nil {
			return nil, fmt.Errorf("downloading %s: %w", src, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("downloading %s: %s", src, resp.Status)
		}
		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", src, err)
		}
	} else {
		fi, err := os.Stat(src)
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", src, err)
		}
		if fi.Mode()&0111 != 0 {
			mode = 0755
		}
		data, err = os.ReadFile(src)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", src, err)
		}
	}

	return layerFromBytes(dst, data, mode)
}
