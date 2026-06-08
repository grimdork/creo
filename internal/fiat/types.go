package fiat

type Var struct {
	Name  string
	Value string
	Eager bool
}

type OCIConfig struct {
	Repo    string
	Tag     string
	Tarball string
	AppDir  string
	User    string
	Pass    string
}

type Target struct {
	Name      string
	Language  string
	Line      int
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
}

type File struct {
	path    string
	Vars    map[string]*Var
	Targets []*Target
	segs    []segment
}
