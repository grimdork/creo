# C and C++

Initialise a C or C++ project:

```sh
$ creo -i c          # C
$ creo -i cxx        # C++
$ creo -i cpp        # C++ (alias)
```

This creates `main.c` or `main.cpp` and a `fiat` file.

## Defaults (C)

| Property | Value |
|---|---|
| `bin=` | `$BUILDDIR/<name>` — from directory name |
| `cmd=` | `$CC $args $CFLAGS $LDFLAGS -o $bin $sources $LIBS` |
| `sources=` | `*.c *.h` |

## Defaults (C++)

| Property | Value |
|---|---|
| `bin=` | `$BUILDDIR/<name>` — from directory name |
| `cmd=` | `$CXX $args $CXXFLAGS $LDFLAGS -o $bin $sources $LIBS` |
| `sources=` | `*.cpp *.hpp *.hxx *.hh *.cppm *.ixx *.mpp` |

`build` targets get optimised flags; `debug` targets (and names ending
in `-debug`) get debug flags.

## Variables

| Variable | C default | C++ default |
|---|---|---|
| `$CC` | `cc` | — |
| `$CXX` / `$CPP` | — | `c++` |
| `$CFLAGS` | `-O2 -Wall` | — |
| `$CXXFLAGS` / `$CPPFLAGS` | — | `-O2 -Wall` |
| `$CDEBUGFLAGS` | `-O0 -g -Wall` | — |
| `$CXXDEBUGFLAGS` / `$CPPDEBUGFLAGS` | — | `-O0 -g -Wall` |
| `$LDFLAGS` | (empty) | (empty) |
| `$LIBS` | (empty) | (empty) |

Set `$CC` or `$CXX` to a cross-compiler prefix for multi-arch builds:

```
nix: c
    os=linux
    arch=arm64
    $CC=aarch64-linux-gnu-gcc
```

Cross-compilation env vars are not set automatically for C/C++ — you
must configure the toolchain via the compiler variable.
