package runner

import "runtime"

// archOrEmpty returns the given arch slice or a slice containing the empty string if nil/empty.
func archOrEmpty(a []string) []string {
	if len(a) == 0 {
		return []string{""}
	}
	return a
}

// osOrEmpty returns the given OS slice or a slice containing the empty string if nil/empty.
func osOrEmpty(o []string) []string {
	if len(o) == 0 {
		return []string{""}
	}
	return o
}

// ensureArch returns runtime.GOARCH when a is empty, otherwise a.
func ensureArch(a string) string {
	if a == "" {
		return runtime.GOARCH
	}
	return a
}

// ensureOS returns runtime.GOOS when o is empty, otherwise o.
func ensureOS(o string) string {
	if o == "" {
		return runtime.GOOS
	}
	return o
}

// hasCombo reports whether the arch and os appear together in the cross-product of archs and oses.
func hasCombo(archs, oses []string, arch, os string) bool {
	for _, a := range archs {
		for _, o := range oses {
			if a == arch && o == os {
				return true
			}
		}
	}
	return false
}
