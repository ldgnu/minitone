APP       := minitone
VERSION   := 0.2.0
PREFIX    := /usr
BINDIR    := $(PREFIX)/bin
GOFLAGS   := -trimpath
LDFLAGS   := -s -w -X github.com/ldgnu/minitone/internal/app.Version=$(VERSION)
DIST      := dist
ARCH      := $(shell uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')

.PHONY: all build install uninstall test vet clean \
	package tarball deb aur-srcinfo release help

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

test-short:
	go test -count=1 -short ./...

vet:
	go vet ./...

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

release: clean test vet package
	@ls -lh $(DIST)/

help:
	@echo "targets: build install test package tarball deb aur-srcinfo release clean"
