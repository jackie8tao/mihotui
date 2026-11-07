NAME       := mihomo-tui
VERSION    := v0.1.0
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || echo "unknown")
LDFLAGS    := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE)

.PHONY: all test build-linux package clean

all: test build-linux

test:
	go test -v ./...

build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(NAME)-linux-amd64 ./cmd/$(NAME)
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(NAME)-linux-arm64 ./cmd/$(NAME)

package:
	go run ./scripts/package.go -v $(VERSION)

clean:
	rm -rf dist/ build/ $(NAME)-linux-* $(NAME).exe
