# Templates

Templates provide pre-built project scaffolds, used with the `-T` flag
alongside `--init`:

```
creo -i go -T basic
creo -i python -T cli
```

Without `-T`, `--init` uses its built-in hardcoded scaffolding.  With
`-T`, the template system drives all file creation.

## Resolution

Templates are resolved from two locations, in order:

1. **User directory** — `$CREO_DIR/templates/<lang>/<name>/`
2. **Embedded** — compiled into the binary

User templates override embedded ones with the same `lang/name`.

## Template format

Every template is a directory with a `template.ini` manifest:

```ini
[template]
name=basic
description=A basic Go project
language=go
files=main.go.tmpl, version.go.tmpl, fiat.tmpl

[vars]
DESCRIPTION=My project
```

### `[template]` keys

| Key | Required | Description |
|-----|----------|-------------|
| `name` | Yes | Template name, used with `-T` |
| `description` | No | Shown in `--list-templates` |
| `language` | Yes | Must match the language passed to `--init` |
| `files` | Yes | Comma-separated list of files to copy |

### `[vars]` section

Optional default values for `$VAR` expansion.  Users can override them
with `-D`:

```
creo -i go -T basic -D VERSION=1.0.0
```

## Variable expansion

Files with the `.tmpl` suffix have `$VAR` references expanded using
[fiat](fiat.md) expansion rules.  The `.tmpl` suffix is stripped from
the destination filename.

Files without `.tmpl` are copied as-is.

### Built-in variables

| Variable | Source | Description |
|----------|--------|-------------|
| `$PROJECT` | Directory name | Lowercased base name of the init directory |
| `$VERSION` | `-D VERSION=` | Defaults to `0.1.0` |

Any variable declared in `[vars]` or passed via `-D` is available.

## Platform variants

For templates that need OS-specific files, append `.GOOS` before `.tmpl`:

```
main.go.tmpl          # default
main.go.darwin.tmpl   # macOS override
main.go.linux.tmpl    # Linux override
```

The template system selects the variant matching `runtime.GOOS` if
present, falling back to the generic file.

## Commands

### List available templates

```
creo --list-templates
creo --list-templates go
```

Without a language filter, shows every template across all languages.

### Save a template to the user directory

```
creo --save-template go/basic
```

Copies the embedded template to `$CREO_DIR/templates/go/basic/` where
you can customise it.

## Built-in templates

| Language | Name | Description |
|----------|------|-------------|
| c | basic | Basic C project |
| cxx | basic | Basic C++ project |
| cxx | arg | C++ with `climate/arg`-style CLI |
| cxx | boost | C++ with `boost::program_options` |
| cxx | toolcmd | C++ with `climate/toolcmd`-style CLI |
| go | basic | Basic Go project |
| go | arg | CLI with `climate/arg` |
| go | toolcmd | CLI with `climate/toolcmd` |
| go | web | HTTP server with OCI target |
| java | basic | Java/Kotlin project with Gradle |
| node | basic | Node/TypeScript project |
| python | cli | Python CLI with argparse and uv |
| python | basic | Python project with hatchling build |
| rust | basic | Basic Rust binary |
| rust | arg | CLI with `climate/arg`-style parsing |
| rust | toolcmd | CLI with `climate/toolcmd`-style commands |
| tinygo | basic | Basic TinyGo project |
