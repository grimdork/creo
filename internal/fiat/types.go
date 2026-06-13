package fiat

type Var struct {
	Name  string
	Value string
	Eager bool
}

type OCIConfig struct {
	Repo       string
	Tag        string
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
}

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

type File struct {
	path    string
	Vars    map[string]*Var
	Targets []*Target
	segs    []segment
}
