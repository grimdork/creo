package oci

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type PushConfig struct {
	Repo string
	Tag  string
	User string
	Pass string
}

// Push uploads an OCI image to a remote registry.
func Push(img v1.Image, cfg PushConfig) error {
	refStr := cfg.Repo
	if cfg.Tag != "" {
		refStr = cfg.Repo + ":" + cfg.Tag
	}

	ref, err := name.ParseReference(refStr)
	if err != nil {
		return fmt.Errorf("invalid repo reference %q: %w", refStr, err)
	}

	var auth authn.Authenticator
	if cfg.User != "" && cfg.Pass != "" {
		auth = &authn.Basic{
			Username: cfg.User,
			Password: cfg.Pass,
		}
	} else if cfg.User != "" || cfg.Pass != "" {
		return fmt.Errorf("ociuser and ocipass must both be set or both be empty")
	} else {
		auth, err = authn.DefaultKeychain.Resolve(ref.Context())
		if err != nil {
			return fmt.Errorf(errAuth, err)
		}
	}

	opts := []remote.Option{remote.WithAuth(auth)}

	return remote.Write(ref, img, opts...)
}
