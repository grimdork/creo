package runner

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/grimdork/climate/fx"
)

type remoteCache struct {
	user string
	host string
	path string
}

func parseRemoteCacheURL(raw string) (*remoteCache, error) {
	if raw == "" {
		return nil, fmt.Errorf("empty cache URL")
	}

	var user, host, path string

	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("parsing cache URL %q: %w", raw, err)
		}
		if u.Scheme != "ssh" {
			return nil, fmt.Errorf("unsupported scheme %q in cache URL %q (use ssh)", u.Scheme, raw)
		}
		user = u.User.Username()
		host = u.Hostname()
		path = u.Path
		if path == "/" || path == "" {
			path = ".creo/cache"
		}
	} else if strings.Contains(raw, "@") {
		idx := strings.IndexByte(raw, ':')
		if idx < 0 {
			return nil, fmt.Errorf("cache URL %q: missing path after colon", raw)
		}
		userHost := raw[:idx]
		path = raw[idx+1:]
		parts := strings.SplitN(userHost, "@", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("cache URL %q: expected user@host:path", raw)
		}
		user = parts[0]
		host = parts[1]
	} else {
		return nil, fmt.Errorf("cache URL %q: expected ssh://user@host/path or user@host:path", raw)
	}

	if user == "" || host == "" {
		return nil, fmt.Errorf("cache URL %q: missing user or host", raw)
	}
	if user == "root" {
		return nil, fmt.Errorf("SSH cache does not allow root connections: %s", raw)
	}
	if path == "" {
		path = ".creo/cache"
	}

	return &remoteCache{user: user, host: host, path: path}, nil
}

func sanitisePathComponent(s string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			return r
		}
		return '_'
	}, s)
}

func (r *remoteCache) remotePath(elem ...string) string {
	safe := make([]string, len(elem))
	for i, e := range elem {
		safe[i] = sanitisePathComponent(e)
	}
	parts := append([]string{r.path}, safe...)
	return r.user + "@" + r.host + ":" + strings.Join(parts, "/")
}

func rsyncArgs() []string {
	return []string{"-az", "-e", "ssh"}
}

func (r *remoteCache) fetchManifest(comboKey string, sources, cmds []string) (string, bool) {
	key, err := computeCacheKey(sources, cmds)
	if err != nil {
		return "", false
	}

	tmpDir, err := os.MkdirTemp("", "creo-remote-cache-*")
	if err != nil {
		return "", false
	}
	defer os.RemoveAll(tmpDir)

	localTmp := filepath.Join(tmpDir, "manifest.json")
	args := append(rsyncArgs(), r.remotePath(comboKey+"_"+key+".json"), localTmp)
	cmd := exec.Command("rsync", args...)
	if err := cmd.Run(); err != nil {
		return "", false
	}

	data, err := os.ReadFile(localTmp)
	if err != nil {
		return "", false
	}
	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return "", false
	}
	if entry.Key != key {
		return "", false
	}
	return key, true
}

func (r *remoteCache) downloadArtifacts(hash, comboKey, localDir string) error {
	remoteDir := r.remotePath(comboKey+"_"+hash) + "/"
	localDir = filepath.Clean(localDir) + "/"

	args := append(rsyncArgs(), remoteDir, localDir)
	cmd := exec.Command("rsync", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("rsync pull: %w\n%s", err, string(out))
	}
	return nil
}

func (r *remoteCache) uploadArtifacts(hash, comboKey, localBin string) error {
	remoteDir := r.remotePath(comboKey+"_"+hash) + "/"

	remoteEnsureDir(r, comboKey+"_"+hash)

	args := append(rsyncArgs(), localBin, remoteDir)
	cmd := exec.Command("rsync", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("rsync push: %w\n%s", err, string(out))
	}
	return nil
}

func (r *remoteCache) uploadManifest(hash, comboKey, localCacheJSON string) error {
	remoteManifest := r.remotePath(comboKey + "_" + hash + ".json")
	args := append(rsyncArgs(), localCacheJSON, remoteManifest)
	cmd := exec.Command("rsync", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("rsync manifest push: %w\n%s", err, string(out))
	}
	return nil
}

func remoteEnsureDir(r *remoteCache, dir string) {
	mkdirCmd := exec.Command("ssh", r.user+"@"+r.host, "mkdir", "-p", filepath.Join(r.path, dir))
	if err := mkdirCmd.Run(); err != nil {
		fx.Fprint(os.Stderr, "  {warning}remote mkdir {:q}: {}{@}\n", filepath.Join(r.path, dir), err)
	}
}

func tryRemoteCache(remoteURL, comboKey string, sources, cmds []string) (string, bool) {
	remote, err := parseRemoteCacheURL(remoteURL)
	if err != nil {
		fx.Fprint(os.Stderr, "  {warning}remote cache: {}{@}\n", err)
		return "", false
	}
	return remote.fetchManifest(comboKey, sources, cmds)
}

func pullAndSave(remoteURL, hash, comboKey, localBin string) bool {
	remote, err := parseRemoteCacheURL(remoteURL)
	if err != nil {
		return false
	}
	if err := remote.downloadArtifacts(hash, comboKey, filepath.Dir(localBin)); err != nil {
		fx.Fprint(os.Stderr, "  {warning}remote cache pull: {}{@}\n", err)
		return false
	}
	return true
}

func pushRemote(remoteURL, hash, comboKey, localBin, dir string, sources, cmds []string) {
	remote, err := parseRemoteCacheURL(remoteURL)
	if err != nil {
		fx.Fprint(os.Stderr, "  {warning}remote cache URL: {}{@}\n", err)
		return
	}

	key, err := computeCacheKey(sources, cmds)
	if err != nil || key != hash {
		return
	}

	if err := remote.uploadArtifacts(hash, comboKey, localBin); err != nil {
		fx.Fprint(os.Stderr, "  {warning}remote cache push: {}{@}\n", err)
		return
	}

	localManifest := cachePath(dir, comboKey)
	if err := remote.uploadManifest(hash, comboKey, localManifest); err != nil {
		fx.Fprint(os.Stderr, "  {warning}remote manifest upload: {}{@}\n", err)
	}
}
