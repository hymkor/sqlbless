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
GOOPT:=-ldflags "-s -w -X github.com/hymkor/sqlbless.Version=$(VERSION)"
EXE=$(shell go env GOEXE)

all:
	go fmt ./...
	$(SET) "CGO_ENABLED=0" && go build $(GOOPT) && go build -C "$(CURDIR)/cmd/sqlbless" -o "$(CURDIR)/$(NAME)$(EXE)" $(GOOPT)

test:
ifeq ($(OS),Windows_NT)
	pwsh "test/test-sqlite3.ps1"
	pwsh "test/test.ps1"
endif
	go test -v

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
	gh release create -d --notes "" -t $(VERSION) $(VERSION) $(wildcard $(NAME)-$(VERSION)-*.zip)

get :
	cd "$(CURDIR)/cmd/sqlbless" && go get -u && go mod tidy

.PHONY: all test dist _dist clean manifest release
