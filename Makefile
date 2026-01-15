VERSION=$(shell git describe --tags)
CMD_PATH := ./cmd/veild/
BIN_NAME := veild
ADDITIONAL_FILES := README.md LICENSE veild
FLAGS := -trimpath -ldflags "-X main.veilVersion=$(VERSION)"
OUTPUT_DIR := dist

all: linux-arm linux-amd64 linux-arm64 darwin-amd64 darwin-arm64

linux-arm:
	GOOS=linux GOARCH=arm GOARM=7 go build $(FLAGS) \
	 -o veild $(CMD_PATH)
	tar -cvzf $(OUTPUT_DIR)/$(BIN_NAME)_$(VERSION)_linux_arm.tar.gz $(ADDITIONAL_FILES)
	shasum -a 256 $(OUTPUT_DIR)/$(BIN_NAME)_$(VERSION)_linux_arm.tar.gz > $(OUTPUT_DIR)/$(BIN_NAME)_$(VERSION)_linux_arm.tar.gz.asc

linux-amd64:
	GOOS=linux GOARCH=amd64 go build $(FLAGS) \
	 -o veild $(CMD_PATH)
	tar -cvzf $(OUTPUT_DIR)/$(BIN_NAME)_$(VERSION)_linux_amd64.tar.gz $(ADDITIONAL_FILES)
	shasum -a 256 $(OUTPUT_DIR)/$(BIN_NAME)_$(VERSION)_linux_amd64.tar.gz > $(OUTPUT_DIR)/$(BIN_NAME)_$(VERSION)_linux_amd64.tar.gz.asc

linux-arm64:
	GOOS=linux GOARCH=arm64 go build $(FLAGS) \
	 -o veild $(CMD_PATH)
	tar -cvzf $(OUTPUT_DIR)/$(BIN_NAME)_$(VERSION)_linux_arm64.tar.gz $(ADDITIONAL_FILES)
	shasum -a 256 $(OUTPUT_DIR)/$(BIN_NAME)_$(VERSION)_linux_arm64.tar.gz > $(OUTPUT_DIR)/$(BIN_NAME)_$(VERSION)_linux_arm64.tar.gz.asc

darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build $(FLAGS) \
	 -o veild $(CMD_PATH)
	tar -cvzf $(OUTPUT_DIR)/$(BIN_NAME)_$(VERSION)_darwin_amd64.tar.gz $(ADDITIONAL_FILES)
	shasum -a 256 $(OUTPUT_DIR)/$(BIN_NAME)_$(VERSION)_darwin_amd64.tar.gz > $(OUTPUT_DIR)/$(BIN_NAME)_$(VERSION)_darwin_amd64.tar.gz.asc

darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build $(FLAGS) \
	 -o veild $(CMD_PATH)
	tar -cvzf $(OUTPUT_DIR)/$(BIN_NAME)_$(VERSION)_darwin_arm64.tar.gz $(ADDITIONAL_FILES)
	shasum -a 256 $(OUTPUT_DIR)/$(BIN_NAME)_$(VERSION)_darwin_arm64.tar.gz > $(OUTPUT_DIR)/$(BIN_NAME)_$(VERSION)_darwin_arm64.tar.gz.asc

clean:
	rm -rf veild $(OUTPUT_DIR)/*

.PHONY: all clean linux-arm linux-amd64 linux-arm64 darwin-amd64 darwin-arm64
