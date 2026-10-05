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
.RI [ \-\-version | \-\-help | \-\-list\-screenshots ]
.br
.B minitone
.RB [ \-\-screenshot
.IR SCENARIO ]
.RI [ WIDTH ]\ [ HEIGHT ]\ [ THEME ]
.SH DESCRIPTION
minitone is a TUI music player. Type to search across YouTube, Radio Browser,
Navidrome/Subsonic and your local library. Press enter to play.
Results stream in per source, and the player bar at the bottom always shows
what is playing.
.SH OPTIONS
.TP
.B \-v, \-\-version
Print version and exit.
.TP
.B \-h, \-\-help
Print help and exit.
.TP
.B \-\-screenshot
.I SCENARIO
.RI [ WIDTH ] [ HEIGHT ] [ THEME ]
Render one UI state as text and exit, without touching mpv. Useful to inspect
a layout or to produce documentation images.
.TP
.B \-\-list-screenshots
Print the available screenshot scenarios and exit.
.SH KEYS
Two focus modes. In the search box every printable key is typed; press tab,
enter or the arrow keys to move into the results, where the single-key actions
live. Any other printable key returns to typing.
.TP
.B enter
Play the highlighted result and add it to the queue.
.TP
.B a / A
Add the highlighted result / all visible results to the queue.
.TP
.B f / i
Toggle favourite / show track details.
.TP
.B d / c
Remove the playing track from the queue / clear the queue.
.TP
.B J / K
Move the playing track up / down in the queue.
.TP
.B n / p
Next / previous track.
.TP
.B space
Play or pause (in the results; types a space while searching).
.TP
.B left / right
Seek \-5s / +5s.
.TP
.B + / \- / m
Volume up / down / mute.
.TP
.B ctrl+j / ctrl+f / ctrl+h / ctrl+l
Queue / favourites / history / local library panels.
.TP
.B ctrl+t / ctrl+v / ctrl+r / ctrl+u / ctrl+s
Theme / video mode / repeat / shuffle / rescan library.
.TP
.B ? / ctrl+/
Toggle the shortcut list.
.TP
.B esc
Back, one step at a time.
.TP
.B q / ctrl+c
Quit. ctrl+c always quits.
.SH SEARCH PREFIXES
A query may start with a source prefix (all followed by a space and the term):
.BR /search ,
.BR /youtube ,
.BR /radio ,
.BR /navidrome ,
.BR /local ,
.BR /fav .
.SH FILES
.TP
.I ~/.config/minitone/config.json
Configuration (Navidrome credentials, theme, library paths, session restore,
search debounce, keybindings).
.TP
.I ~/.config/minitone/favorites.json
Favorite tracks.
.TP
.I ~/.config/minitone/history.json
Play history.
.TP
.I ~/.config/minitone/session.json
Saved queue, position and playback preferences (used when restore_session is
enabled).
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
 servers, and local audio libraries. Results stream in per source, with a
 persistent player bar, a browseable queue and local library, session
 restore, and actionable errors. It works in narrow terminals.
EOF

# Optional changelog
cat > "${STAGE}/usr/share/doc/${PKGNAME}/changelog.Debian" <<EOF
minitone (${VERSION}) unstable; urgency=medium

  * Persistent player bar showing real playback state.
  * Search results stream in per source, with per-source errors and retry.
  * Single key actions (a, A, f, i, d, c, J, K, n, p) via search/browse
    focus modes.
  * Session restore: queue, position, volume, shuffle and repeat.
  * Browseable local library with scan progress and missing file detection.
  * Search prefixes (/youtube, /radio, /local, ...) and a source selector.
  * Actionable errors when mpv, yt-dlp or a source is unavailable.
  * Compact layout for narrow terminals.
  * Fixes: q quit, stop no longer skips tracks, playback errors surfaced,
    queue order no longer scrambled, no leaked mpv sockets.

 -- ldgnu <ldgnu@users.noreply.github.com>  $(date -R)
EOF
gzip -9n -f "${STAGE}/usr/share/doc/${PKGNAME}/changelog.Debian"

# Permissions for DEBIAN
chmod 755 "${STAGE}/DEBIAN"
chmod 644 "${STAGE}/DEBIAN/control"

mkdir -p "${DIST}"

# A .deb is an ar archive holding debian-binary + control.tar.* + data.tar.*.
# Prefer dpkg-deb, but assemble it by hand with ar+tar so `make deb` also works
# on non-Debian machines (Arch, Fedora) instead of silently producing a tarball
# named .deb that dpkg cannot install.
if command -v dpkg-deb >/dev/null 2>&1; then
	dpkg-deb --root-owner-group --build "${STAGE}" "${DEB}"
	echo "→ ${DEB}"
	exit 0
fi

if ! command -v ar >/dev/null 2>&1; then
	echo "error: neither dpkg-deb nor ar is available; cannot build a .deb" >&2
	echo "       install dpkg (Debian/Ubuntu) or binutils, or build from" >&2
	echo "       source: https://github.com/ldgnu/minitone#manual" >&2
	exit 1
fi

# Reproducible: sorted entries, pinned ownership and timestamps.
TAR_OPTS=(--sort=name --owner=0 --group=0 --numeric-owner --mtime=@0)

# data.tar holds only the payload: DEBIAN/ must stay out of it.
DATA_TAR="data.tar.gz"
if tar "${TAR_OPTS[@]}" --zstd --exclude=./DEBIAN -cf "${STAGE}.data.tar.zst" -C "${STAGE}" . 2>/dev/null; then
	DATA_TAR="data.tar.zst"
elif tar "${TAR_OPTS[@]}" --xz --exclude=./DEBIAN -cf "${STAGE}.data.tar.xz" -C "${STAGE}" . 2>/dev/null; then
	DATA_TAR="data.tar.xz"
else
	tar "${TAR_OPTS[@]}" --gzip --exclude=./DEBIAN -cf "${STAGE}.data.tar.gz" -C "${STAGE}" .
fi

# control.tar holds the control file at its ROOT, not under DEBIAN/.
tar "${TAR_OPTS[@]}" --gzip -cf "${STAGE}.control.tar.gz" -C "${STAGE}/DEBIAN" ./control
printf '2.0\n' > "${STAGE}/debian-binary"

# Members must be plain names, not paths, and ordered as dpkg expects.
WORK="$(mktemp -d)"
trap 'rm -rf "${WORK}"' EXIT
cp "${STAGE}/debian-binary" "${WORK}/debian-binary"
cp "${STAGE}.control.tar.gz" "${WORK}/control.tar.gz"
cp "${STAGE}.${DATA_TAR}" "${WORK}/${DATA_TAR}"

rm -f "${DEB}"
(cd "${WORK}" && ar rcD "${DEB}" debian-binary control.tar.gz "${DATA_TAR}") \
	|| (cd "${WORK}" && ar rc "${DEB}" debian-binary control.tar.gz "${DATA_TAR}")

rm -f "${STAGE}/debian-binary" "${STAGE}.control.tar.gz" "${STAGE}.${DATA_TAR}"
echo "→ ${DEB} (built with ar; install with: dpkg -i ${DEB})"
ls -lh "${DEB}"
