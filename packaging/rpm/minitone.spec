Name:           minitone
Version:        0.2.3
Release:        1%{?dist}
Summary:        TUI music player for YouTube, Radio Browser, Navidrome and local files
%global debug_package %{nil}

License:        MIT
URL:            https://github.com/ldgnu/minitone
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  golang >= 1.22
Requires:       mpv
Recommends:     yt-dlp

%description
minitone is a terminal music player that searches and plays from YouTube,
Radio Browser, Navidrome (Subsonic), your local library and favorites.
Playback is handled by mpv. Fuzzy search, multiple themes, queue with
shuffle/repeat, and favorites/history panels.

%prep
%setup -q -n %{name}-%{version}

%build
export CGO_ENABLED=0
go build -ldflags="-s -w -X github.com/ldgnu/minitone/internal/app.Version=%{version}" \
    -o minitone ./cmd/minitone/

%install
install -Dm755 minitone %{buildroot}%{_bindir}/minitone
install -Dm644 README.md %{buildroot}%{_datadir}/doc/%{name}/README.md
install -Dm644 LICENSE %{buildroot}%{_datadir}/licenses/%{name}/LICENSE

%files
%{_bindir}/minitone
%{_datadir}/doc/%{name}/README.md
%{_datadir}/licenses/%{name}/LICENSE

%changelog
* Fri Jul 10 2026 ldgnu <ldgnu@users.noreply.github.com> - 0.2.3-1
- all letters type in search; single-key shortcuts work with empty box
* Fri Jul 10 2026 ldgnu <ldgnu@users.noreply.github.com> - 0.2.2-1
- terminal (system) theme uses default fg; type j/k in search
* Fri Jul 10 2026 ldgnu <ldgnu@users.noreply.github.com> - 0.2.1-1
- fix typing j/k in search
* Fri Jul 10 2026 ldgnu <ldgnu@users.noreply.github.com> - 0.2.0-1
- multi-source TUI player (YouTube, Radio, Navidrome, library)
