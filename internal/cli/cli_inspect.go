package cli

import "github.com/grimdork/creo/internal/oci"

// RunInspect displays the manifest and config of a remote OCI image.
func RunInspect(ref string) error {
	return oci.Inspect(ref)
}
