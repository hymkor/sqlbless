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

ifndef GO
    SUPPORTGO=go1.20.14
    GO:=$(shell $(WHICH) $(SUPPORTGO) 2>$(NUL) || echo go)
endif

NAME:=$(notdir $(CURDIR))
VERSION:=$(shell git describe --tags 2>$(NUL) || echo v0.0.0)
GOOPT:=-ldflags "-s -w -X github.com/hymkor/sqlbless.Version=$(VERSION)"
EXE=$(shell $(GO) env GOEXE)

build:
	$(GO) fmt ./...
	$(SET) "CGO_ENABLED=0" && $(GO) build -C "$(CURDIR)/cmd/sqlbless" -o "$(CURDIR)" $(GOOPT)

test:
	$(GO) test -v ./...

_dist:
	$(SET) "CGO_ENABLED=0" && $(GO) build -C "$(CURDIR)/cmd/sqlbless" -o "$(CURDIR)" $(GOOPT)
	zip -9 $(NAME)-$(VERSION)-$(GOOS)-$(GOARCH).zip $(NAME)$(EXE)

dist:
	$(SET) "GOOS=linux"   && $(SET) "GOARCH=386"   && $(MAKE) _dist
	$(SET) "GOOS=linux"   && $(SET) "GOARCH=amd64" && $(MAKE) _dist
	$(SET) "GOOS=windows" && $(SET) "GOARCH=386"   && $(MAKE) _dist
	$(SET) "GOOS=windows" && $(SET) "GOARCH=amd64" && $(MAKE) _dist

clean:
	$(DEL) *.zip $(NAME)$(EXE)

manifest:
	make-scoop-manifest *-windows-*.zip > $(NAME).json

release:
	$(GO) run github.com/hymkor/latest-notes@master | gh release create -d --notes-file - -t $(VERSION) $(VERSION) $(wildcard $(NAME)-$(VERSION)-*.zip)

docs:
	minipage -outline-in-sidebar -readme-to-index README.md    > docs/index.html
	minipage -outline-in-sidebar -readme-to-index README_ja.md > docs/index_ja.html

.PHONY: all test dist _dist clean manifest release docs
