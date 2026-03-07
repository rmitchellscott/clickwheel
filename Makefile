.PHONY: dev build build-all clean test

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

dev:
	wails dev

build:
	wails build $(LDFLAGS)

build-all:
	wails build -platform darwin/amd64 $(LDFLAGS) -o clickwheel-darwin-amd64
	wails build -platform darwin/arm64 $(LDFLAGS) -o clickwheel-darwin-arm64
	wails build -platform linux/amd64 $(LDFLAGS) -o clickwheel-linux-amd64
	wails build -platform windows/amd64 $(LDFLAGS) -o clickwheel-windows-amd64.exe

build-darwin-universal:
	wails build -platform darwin/universal $(LDFLAGS)

test:
	go test ./internal/...

clean:
	rm -rf build/bin
	rm -rf frontend/dist
