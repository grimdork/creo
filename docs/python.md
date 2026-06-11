# Python

Initialise a Python project:

```sh
$ creo -i python
```

This creates `pyproject.toml`, `src/<name>/main.py`, `src/<name>/__init__.py`,
and a `fiat` file.

## Defaults

| Property | Value |
|---|---|
| `bin=` | `src` — source directory (used for OCI directory layering) |
| `cmd=` | `$UV sync --frozen` |
| `sources=` | `*.py pyproject.toml setup.py setup.cfg` |

The `src` directory is used directly as the application root in OCI
images.  If you need a different source layout, set `bin=` explicitly.

## Variables

| Variable | Default |
|---|---|
| `$UV` | `uv` |
| `$PYTHON` | `python3` |

## OCI packaging

Python projects use directory-based OCI layering — the entire `src/`
tree is added to the image at `/app/`:

```
image: oci
    require=build
    repo=ghcr.io/myorg/myapp
    from=python:3.11-slim
    entrypoint=python3 /app/main.py
```
