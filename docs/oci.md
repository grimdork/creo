# OCI container images

OCI targets package a compiled binary into an OCI container image and
write a tarball or push to a registry.  They are packaging-only —
use `require=` to reference a build target and `$OUTPUT_<target>` to
locate its binary.

```
build: go

image: oci
    repo=ghcr.io/myorg/myapp
    tag=latest
    require=build
```

With a base image (e.g. Alpine for `/bin/sh` and libc):

```
build: go

image: oci
    from=alpine:latest
    repo=ghcr.io/myorg/myapp
    require=build
```

After `build` completes, its binary path is available as
`$OUTPUT_build`.  The image places the binary at `/app/<name>`
(override with `appdir=`).

## Properties

| Property | What it does |
|---|---|
| `repo=` | Container registry (e.g. `ghcr.io/user/repo`) |
| `tag=` | Image tag (default: `latest` for tarball; push uses this if set) |
| `tarball=` | Path to write a docker-compatible `.tar` file |
| `appdir=` | Directory in the image for the binary (default: `/app`) |
| `from=` | Base image to layer the binary on; pulls and caches to the platform user config dir (`~/.config/creo/oci/` on Linux, `~/Library/Application Support/creo/oci/` on macOS) |
| `entrypoint=` | Override the default entrypoint (`/app/<name>`) — e.g. `entrypoint=node /app/index.js` |
| `arch=` | Subset of architectures from the dependency |
| `os=` | Subset of operating systems from the dependency |
| `sbom=` | Set `sbom=true` to attach an SPDX 2.3 JSON SBOM at `/sbom.spdx.json` |
| `ociuser=` | Registry username (for basic auth) |
| `ocipass=` | Registry password or token |
| `ocicred=` | Credential helper command — prints `user:password` to stdout |
| `region=` | Registry region for `ecr` / `scw` aliases (e.g. `us-west-2`, `nl-ams`) |
| `cacert=` | CA certificate bundle — `auto` to download from curl.se, or a local file path |

If no `tarball=` and no `repo=` is set, the tarball defaults to
`$BUILDDIR/<target>.tar`.

## Directory layering

For interpreted languages (Python, Node, Java), the "binary" is a
directory.  The entire directory tree is added to the image, preserving
the structure under `appdir`:

```
image: oci
    require=build
    repo=ghcr.io/myorg/myapp
    from=python:3.11-slim
    entrypoint=python3 /app/main.py
```

Compiled languages (Go, C, C++, Rust) use single-binary layering —
only the compiled file is placed in the image.

## Entrypoint

When `entrypoint=` is not set, the default entrypoint is `/app/<name>`.
Set it to any command to customise how the image starts:

```
entrypoint=/srv/myapp --port 8080
entrypoint=node /app/index.js
entrypoint=python3 /app/main.py
```

## Registry aliases

Instead of writing out the full `repo=` URL, use a registry alias in
the language field:

```
deploy: oci:ghcr OWNER=myorg
    tag=latest
    arch=amd64
```

| Alias | `repo=` | Auth |
|---|---|---|
| `ghcr` | `ghcr.io/<owner>/<name>` | keychain |
| `docker` / `dockerhub` | `docker.io/<owner>/<name>` | keychain |
| `ecr` | `<owner>.dkr.ecr.<region>.amazonaws.com/<name>` | `ociuser=AWS` + `ocicred=aws ecr get-login-password --region <region>` |
| `gcr` | `gcr.io/<owner>/<name>` | keychain |
| `acr` | `<owner>.azurecr.io/<name>` | keychain |
| `scw` | `rg.<region>.scw.cloud/<owner>/<name>` | keychain |

`<owner>` is resolved from (in order):

1. Target-level `OWNER=myorg`
2. File-level `$OWNER=myorg`
3. `CREO_OWNER` environment variable
4. Git remote owner (ghcr only)
5. Directory basename

`<region>` is resolved from (in order):

1. `region=` property
2. File-level `$REGION`
3. `CREO_REGION` environment variable
4. Default: `us-east-1` (ECR) or `fr-par` (Scaleway)

Explicit `repo=`, `ociuser=`, `ocicred=`, or `region=` always override
the alias defaults.

## Extra files

Extra files (libraries, data files, config) can be added to the image
via a shared [manifest.ini](manifest.md):

```ini
[files]
/usr/lib/libfoo.so = lib/libfoo.so

[download]
/usr/lib/libbar.so = https://example.com/libbar.so
```

The `[files]` section copies local files into the image at the
specified destination path.  The `[download]` section fetches URLs
at build time and places them in the image.  Files are added as
individual layers in the order they appear in the manifest.

Without a manifest.ini, no extra files are added — only the binary
and CA certificate (if configured) are included.

## Base image caching

Base images (`from=`) are pulled once and cached at the platform
user config directory (`~/.config/creo/oci/` on Linux,
`~/Library/Application Support/creo/oci/` on macOS).  The cache
is valid for 24 hours; stale entries
are re-fetched automatically.  Use `--refresh-cacerts` to re-download
CA certificates only.

## Authentication

Auth priority:

1. `ociuser` + `ocipass` (explicit)
2. `ocicred=` helper command (prints `user:password` to stdout)
3. Default Docker keychain (`~/.docker/config.json`)

Store credentials for reuse with `creo -L` (interactive login that
writes to `~/.docker/config.json`).

## Multi-arch OCI

OCI targets implicitly set `os=linux` — containers always target
Linux regardless of the host OS.  Non-Linux combos from a dependency
are skipped with a warning.

OCI targets respect `arch=`/`os=` from their dependency but may
declare a subset:

```
build: go
    os=linux darwin
    arch=amd64 arm64
    bin=$bin-$os-$arch

image: oci
    repo=ghcr.io/myorg/myapp
    tag=$os-$arch
    require=build
```

A warning is printed for any dependency combos the OCI target skips.
