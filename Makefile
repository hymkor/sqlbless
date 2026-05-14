ifeq ($(OS),Windows_NT)
    SHELL=CMD.EXE
    SET=set
    WHICH=where.exe
    DEL=del
    NUL=nul
else
    SET=export
    WHICH=which
    DEL=rm
    NUL=/dev/null
endif

GO=go
NAME:=$(notdir $(CURDIR))
VERSION:=$(shell git describe --tags 2>$(NUL) || echo v0.0.0)
GOOPT:=-ldflags "-s -w -X github.com/hymkor/$(NAME).Version=$(VERSION)"
EXE=$(shell $(GO) env GOEXE)

build:
	$(GO) fmt ./...
	$(SET) "CGO_ENABLED=0" && $(GO) build -tags debug $(GOOPT) "./cmd/$(NAME)"

test:
	$(GO) test ./...

_dist:
	$(SET) "CGO_ENABLED=0" && $(GO) build $(GOOPT) "./cmd/$(NAME)"
	zip -9 $(NAME)-$(VERSION)-$(GOOS)-$(GOARCH).zip $(NAME)$(EXE)

dist:
	$(SET) "GOOS=linux"   && $(SET) "GOARCH=386"   && $(MAKE) _dist
	$(SET) "GOOS=linux"   && $(SET) "GOARCH=amd64" && $(MAKE) _dist
	$(SET) "GOOS=windows" && $(SET) "GOARCH=386"   && $(MAKE) _dist
	$(SET) "GOOS=windows" && $(SET) "GOARCH=amd64" && $(MAKE) _dist

clean:
	$(DEL) *.zip $(NAME)$(EXE)

manifest:
	$(GO) run github.com/hymkor/make-scoop-manifest@master -all *-windows-*.zip > $(NAME).json

release:
	$(GO) run github.com/hymkor/latest-notes@latest | gh release create -d --notes-file - -t $(VERSION) $(VERSION) $(wildcard $(NAME)-$(VERSION)-*.zip)

bump:
	$(GO) run github.com/hymkor/latest-notes@latest -suffix "-goinstall" -gosrc $(NAME) CHANGELOG*.md > version.go

docs:
	$(GO) run github.com/hymkor/minipage@latest -outline-in-sidebar -readme-to-index README.md    > docs/index.html
	$(GO) run github.com/hymkor/minipage@latest -outline-in-sidebar -readme-to-index README_ja.md > docs/index_ja.html

readme:
	$(GO) run github.com/hymkor/example-into-readme@latest
	$(GO) run github.com/hymkor/example-into-readme@latest -target README_ja.md

get:
	$(GO) get -u
	$(GO) get github.com/mattn/go-tty@v0.0.7
	$(GO) get github.com/sijms/go-ora/v2@v2.8.22
	$(GO) mod tidy

.PHONY: test dist _dist clean manifest release docs bump drivers
