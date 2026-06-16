# Homebrew

Generate a Homebrew formula and optionally push it to a tap repository.

## Target

```
brew: brew
    require=archive
    tap=user/homebrew-tools
    desc=My CLI tool
    homepage=https://github.com/user/project
```

| Property | Default | Description |
|----------|---------|-------------|
| `tap` | — | GitHub tap repo (e.g. `user/homebrew-tools`) |
| `repo` | — | GitHub repo for download URL (e.g. `user/project`) |
| `homepage` | — | Formula homepage |
| `license` | `MIT` | Formula license |
| `output` | `$BUILDDIR/$PROJECT.rb` | Local formula output path |
| `token` | env `$GH_TOKEN` | GitHub token for tap push |

The formula references an archive produced by a dependency target
(typically `archive`). SHA256 is computed from the archive file
at build time.

If `tap` is set and `$GH_TOKEN` is available, creo clones the tap
repo, commits the formula, and pushes.

## Init

```
creo -i brew
```
