VERSION=$(shell git describe --tags)
CMD_PATH := ./cmd/veild/
BIN_NAME := veild
ADDITIONAL_FILES := README.md LICENSE veild

OUTPUT = $(BIN_NAME)_$(VERSION)_$(GOOS)_$(GOARCH)

all: linux-arm linux-amd64 linux-arm64 darwin-amd64 darwin-arm64

linux-arm:
	GOOS=linux GOARCH=arm GOARM=7 go build -ldflags "-X main.veilVersion=$(VERSION)" \
	 -o veild $(CMD_PATH)
	tar -cvzf $(BIN_NAME)_$(VERSION)_linux_arm.tar.gz $(ADDITIONAL_FILES)
	shasum -a 256 $(BIN_NAME)_$(VERSION)_linux_arm.tar.gz > $(BIN_NAME)_$(VERSION)_linux_arm.tar.gz.asc

linux-amd64:
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.veilVersion=$(VERSION)" \
	 -o veild $(CMD_PATH)
	tar -cvzf $(BIN_NAME)_$(VERSION)_linux_amd64.tar.gz $(ADDITIONAL_FILES)
	shasum -a 256 $(BIN_NAME)_$(VERSION)_linux_amd64.tar.gz > $(BIN_NAME)_$(VERSION)_linux_amd64.tar.gz.asc

linux-arm64:
	GOOS=linux GOARCH=arm64 go build -ldflags "-X main.veilVersion=$(VERSION)" \
	 -o veild $(CMD_PATH)
	tar -cvzf $(BIN_NAME)_$(VERSION)_linux_arm64.tar.gz $(ADDITIONAL_FILES)
	shasum -a 256 $(BIN_NAME)_$(VERSION)_linux_arm64.tar.gz > $(BIN_NAME)_$(VERSION)_linux_arm64.tar.gz.asc

darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.veilVersion=$(VERSION)" \
	 -o veild $(CMD_PATH)
	zip $(BIN_NAME)_$(VERSION)_darwin_amd64.zip $(ADDITIONAL_FILES)
	shasum -a 256 $(BIN_NAME)_$(VERSION)_darwin_amd64.zip > $(BIN_NAME)_$(VERSION)_darwin_amd64.zip.asc

darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.veilVersion=$(VERSION)" \
	 -o veild $(CMD_PATH)
	zip $(BIN_NAME)_$(VERSION)_darwin_arm64.zip $(ADDITIONAL_FILES)
	shasum -a 256 $(BIN_NAME)_$(VERSION)_darwin_arm64.zip > $(BIN_NAME)_$(VERSION)_darwin_arm64.zip.asc

clean:
	rm veild veild_v*

.PHONY: all clean linux-arm linux-amd64 linux-arm64 darwin-amd64 darwin-arm64
