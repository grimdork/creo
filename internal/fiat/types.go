package fiat

// Var represents a named variable with a value and optional eager-expansion flag.
type Var struct {
	Name  string
	Value string
	Eager bool
}

// ManifestFile pairs a source path with its install destination.
// Src is a local path or URL; Dst is the absolute or relative install path.
type ManifestFile struct {
	Dst string
	Src string
}

// OCIConfig holds OCI image build configuration.
type OCIConfig struct {
	Repo       string
	Tag        string
	Image      string
	Tarball    string
	AppDir     string
	User       string
	Pass       string
	CredHelper string
	Region     string
	CACert     string
	BaseImage  string
	SBOM       bool
	Entrypoint string
	Files      []ManifestFile
	Downloads  []ManifestFile
}

// Target represents a build target parsed from a fiat file.
type Target struct {
	Name      string
	Language  string
	LangAlias string
	Desc      string
	IsVirtual bool
	Cmds      []string
	Bin       string
	Sources   string
	Tmp       []string
	Requires  []string
	Arch      []string
	OS        []string
	Install   []string
	Vars      []*Var
	OCI       *OCIConfig
	Brew      *BrewConfig
}

// BrewConfig holds Homebrew formula configuration.
type BrewConfig struct {
	Tap       string
	Homepage  string
	License   string
	Desc      string
	Output    string
	Repo      string
	Token     string
	ClassName string
}

// MergeVars combines file-level and target-level variables into a new map.
// Target vars take precedence over file vars when names collide.
func MergeVars(fileVars map[string]*Var, targetVars []*Var) map[string]*Var {
	m := make(map[string]*Var, len(fileVars)+len(targetVars))
	for k, v := range fileVars {
		m[k] = v
	}
	for _, v := range targetVars {
		m[v.Name] = v
	}
	return m
}

// File represents a parsed fiat configuration file.
type File struct {
	path    string
	Vars    map[string]*Var
	Targets []*Target
	segs    []segment
}
