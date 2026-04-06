SHELL := /bin/bash

BINARY := utilivos
MODULE := utilivos/sdk
VERSION ?= 0.1.8
PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin

.PHONY: all build test lint run-help install uninstall clean clean-bin deb release

all: build

build:
	@mkdir -p bin
	go build -trimpath -ldflags "-s -w -X $(MODULE)/internal/util.Version=$(VERSION)" -o bin/$(BINARY) ./cmd/utilivos

test:
	go test ./...

lint:
	go vet ./...

run-help:
	go run ./cmd/utilivos --help

install:
	@test -x bin/$(BINARY) || (echo "missing bin/$(BINARY); run 'make build' first" >&2; exit 1)
	install -d $(DESTDIR)$(BINDIR)
	install -m 0755 bin/$(BINARY) $(DESTDIR)$(BINDIR)/$(BINARY)
	ln -sf $(BINARY) $(DESTDIR)$(BINDIR)/create
	ln -sf $(BINARY) $(DESTDIR)$(BINDIR)/verify
	ln -sf $(BINARY) $(DESTDIR)$(BINDIR)/test
	ln -sf $(BINARY) $(DESTDIR)$(BINDIR)/run
	ln -sf $(BINARY) $(DESTDIR)$(BINDIR)/build

uninstall:
	rm -f $(DESTDIR)$(BINDIR)/$(BINARY)
	rm -f $(DESTDIR)$(BINDIR)/create
	rm -f $(DESTDIR)$(BINDIR)/verify
	rm -f $(DESTDIR)$(BINDIR)/test
	rm -f $(DESTDIR)$(BINDIR)/run
	rm -f $(DESTDIR)$(BINDIR)/build

clean-bin:
	rm -rf bin

clean:
	rm -rf bin dist .pkg

deb: build
	VERSION=$(VERSION) ./scripts/package_deb.sh

release: test build deb
