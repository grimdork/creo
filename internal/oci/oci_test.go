package oci

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grimdork/creo/internal/util"
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

func TestFmtSize(t *testing.T) {
	tests := []struct {
		size int64
		want string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KiB"},
		{2048, "2.0 KiB"},
		{1048576, "1.0 MiB"},
		{1073741824, "1.0 GiB"},
	}
	for _, tt := range tests {
		got := util.FmtSize(tt.size)
		if got != tt.want {
			t.Errorf("fmtSize(%d) = %q, want %q", tt.size, got, tt.want)
		}
	}
}

func TestInspect_InvalidRef(t *testing.T) {
	err := Inspect(":::invalid")
	if err == nil {
		t.Fatal("Inspect with invalid ref expected error, got nil")
	}
}

func TestInspect_EmptyRef(t *testing.T) {
	err := Inspect("")
	if err == nil {
		t.Fatal("Inspect with empty ref expected error, got nil")
	}
}

func TestLogin_WritesConfig(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldStdin := os.Stdin
	os.Stdin = r

	w.Write([]byte("docker.io\n"))
	w.Write([]byte("testuser\n"))
	w.Write([]byte("testpass\n"))
	w.Close()

	err = Login()
	os.Stdin = oldStdin
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}

	configPath := tmp + "/.docker/config.json"
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	var cfg dockerConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("invalid config JSON: %v", err)
	}

	entry, ok := cfg.Auths[dockerIOKey]
	if !ok {
		t.Fatal("expected auth entry for docker.io")
	}
	if entry.Auth == "" {
		t.Fatal("expected non-empty auth field")
	}
}

func TestCacheKeyName(t *testing.T) {
	key1 := cacheKeyName("alpine:latest", "amd64", "linux")
	key2 := cacheKeyName("alpine:latest", "amd64", "linux")
	if key1 != key2 {
		t.Fatal("cache key not deterministic")
	}
	if key1 == "" {
		t.Fatal("cache key empty")
	}
}

func TestLoadFromCache_Missing(t *testing.T) {
	img := loadFromCache("/nonexistent/path.tar")
	if img != nil {
		t.Fatal("expected nil for missing cache")
	}
}

func TestSaveAndLoadCache(t *testing.T) {
	img := buildStubImage(t)
	cachePath := filepath.Join(t.TempDir(), "cache.tar")

	if err := saveToCache(img, cachePath); err != nil {
		t.Fatalf("saveToCache: %v", err)
	}

	loaded := loadFromCache(cachePath)
	if loaded == nil {
		t.Fatal("loadFromCache returned nil for valid cache")
	}
}

func TestLoadFromCache_Expired(t *testing.T) {
	img := buildStubImage(t)
	cachePath := filepath.Join(t.TempDir(), "expired.tar")

	if err := saveToCache(img, cachePath); err != nil {
		t.Fatalf("saveToCache: %v", err)
	}

	past := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(cachePath, past, past); err != nil {
		t.Fatal(err)
	}

	loaded := loadFromCache(cachePath)
	if loaded != nil {
		t.Fatal("loadFromCache should return nil for expired cache")
	}
}

func TestBuildWithBaseImage_InvalidRef(t *testing.T) {
	tmp := t.TempDir()
	binaryPath := filepath.Join(tmp, "myapp")
	if err := os.WriteFile(binaryPath, []byte("binary"), 0755); err != nil {
		t.Fatal(err)
	}

	_, err := Build(Config{
		Binary:    binaryPath,
		AppDir:    "/app",
		Name:      "myapp",
		BaseImage: ":::invalid",
	})
	if err == nil {
		t.Fatal("Build with invalid base image ref expected error, got nil")
	}
}

func TestCacheDirectory_CreatesDir(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")

	dir, err := cacheDirectory()
	if err != nil {
		t.Fatalf("cacheDirectory: %v", err)
	}
	if dir == "" {
		t.Fatal("cacheDirectory returned empty path")
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Fatal("cacheDirectory did not create the directory")
	}
}

func TestGenerateSBOM(t *testing.T) {
	tmp := t.TempDir()
	binaryPath := filepath.Join(tmp, "myapp")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\necho hello"), 0755); err != nil {
		t.Fatal(err)
	}

	data, err := generateSBOM(binaryPath, "myapp")
	if err != nil {
		t.Fatalf("generateSBOM: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("generateSBOM returned empty data")
	}
	if data[0] != '{' {
		t.Fatal("generateSBOM did not return JSON object")
	}
}

func TestSBOMLayer(t *testing.T) {
	layer, err := sbomLayer([]byte(`{"test": true}`))
	if err != nil {
		t.Fatalf("sbomLayer: %v", err)
	}
	if layer == nil {
		t.Fatal("sbomLayer returned nil")
	}
}

func TestBuildWithSBOM(t *testing.T) {
	tmp := t.TempDir()
	binaryPath := filepath.Join(tmp, "myapp")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\necho hello"), 0755); err != nil {
		t.Fatal(err)
	}

	img, err := Build(Config{
		Binary: binaryPath,
		AppDir: "/app",
		Name:   "myapp",
		SBOM:   true,
	})
	if err != nil {
		t.Fatalf("Build with SBOM: %v", err)
	}
	if img == nil {
		t.Fatal("Build returned nil image")
	}

	manifest, err := img.Manifest()
	if err != nil {
		t.Fatal(err)
	}
	// binary layer + SBOM layer
	if len(manifest.Layers) != 2 {
		t.Fatalf("expected 2 layers with SBOM, got %d", len(manifest.Layers))
	}
}

func TestLogin_EmptyUsername(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	r, w, _ := os.Pipe()
	oldStdin := os.Stdin
	os.Stdin = r
	w.Write([]byte("registry.example.com\n"))
	w.Write([]byte("\n"))
	w.Close()

	err := Login()
	os.Stdin = oldStdin
	if err == nil {
		t.Fatal("Login with empty username expected error, got nil")
	}
}
