#!/usr/bin/env bash
# Build a .deb package for minitone (no external tooling beyond dpkg-deb/ar).
set -euo pipefail

VERSION="${1:-0.2.0}"
ARCH="${2:-amd64}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST="${ROOT}/dist"
PKGNAME="minitone"
STAGE="${DIST}/deb-root"
DEB="${DIST}/${PKGNAME}_${VERSION}_${ARCH}.deb"

rm -rf "${STAGE}"
mkdir -p "${STAGE}/DEBIAN" \
	"${STAGE}/usr/bin" \
	"${STAGE}/usr/share/doc/${PKGNAME}" \
	"${STAGE}/usr/share/man/man1"

# Binary
if [[ ! -x "${ROOT}/minitone" ]]; then
	(cd "${ROOT}" && go build -trimpath -ldflags="-s -w -X github.com/ldgnu/minitone/internal/app.Version=${VERSION}" -o minitone ./cmd/minitone/)
fi
install -Dm755 "${ROOT}/minitone" "${STAGE}/usr/bin/minitone"

# Docs
install -Dm644 "${ROOT}/README.md" "${STAGE}/usr/share/doc/${PKGNAME}/README.md"
if [[ -f "${ROOT}/LICENSE" ]]; then
	install -Dm644 "${ROOT}/LICENSE" "${STAGE}/usr/share/doc/${PKGNAME}/copyright"
else
	cat > "${STAGE}/usr/share/doc/${PKGNAME}/copyright" <<EOF
Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/
Upstream-Name: minitone
Source: https://github.com/ldgnu/minitone

Files: *
Copyright: ldgnu
License: MIT
EOF
fi

# Man page
cat > "${STAGE}/usr/share/man/man1/minitone.1" <<'EOF'
.TH MINITONE 1 "2026" "minitone" "User Commands"
.SH NAME
minitone \- terminal music player (YouTube, Radio, Navidrome, local)
.SH SYNOPSIS
.B minitone
[\fB\-\-version\fR|\fB\-\-help\fR]
.SH DESCRIPTION
minitone is a TUI music player. Type to search across YouTube, Radio Browser,
Navidrome/Subsonic and your local library. Press enter to play.
.SH OPTIONS
.TP
.B \-v, \-\-version
Print version and exit.
.TP
.B \-h, \-\-help
Print help and exit.
.SH FILES
.TP
.I ~/.config/minitone/config.json
Configuration (Navidrome credentials, theme, library paths).
.TP
.I ~/.config/minitone/favorites.json
Favorite tracks.
.TP
.I ~/.config/minitone/history.json
Play history.
.SH SEE ALSO
.BR mpv (1),
.BR yt-dlp (1)
EOF
gzip -9n -f "${STAGE}/usr/share/man/man1/minitone.1"

# Control
SIZE_KB=$(du -sk "${STAGE}/usr" | cut -f1)
cat > "${STAGE}/DEBIAN/control" <<EOF
Package: ${PKGNAME}
Version: ${VERSION}
Section: sound
Priority: optional
Architecture: ${ARCH}
Maintainer: ldgnu <ldgnu@users.noreply.github.com>
Depends: mpv
Recommends: yt-dlp
Installed-Size: ${SIZE_KB}
Homepage: https://github.com/ldgnu/minitone
Description: TUI music player for YouTube, Radio, Navidrome and local files
 minitone is a terminal user interface music player backed by mpv.
 It searches YouTube (via yt-dlp), Radio Browser, optional Navidrome
 servers, and local audio libraries. Supports queue, shuffle, repeat,
 favorites and play history.
EOF

# Optional changelog
cat > "${STAGE}/usr/share/doc/${PKGNAME}/changelog.Debian" <<EOF
minitone (${VERSION}) unstable; urgency=medium

  * Release ${VERSION}: favorites, history, packaging.

 -- ldgnu <ldgnu@users.noreply.github.com>  $(date -R)
EOF
gzip -9n -f "${STAGE}/usr/share/doc/${PKGNAME}/changelog.Debian"

# Permissions for DEBIAN
chmod 755 "${STAGE}/DEBIAN"
chmod 644 "${STAGE}/DEBIAN/control"

mkdir -p "${DIST}"
if command -v dpkg-deb >/dev/null 2>&1; then
	dpkg-deb --root-owner-group --build "${STAGE}" "${DEB}"
else
	# Fallback: produce a simple tarball named .deb-like if dpkg-deb missing
	echo "warning: dpkg-deb not found; creating tarball instead" >&2
	tar -C "${STAGE}" -czf "${DIST}/${PKGNAME}_${VERSION}_${ARCH}.tar.gz" .
	echo "→ ${DIST}/${PKGNAME}_${VERSION}_${ARCH}.tar.gz"
	exit 0
fi

echo "→ ${DEB}"
ls -lh "${DEB}"
