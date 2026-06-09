package oci

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/term"
)

type dockerConfig struct {
	Auths map[string]dockerAuthEntry `json:"auths"`
}

type dockerAuthEntry struct {
	Auth string `json:"auth,omitempty"`
}

const dockerIOKey = "https://index.docker.io/v1/"

func Login() error {
	fmt.Print("Registry (default: docker.io): ")
	var registry string
	if _, err := fmt.Scanln(&registry); err != nil && registry == "" {
		registry = "docker.io"
	}
	if registry == "docker.io" || registry == "index.docker.io" {
		registry = dockerIOKey
	}

	fmt.Print("Username: ")
	var username string
	if _, err := fmt.Scanln(&username); err != nil {
		return fmt.Errorf("reading username: %w", err)
	}
	if username == "" {
		return fmt.Errorf("username required")
	}

	fmt.Print("Password: ")
	var password string
	passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		var pass string
		if _, err := fmt.Scanln(&pass); err != nil {
			return fmt.Errorf("reading password: %w", err)
		}
		password = pass
	} else {
		password = string(passwordBytes)
	}
	fmt.Println()
	if password == "" {
		return fmt.Errorf("password required")
	}

	auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))


	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}
	configPath := filepath.Join(home, ".docker", "config.json")

	var raw map[string]any
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &raw); err != nil {
			return fmt.Errorf("parsing existing config: %w", err)
		}
	}
	if raw == nil {
		raw = make(map[string]any)
	}
	auths, _ := raw["auths"].(map[string]any)
	if auths == nil {
		auths = make(map[string]any)
	}
	auths[registry] = dockerAuthEntry{Auth: auth}
	raw["auths"] = auths

	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}
