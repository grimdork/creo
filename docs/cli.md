# CLI reference

```
creo [flags] [target...]
```

Targets are positional: `creo debug test` runs both.  Without targets,
`build` is the default.  `all` runs every target.

## Flags

| Flag | Description |
|---|---|
| `-i`, `--init` | Initialise project (optionally with target types: `go`/`go:1.25`/`tinygo`/`c`/`cxx`/`cpp`/`rust`/`python`/`node`/`typescript`/`java`/`kotlin`/`gradle`/`oci`/`archive`/`deb`/`rpm`/`brew`; multiple accepted) |
| `-f`, `--file` | Alternative fiat file path |
| `-o`, `--output` | Build output directory (default: `build`) |
| `-F`, `--force` | Force rebuild |
| `-T`, `--template` | Project template name, used with `-i` (e.g. `creo -i go -T arg`); see [templates.md](templates.md) |
| `--save-template` | Extract an embedded template to the user template directory (`lang/name`); see [templates.md](templates.md) |
| `--list-templates` | List available project templates; see [templates.md](templates.md) |
| `-l`, `--list` | List available targets with descriptions |
| `-g`, `--git` | Initialise a git repository and commit (works standalone or after `--init`) |
| `-w`, `--watch` | Watch sources and rebuild on change |
| `-k`, `--keep-going` | Continue past errors, report all at the end |
| `-n`, `--dry-run` | Print commands and install actions without executing |
| `-j`, `--jobs` | Parallel jobs for multi-arch builds (default: CPU count) |
| `-r`, `--recursive` | Walk subdirectories for fiat files |
| `-c`, `--clean` | Remove target binaries and installed files |
| `-v`, `--verbose` | Show what's happening |
| `-L`, `--login` | Store registry credentials in `~/.docker/config.json` |
| `-I`, `--inspect` | Inspect a remote OCI image manifest and config |
| `--graph` | Draw dependency graph — `tree`, `dot`, or `svg` |
| `--status` | With `--graph`: annotate nodes with cache state |
| `--refresh-cacerts` | Re-download cached CA certificates |
| `--clean-cache` | Remove cached build artifacts |
| `--completion` | Print bash shell completion script |
| `--no-color` / `--no-colour` | Disable coloured terminal output |
| `--cache-remote` | SSH remote cache URL (`user@host:path` or `ssh://user@host/path`) |
| `--cache-stats` | Print L1 (local) and L2 (remote) cache hit/miss statistics |
| `--version` | Print version and exit |
| `-h`, `--help` | Show help |

## Target listing

```
creo -l
```

Prints every target, its language, and its `desc=` description:

```
  build       (go)   Build the project binary
  debug       (go)   Debug build with symbols
  install     (go)   Build and install to ~/bin
```

## Dependency graph

```
creo --graph tree
creo --graph dot | dot -Tpng -o graph.png
creo --graph svg > graph.svg
creo --graph svg --status > graph.svg
```

Three formats:

- **`tree`** — box-drawing text forest.  `--status` appends
  `[cached]`/`[stale]` annotations.
- **`dot`** — Graphviz DOT output.  Pipe to `dot -Tpng` for a rendered
  image.  `--status` colours nodes green/orange.
- **`svg`** — standalone SVG with layered DAG layout and inline
  rendering.  `--status` colours node borders green (cached) or orange
  (stale).  Open in any browser.

## Watch mode

```
creo -w [target]
```

Watches a target's source files and rebuilds on every change.  Default
target is `build`.  Polls every second (no external dependencies).

## Parallel builds

Multi-arch targets build each combination in parallel.  Use `-j N`
to limit concurrency:

```
creo -j 2 nix
```

Without `-j`, the number of CPUs is used.  `-j 1` runs serially.

## Shell completion

```
creo --completion
```

Outputs a bash completion script with tab completion for flags, target
names, and init languages.  Install:

```
source <(creo --completion)
```

Add it to `~/.bashrc` for persistence.
