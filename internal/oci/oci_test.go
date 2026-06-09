package oci

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

func TestBinaryLayer(t *testing.T) {
	t.Helper()

	tmp := t.TempDir()
	binaryPath := filepath.Join(tmp, "myapp")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\necho hello"), 0755); err != nil {
		t.Fatal(err)
	}

	layer, err := binaryLayer(binaryPath, "/app", "myapp")
	if err != nil {
		t.Fatalf("binaryLayer with valid file returned error: %v", err)
	}
	if layer == nil {
		t.Fatal("binaryLayer returned nil layer")
	}
}

func TestBinaryLayer_NonExistent(t *testing.T) {
	t.Helper()

	_, err := binaryLayer("/nonexistent/binary", "/app", "myapp")
	if err == nil {
		t.Fatal("binaryLayer with non-existent file expected error, got nil")
	}
}

func TestCertsLayer(t *testing.T) {
	t.Helper()

	tmp := t.TempDir()
	certPath := filepath.Join(tmp, "ca.pem")
	certContent := []byte("-----BEGIN CERTIFICATE-----\nFAKE\n-----END CERTIFICATE-----")
	if err := os.WriteFile(certPath, certContent, 0644); err != nil {
		t.Fatal(err)
	}

	layer, err := certsLayer(certPath)
	if err != nil {
		t.Fatalf("certsLayer with valid file returned error: %v", err)
	}
	if layer == nil {
		t.Fatal("certsLayer returned nil layer")
	}
}

func TestCertsLayer_NonExistent(t *testing.T) {
	t.Helper()

	_, err := certsLayer("/nonexistent/ca.pem")
	if err == nil {
		t.Fatal("certsLayer with non-existent path expected error, got nil")
	}
}

func TestBuild(t *testing.T) {
	t.Helper()

	tmp := t.TempDir()
	binaryPath := filepath.Join(tmp, "myapp")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\necho hello"), 0755); err != nil {
		t.Fatal(err)
	}

	img, err := Build(Config{
		Binary: binaryPath,
		AppDir: "/app",
		Name:   "myapp",
	})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if img == nil {
		t.Fatal("Build returned nil image")
	}
}

func TestBuildWithCACert(t *testing.T) {
	t.Helper()

	tmp := t.TempDir()
	binaryPath := filepath.Join(tmp, "myapp")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\necho hello"), 0755); err != nil {
		t.Fatal(err)
	}

	certPath := filepath.Join(tmp, "ca.pem")
	certContent := []byte("-----BEGIN CERTIFICATE-----\nFAKE\n-----END CERTIFICATE-----")
	if err := os.WriteFile(certPath, certContent, 0644); err != nil {
		t.Fatal(err)
	}

	img, err := Build(Config{
		Binary: binaryPath,
		AppDir: "/app",
		Name:   "myapp",
		CACert: certPath,
	})
	if err != nil {
		t.Fatalf("Build with CACert returned error: %v", err)
	}
	if img == nil {
		t.Fatal("Build with CACert returned nil image")
	}
}

func TestWriteTarball(t *testing.T) {
	t.Helper()

	tmp := t.TempDir()
	binaryPath := filepath.Join(tmp, "myapp")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\necho hello"), 0755); err != nil {
		t.Fatal(err)
	}

	img, err := Build(Config{
		Binary: binaryPath,
		AppDir: "/app",
		Name:   "myapp",
	})
	if err != nil {
		t.Fatal(err)
	}

	tarPath := filepath.Join(tmp, "out", "image.tar")
	if err := WriteTarball(img, tarPath, "myapp:latest"); err != nil {
		t.Fatalf("WriteTarball returned error: %v", err)
	}

	if _, err := os.Stat(tarPath); os.IsNotExist(err) {
		t.Fatal("WriteTarball did not create the tar file")
	}

	_, err = tarball.ImageFromPath(tarPath, nil)
	if err != nil {
		t.Fatalf("tarball.ImageFromPath could not read written tar: %v", err)
	}
}

func TestWriteTarball_CreatesParentDirs(t *testing.T) {
	t.Helper()

	tmp := t.TempDir()
	binaryPath := filepath.Join(tmp, "myapp")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\necho hello"), 0755); err != nil {
		t.Fatal(err)
	}

	img, _ := Build(Config{
		Binary: binaryPath,
		AppDir: "/app",
		Name:   "myapp",
	})

	depthPath := filepath.Join(tmp, "a", "b", "c", "image.tar")
	if err := WriteTarball(img, depthPath, "myapp:latest"); err != nil {
		t.Fatalf("WriteTarball with deep path returned error: %v", err)
	}

	if _, err := os.Stat(depthPath); os.IsNotExist(err) {
		t.Fatal("WriteTarball did not create parent directories")
	}
}

func TestFetchCACert(t *testing.T) {
	t.Helper()

	originalURL := caCertURL
	t.Cleanup(func() { caCertURL = originalURL })

	expectedContent := "FAKE CA CERT DATA"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedContent))
	}))
	t.Cleanup(srv.Close)

	caCertURL = srv.URL

	data, err := FetchCACert()
	if err != nil {
		t.Fatalf("FetchCACert returned error: %v", err)
	}

	if string(data) != expectedContent {
		t.Fatalf("FetchCACert returned %q, want %q", string(data), expectedContent)
	}
}

func TestFetchCACert_404(t *testing.T) {
	t.Helper()

	originalURL := caCertURL
	t.Cleanup(func() { caCertURL = originalURL })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	caCertURL = srv.URL

	_, err := FetchCACert()
	if err == nil {
		t.Fatal("FetchCACert with 404 expected error, got nil")
	}
}

func buildStubImage(t *testing.T) v1.Image {
	t.Helper()

	tmp := t.TempDir()
	binaryPath := filepath.Join(tmp, "stub")
	if err := os.WriteFile(binaryPath, []byte("stub"), 0755); err != nil {
		t.Fatal(err)
	}

	img, err := Build(Config{
		Binary: binaryPath,
		AppDir: "/app",
		Name:   "stub",
	})
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func TestPush_InvalidRepo(t *testing.T) {
	t.Helper()

	img := buildStubImage(t)
	cfg := PushConfig{Repo: ""}

	err := Push(img, cfg)
	if err == nil {
		t.Fatal("Push with empty repo expected error, got nil")
	}
}

func TestPush_BadRepoFormat(t *testing.T) {
	t.Helper()

	img := buildStubImage(t)
	cfg := PushConfig{Repo: ":"}

	err := Push(img, cfg)
	if err == nil {
		t.Fatal("Push with bad repo format expected error, got nil")
	}
}

func TestPush_PartialCredentials(t *testing.T) {
	t.Helper()

	img := buildStubImage(t)
	cfg := PushConfig{
		Repo: "example.com/repo",
		Tag:  "latest",
		User: "user",
		Pass: "",
	}

	err := Push(img, cfg)
	if err == nil {
		t.Fatal("Push with user but no pass expected error, got nil")
	}
}

func TestPush_PartialCredentialsReversed(t *testing.T) {
	t.Helper()

	img := buildStubImage(t)
	cfg := PushConfig{
		Repo: "example.com/repo",
		Tag:  "latest",
		User: "",
		Pass: "pass",
	}

	err := Push(img, cfg)
	if err == nil {
		t.Fatal("Push with pass but no user expected error, got nil")
	}
}
