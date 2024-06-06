ifeq ($(OS),Windows_NT)
    SHELL=CMD.EXE
    SET=set
    DEL=del
    NUL=nul
else
    SET=export
    DEL=rm
    NUL=/dev/null
endif

NAME:=$(notdir $(CURDIR))
VERSION:=$(shell git describe --tags 2>$(NUL) || echo v0.0.0)
GOOPT:=-ldflags "-s -w -X main.version=$(VERSION)"
EXE=$(shell go env GOEXE)

all:
	go fmt
ifeq ($(shell go env GOOS)-$(shell go env GOARCH),windows-386)
	$(SET) "CGO_ENABLED=1" && go build $(GOOPT)
else
	$(SET) "CGO_ENABLED=0" && go build $(GOOPT)
endif

test:
ifeq ($(OS),Windows_NT)
	pwsh "test/test-sqlite3.ps1"
	pwsh "test/test.ps1"
endif
	go test -v

_dist:
	go build $(GOOPT)
	zip -9 $(NAME)-$(VERSION)-$(GOOS)-$(GOARCH).zip $(NAME)$(EXE)

dist:
	$(SET) "CGO_ENABLED=0" && $(SET) "GOOS=linux" && $(SET) "GOARCH=386"   && $(MAKE) _dist
	$(SET) "CGO_ENABLED=0" && $(SET) "GOOS=linux" && $(SET) "GOARCH=amd64" && $(MAKE) _dist
	$(SET) "CGO_ENABLED=1" && $(SET) "GOOS=windows" && $(SET) "GOARCH=386"   && $(MAKE) _dist
	$(SET) "CGO_ENABLED=0" && $(SET) "GOOS=windows" && $(SET) "GOARCH=amd64" && $(MAKE) _dist

clean:
	$(DEL) *.zip $(NAME)$(EXE)

manifest:
	make-scoop-manifest *-windows-*.zip > $(NAME).json

release:
	gh release create -d --notes "" -t $(VERSION) $(VERSION) $(wildcard $(NAME)-$(VERSION)-*.zip)

.PHONY: all test dist _dist clean manifest release
