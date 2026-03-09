.PHONY: dev build build-all clean test helper

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
ENTITLEMENTS := build/darwin/entitlements.plist
HELPER_BIN := build/bin/clickwheel-helper

dev:
	wails dev

helper:
	go build -o $(HELPER_BIN) ./cmd/clickwheel-helper
	codesign --sign - --options runtime --entitlements $(ENTITLEMENTS) --force $(HELPER_BIN)

build: helper
	wails build $(LDFLAGS)
	cp $(HELPER_BIN) build/bin/clickwheel.app/Contents/MacOS/clickwheel-helper
	codesign --sign - --options runtime --entitlements $(ENTITLEMENTS) --force build/bin/clickwheel.app

build-all:
	wails build -platform darwin/amd64 $(LDFLAGS) -o clickwheel-darwin-amd64
	GOARCH=amd64 go build -o build/bin/clickwheel-helper-amd64 ./cmd/clickwheel-helper
	codesign --sign - --options runtime --entitlements $(ENTITLEMENTS) --force build/bin/clickwheel-helper-amd64
	cp build/bin/clickwheel-helper-amd64 build/bin/clickwheel-darwin-amd64.app/Contents/MacOS/clickwheel-helper
	codesign --sign - --options runtime --entitlements $(ENTITLEMENTS) --force build/bin/clickwheel-darwin-amd64.app
	wails build -platform darwin/arm64 $(LDFLAGS) -o clickwheel-darwin-arm64
	GOARCH=arm64 go build -o build/bin/clickwheel-helper-arm64 ./cmd/clickwheel-helper
	codesign --sign - --options runtime --entitlements $(ENTITLEMENTS) --force build/bin/clickwheel-helper-arm64
	cp build/bin/clickwheel-helper-arm64 build/bin/clickwheel-darwin-arm64.app/Contents/MacOS/clickwheel-helper
	codesign --sign - --options runtime --entitlements $(ENTITLEMENTS) --force build/bin/clickwheel-darwin-arm64.app
	wails build -platform linux/amd64 $(LDFLAGS) -o clickwheel-linux-amd64
	wails build -platform windows/amd64 $(LDFLAGS) -o clickwheel-windows-amd64.exe

build-darwin-universal: helper
	wails build -platform darwin/universal $(LDFLAGS)
	cp $(HELPER_BIN) build/bin/clickwheel.app/Contents/MacOS/clickwheel-helper
	codesign --sign - --options runtime --entitlements $(ENTITLEMENTS) --force build/bin/clickwheel.app

test:
	go test ./internal/...

clean:
	rm -rf build/bin
	rm -rf frontend/dist
