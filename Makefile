APP       := minitone
VERSION   := 0.4.0
PREFIX    := /usr
BINDIR    := $(PREFIX)/bin
GOFLAGS   := -trimpath
LDFLAGS   := -s -w -X github.com/ldgnu/minitone/internal/app.Version=$(VERSION)
DIST      := dist
ARCH      := $(shell uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')

.PHONY: all build install uninstall test test-race test-short vet fmt clean \
	package tarball deb aur-srcinfo aur-push release help screenshot

all: build

build:
	go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(APP) ./cmd/minitone/

install: build
	install -Dm755 $(APP) $(DESTDIR)$(BINDIR)/$(APP)
	install -Dm644 README.md $(DESTDIR)$(PREFIX)/share/doc/$(APP)/README.md
	install -Dm644 LICENSE $(DESTDIR)$(PREFIX)/share/licenses/$(APP)/LICENSE 2>/dev/null || true

uninstall:
	rm -f $(DESTDIR)$(BINDIR)/$(APP)
	rm -rf $(DESTDIR)$(PREFIX)/share/doc/$(APP)

test:
	go test -count=1 ./...

test-race:
	go test -count=1 -race ./...

test-short:
	go test -count=1 -short ./...

vet:
	go vet ./...

fmt:
	gofmt -w ./cmd ./internal

clean:
	rm -f $(APP)
	rm -rf $(DIST)

# ── packaging ──────────────────────────────────────────────

package: tarball deb
	@echo "artifacts in $(DIST)/"

tarball: build
	@mkdir -p $(DIST)
	tar -czf $(DIST)/$(APP)-$(VERSION)-linux-$(ARCH).tar.gz \
		$(APP) README.md
	@echo "→ $(DIST)/$(APP)-$(VERSION)-linux-$(ARCH).tar.gz"

deb: build
	@bash scripts/build-deb.sh $(VERSION) $(ARCH)

aur-srcinfo:
	@command -v makepkg >/dev/null || { echo "makepkg required"; exit 1; }
	cd packaging/aur && makepkg --printsrcinfo > .SRCINFO
	@echo "→ packaging/aur/.SRCINFO"
	@tmpdir="$$(mktemp -d)" && cp packaging/aur/PKGBUILD-bin "$$tmpdir/PKGBUILD" && \
		(cd "$$tmpdir" && makepkg --printsrcinfo > "$(CURDIR)/packaging/aur/.SRCINFO-bin") && \
		rm -rf "$$tmpdir"
	@echo "→ packaging/aur/.SRCINFO-bin (copy into your minitone-bin AUR clone as .SRCINFO)"

# Push both AUR packages (needs your AUR SSH key; runs git push in ~/minitone*).
aur-push:
	@test -d $(HOME)/minitone || { echo "~/minitone AUR clone missing"; exit 1; }
	@test -d $(HOME)/minitone-bin || { echo "~/minitone-bin AUR clone missing"; exit 1; }
	cp packaging/aur/PKGBUILD packaging/aur/.SRCINFO packaging/aur/minitone.install $(HOME)/minitone/
	cp packaging/aur/PKGBUILD-bin $(HOME)/minitone-bin/PKGBUILD
	cp packaging/aur/.SRCINFO-bin $(HOME)/minitone-bin/.SRCINFO
	cp packaging/aur/minitone.install $(HOME)/minitone-bin/
	@echo "staged — review with: (cd $(HOME)/minitone && git status) (cd $(HOME)/minitone-bin && git status)"
	@echo "then: (cd $(HOME)/minitone && git add -A && git commit -m \"$(VERSION)\" && git push)"
	@echo "      (cd $(HOME)/minitone-bin && git add -A && git commit -m \"$(VERSION)\" && git push)"

release: clean test vet package
	@ls -lh $(DIST)/

# Render one UI state without touching mpv (welcome search playing queue
# favorites history library playlists details help video searching error compact narrow)
screenshot:
	@test -n "$(SCENARIO)" || { echo "usage: make screenshot SCENARIO=playing [W=100] [H=28] [THEME=tokyonight]"; exit 1; }
	@go run ./cmd/minitone --screenshot $(SCENARIO) $(or $(W),100) $(or $(H),28) $(or $(THEME),tokyonight)

help:
	@echo "targets: build install test test-race test-short vet fmt package tarball deb aur-srcinfo release clean screenshot"
