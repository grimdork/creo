package targets

const (
	errGoModInit    = "go mod init: %s"
	errCargoInit    = "cargo init: %s"
	errUnknownLang  = "%s: unknown language %q"
	errCreating     = "creating %s: %w"
	errWriting      = "writing %s: %w"
	errRemoving     = "removing %s: %w"
	errGofmt        = "gofmt: %s"
	errGoImports    = "goimports: %s"
	errGoModTidy    = "go mod tidy: %s"
	errRemovingCreo = "removing .creo: %w"
)
