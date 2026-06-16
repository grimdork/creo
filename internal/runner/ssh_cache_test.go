package runner

import (
	"testing"
)

func TestParseRemoteCacheURLSSH(t *testing.T) {
	r, err := parseRemoteCacheURL("ssh://alice@cache.example.com/var/cache")
	if err != nil {
		t.Fatal(err)
	}
	if r.user != "alice" {
		t.Fatalf("expected user alice, got %q", r.user)
	}
	if r.host != "cache.example.com" {
		t.Fatalf("expected host cache.example.com, got %q", r.host)
	}
	if r.path != "/var/cache" {
		t.Fatalf("expected path /var/cache, got %q", r.path)
	}
}

func TestParseRemoteCacheURLUserHostPort(t *testing.T) {
	r, err := parseRemoteCacheURL("ssh://bob@build.local:2222/opt/creo-cache")
	if err != nil {
		t.Fatal(err)
	}
	if r.user != "bob" {
		t.Fatalf("expected user bob, got %q", r.user)
	}
	if r.host != "build.local" {
		t.Fatalf("expected host build.local, got %q", r.host)
	}
	if r.path != "/opt/creo-cache" {
		t.Fatalf("expected path /opt/creo-cache, got %q", r.path)
	}
}

func TestParseRemoteCacheURLScpStyle(t *testing.T) {
	r, err := parseRemoteCacheURL("deploy@remote:.creo/cache")
	if err != nil {
		t.Fatal(err)
	}
	if r.user != "deploy" {
		t.Fatalf("expected user deploy, got %q", r.user)
	}
	if r.host != "remote" {
		t.Fatalf("expected host remote, got %q", r.host)
	}
	if r.path != ".creo/cache" {
		t.Fatalf("expected path .creo/cache, got %q", r.path)
	}
}

func TestParseRemoteCacheURLDefaultPath(t *testing.T) {
	r, err := parseRemoteCacheURL("user@host:")
	if err != nil {
		t.Fatal(err)
	}
	if r.path != ".creo/cache" {
		t.Fatalf("expected default path .creo/cache, got %q", r.path)
	}
}

func TestParseRemoteCacheURLDefaultPathSSH(t *testing.T) {
	r, err := parseRemoteCacheURL("ssh://user@host")
	if err != nil {
		t.Fatal(err)
	}
	if r.path != ".creo/cache" {
		t.Fatalf("expected default path .creo/cache, got %q", r.path)
	}
}

func TestParseRemoteCacheURLEmpty(t *testing.T) {
	_, err := parseRemoteCacheURL("")
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestParseRemoteCacheURLRootRejected(t *testing.T) {
	_, err := parseRemoteCacheURL("ssh://root@host/path")
	if err == nil {
		t.Fatal("expected error for root user")
	}
}

func TestParseRemoteCacheURLRootRejectedScp(t *testing.T) {
	_, err := parseRemoteCacheURL("root@host:/path")
	if err == nil {
		t.Fatal("expected error for root user")
	}
}

func TestParseRemoteCacheURLMissingUser(t *testing.T) {
	_, err := parseRemoteCacheURL("ssh://host/path")
	if err == nil {
		t.Fatal("expected error for missing user")
	}
}

func TestParseRemoteCacheURLMissingHost(t *testing.T) {
	_, err := parseRemoteCacheURL("ssh://user@/path")
	if err == nil {
		t.Fatal("expected error for missing host")
	}
}

func TestParseRemoteCacheURLScpMissingColon(t *testing.T) {
	_, err := parseRemoteCacheURL("user@host")
	if err == nil {
		t.Fatal("expected error for missing colon")
	}
}

func TestParseRemoteCacheURLScpMissingUser(t *testing.T) {
	_, err := parseRemoteCacheURL("host:/path")
	if err == nil {
		t.Fatal("expected error for missing user (no @)")
	}
}

func TestParseRemoteCacheURLBadScheme(t *testing.T) {
	_, err := parseRemoteCacheURL("https://user@host/path")
	if err == nil {
		t.Fatal("expected error for invalid scheme")
	}
}

func TestParseRemoteCacheURLGarbage(t *testing.T) {
	_, err := parseRemoteCacheURL("!@#$%^")
	if err == nil {
		t.Fatal("expected error for garbage input")
	}
}

func TestParseRemoteCacheURLRemotePath(t *testing.T) {
	r, err := parseRemoteCacheURL("jenkins@ci.example.com:/var/lib/creo/cache")
	if err != nil {
		t.Fatal(err)
	}
	got := r.remotePath("build_linux_amd64_abc123.json")
	want := "jenkins@ci.example.com:/var/lib/creo/cache/build_linux_amd64_abc123.json"
	if got != want {
		t.Fatalf("remotePath: got %q, want %q", got, want)
	}
}

func TestValidGraphFormat(t *testing.T) {
	if !ValidGraphFormat("tree") {
		t.Error("expected tree to be valid")
	}
	if !ValidGraphFormat("dot") {
		t.Error("expected dot to be valid")
	}
	if !ValidGraphFormat("svg") {
		t.Error("expected svg to be valid")
	}
	if ValidGraphFormat("png") {
		t.Error("expected png to be invalid")
	}
	if ValidGraphFormat("") {
		t.Error("expected empty to be invalid")
	}
}
