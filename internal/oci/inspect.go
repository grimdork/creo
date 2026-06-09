package oci

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/grimdork/creo/internal/util"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

func Inspect(imageRef string) error {
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return fmt.Errorf("invalid reference %q: %w", imageRef, err)
	}

	auth, err := authn.DefaultKeychain.Resolve(ref.Context())
	if err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	desc, err := remote.Get(ref, remote.WithAuth(auth))
	if err != nil {
		return fmt.Errorf("fetching %q: %w", imageRef, err)
	}

	img, err := desc.Image()
	if err != nil {
		return fmt.Errorf("reading image: %w", err)
	}

	manifest, err := img.Manifest()
	if err != nil {
		return fmt.Errorf("reading manifest: %w", err)
	}

	cfg, err := img.ConfigFile()
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintf(w, "Repository:\t%s\n", imageRef)
	fmt.Fprintf(w, "Digest:\t%s\n", desc.Digest.String())
	fmt.Fprintf(w, "Media type:\t%s\n", desc.MediaType)
	fmt.Fprintf(w, "OS/Arch:\t%s/%s\n", cfg.OS, cfg.Architecture)
	fmt.Fprintf(w, "Created:\t%s\n", cfg.Created.Time)
	fmt.Fprintf(w, "Layer count:\t%d\n", len(manifest.Layers))
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Layers:")
	for _, layer := range manifest.Layers {
		fmt.Fprintf(w, "  %s\t%s\n", layer.Digest.String(), util.FmtSize(layer.Size))
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Config:")
	fmt.Fprintf(w, "  Cmd:\t%v\n", cfg.Config.Cmd)
	fmt.Fprintf(w, "  Entrypoint:\t%v\n", cfg.Config.Entrypoint)
	fmt.Fprintf(w, "  Env:\t%v\n", cfg.Config.Env)
	fmt.Fprintf(w, "  Labels:\t%v\n", cfg.Config.Labels)

	return w.Flush()
}


