package lang

type Var struct {
	Name  string
	Value string
	Eager bool
}

type Target struct {
	Name     string
	Language string
	Line     int
	Cmds     []string
	Bin      string
	Sources  string
	Tmp      []string
	Requires []string
	Arch     []string
	OS       []string
	Install  []string
	Vars     []*Var
}

type FiatFile struct {
	Path    string
	Vars    map[string]*Var
	Targets []*Target
}
