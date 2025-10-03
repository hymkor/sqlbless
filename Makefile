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

all:
	$(GO) fmt ./...
	$(SET) "CGO_ENABLED=0" && $(GO) build $(GOOPT) && $(GO) build -C "$(CURDIR)/cmd/sqlbless" -o "$(CURDIR)/$(NAME)$(EXE)" $(GOOPT)

test:
ifeq ($(OS),Windows_NT)
	pwsh "test/test-sqlite3.ps1"
	pwsh "test/test.ps1"
endif
	$(GO) test -v

_dist:
	$(MAKE) all
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
	goawk -f latest-notes.awk release_note*.md | gh release create -d --notes-file - -t $(VERSION) $(VERSION) $(wildcard $(NAME)-$(VERSION)-*.zip)

get:
	$(GO) get -u
	$(GO) get golang.org/x/sys@v0.30.0
	$(GO) get golang.org/x/text@v0.22.0
	$(GO) get golang.org/x/term@v0.29.0 
	$(GO) get golang.org/x/exp@v0.0.0-20240531132922-fd00a4e0eefc
	$(GO) mod tidy
# cd "$(CURDIR)/cmd/sqlbless" && $(GO) get -u && $(GO) mod tidy

docs:
	minipage -outline-in-sidebar -readme-to-index README.md    > docs/index.html
	minipage -outline-in-sidebar -readme-to-index README_ja.md > docs/index_ja.html

.PHONY: all test dist _dist clean manifest release docs
