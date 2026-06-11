# Node / TypeScript

Initialise a Node/TypeScript project:

```sh
$ creo -i node        # Node / TypeScript
$ creo -i typescript  # same
```

This creates `package.json`, `tsconfig.json`, `src/index.ts`, and a
`fiat` file (TypeScript target by default; change `scripts.build` in
`package.json` for pure-Node projects).

## Package manager detection

The package manager is detected from lockfile presence:

| Lockfile | Manager | Variable set |
|---|---|---|
| `pnpm-lock.yaml` | pnpm | `$PNPM=pnpm` |
| `yarn.lock` | yarn | `$YARN=yarn` |
| (none found) | npm | `$NPM=npm` |

The detected manager variable is used in the build command.

## Defaults

| Property | Value |
|---|---|
| `bin=` | `dist` — TypeScript output directory (used for OCI layering) |
| `cmd=` | `$<PM> run build` — where `<PM>` is the detected manager |
| `sources=` | `*.js *.jsx *.ts *.tsx package.json tsconfig.json` |

## Variables

| Variable | Default |
|---|---|
| `$NPM` | `npm` (if no pnpm/yarn lockfile) |
| `$YARN` | `yarn` (if `yarn.lock` exists) |
| `$PNPM` | `pnpm` (if `pnpm-lock.yaml` exists) |

## OCI packaging

Node/TypeScript projects use directory-based OCI layering — the `dist/`
output is added to the image at `/app/`:

```
image: oci
    require=build
    repo=ghcr.io/myorg/myapp
    from=node:20-slim
    entrypoint=node /app/index.js
```
