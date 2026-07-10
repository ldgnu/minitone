# Packaging minitone

## Debian / Ubuntu (`.deb`)

```bash
# from repo root
make build
make deb
# → dist/minitone_0.2.0_amd64.deb

sudo dpkg -i dist/minitone_*.deb
# if deps missing:
sudo apt-get install -f
```

Requires `dpkg-deb` (package `dpkg-dev`). Depends on `mpv`; recommends `yt-dlp`.

## Arch Linux (AUR)

### From source (`minitone`)

```bash
cd packaging/aur
# When a GitHub release tag v0.2.0 exists:
#   updpkgsums   # or: makepkg -g >> PKGBUILD
makepkg -si
```

Files:
- `PKGBUILD` — builds from the tagged source tarball
- `PKGBUILD-bin` — installs prebuilt release binary
- `minitone.install` — post-install hints

### Local test without a release tag

```bash
# build binary + tarball locally
make tarball

# or install straight from the git tree
cd /path/to/minitone
makepkg -si   # if you adapt PKGBUILD to source=(.)
```

Generate `.SRCINFO` for AUR submission:

```bash
make aur-srcinfo
# → packaging/aur/.SRCINFO
```

## Generic tarball

```bash
make tarball
# → dist/minitone-0.2.0-linux-amd64.tar.gz
```

## Release checklist

1. Bump `VERSION` in `Makefile`, `internal/app/app.go`, `packaging/aur/PKGBUILD*`
2. `make release` (test + packages)
3. Tag `vX.Y.Z` and push
4. Upload `dist/*` as GitHub release assets
5. Update AUR `sha256sums` with `makepkg -g`
