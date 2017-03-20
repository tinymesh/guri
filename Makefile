
FILES := $(shell find src/ -type f)

all: linux darwin windows

clean:
	rm dist/*

deps:
	go get golang.org/x/sys/windows
	go get go.bug.st/serial.v1

linux: dist/guri-linux-amd64 dist/guri-linux-386 dist/guri-linux-arm dist/guri-linux-arm64
darwin: dist/guri-darwin-amd64 dist/guri-darwin-386 # dist/guri-darwin-arm dist/guri-darwin-arm64
windows: dist/guri-windows-amd64.exe dist/guri-windows-386.exe

dist/guri-linux-amd64: GOOS  = linux
dist/guri-linux-amd64: GOARCH = amd64
dist/guri-linux-386: GOOS = linux
dist/guri-linux-386: GOARCH = 386
dist/guri-linux-arm: GOOS = linux
dist/guri-linux-arm: GOARCH = arm
dist/guri-linux-arm64: GOOS = linux
dist/guri-linux-arm64: GOARCH = arm64
dist/guri-darwin-amd64: GOOS = darwin
dist/guri-darwin-amd64: GOARCH = amd64
dist/guri-darwin-386: GOOS = darwin
dist/guri-darwin-386: GOARCH = 386
dist/guri-darwin-arm: GOOS = darwin
dist/guri-darwin-arm: GOARCH = arm
dist/guri-darwin-arm64: GOOS = darwin
dist/guri-darwin-arm64: GOARCH = arm64
dist/guri-windows-amd64.exe: GOOS = windows
dist/guri-windows-amd64.exe: GOARCH = amd64
dist/guri-windows-386.exe: GOOS = windows
dist/guri-windows-386.exe: GOARCH = 386
dist/guri-%: $(FILES)
	cd src/; GOARCH=$(GOARCH) GOOS=$(GOOS) go build -o ../$@
