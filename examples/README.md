# creo examples

Each `.fiat` file is self-documenting with `#` comments explaining every
property.

| File | Language | Key features demonstrated |
|---|---|---|
| `basic.fiat` | Go | Minimal one-line build |
| `subpackage.fiat` | Go | `SRCDIR`, `cmd/<name>` layout |
| `debug.fiat` | Go | Release + debug targets |
| `cross-compile.fiat` | Go | `arch`, `os`, `$bin-$os-$arch` |
| `install.fiat` | Go | `install=`, `$LDFLAGS`, built-in vars |
| `c.fiat` | C | `c` language, `$CC`, `$CFLAGS` |
| `cxx.fiat` | C++ | `cxx`/`cpp` language, `$CXXFLAGS` |
| `oci.fiat` | oci | Container image via built-in OCI builder |
| `virtual.fiat` | — | `.test`, `.lint`, `.release` virtual targets |
| `full.fiat` | All | Combined: build + nix + image + install + test + lint |

## Quick reference

```sh
creo                   # default: build
creo -l                # list targets with descriptions
creo -v                # verbose output
creo -n                # dry run (print commands, don't execute)
creo -c                # clean (remove binaries and installed files)
creo -k                # keep going past errors
creo -j 4              # parallel multi-arch builds (4 jobs)
creo -w                # watch sources and rebuild on change
creo all               # run every target
creo <target> <...>    # run specific targets
```
