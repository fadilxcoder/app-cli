BINARY      := myapp
PKG         := github.com/fadilxcoder/app-cli
CMD         := ./cmd/myapp
DIST        := dist
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS     := -s -w -X main.Version=$(VERSION)
GO          ?= go

.PHONY: all tidy fmt vet test build build-linux build-mac build-mac-arm release clean install

all: build

tidy:
	$(GO) mod tidy

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

test:
	$(GO) test ./...

build:
	mkdir -p $(DIST)
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags '$(LDFLAGS)' -o $(DIST)/$(BINARY) $(CMD)

build-linux:
	mkdir -p $(DIST)
	CGO_ENABLED=0 GOOS=linux  GOARCH=amd64 $(GO) build -trimpath -ldflags '$(LDFLAGS)' -o $(DIST)/$(BINARY)-linux-amd64  $(CMD)
	CGO_ENABLED=0 GOOS=linux  GOARCH=arm64 $(GO) build -trimpath -ldflags '$(LDFLAGS)' -o $(DIST)/$(BINARY)-linux-arm64  $(CMD)

build-mac:
	mkdir -p $(DIST)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build -trimpath -ldflags '$(LDFLAGS)' -o $(DIST)/$(BINARY)-darwin-amd64 $(CMD)

build-mac-arm:
	mkdir -p $(DIST)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build -trimpath -ldflags '$(LDFLAGS)' -o $(DIST)/$(BINARY)-darwin-arm64 $(CMD)

# Build all release artifacts (linux + darwin, amd64 + arm64).
release: clean build-linux build-mac build-mac-arm
	cd $(DIST) && (command -v sha256sum >/dev/null && sha256sum $(BINARY)-* || shasum -a 256 $(BINARY)-*) > SHA256SUMS

install: build
	install -m 0755 $(DIST)/$(BINARY) /usr/local/bin/$(BINARY)

clean:
	rm -rf $(DIST)
