.PHONY: build build-linux-amd64 build-linux-arm64 build-all clean install test

BINARY := autogdb
BIN_DIR := ../bin

build:
	go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY) ./cmd/autogdb

build-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY)-Linux-amd64 ./cmd/autogdb

build-linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY)-Linux-arm64 ./cmd/autogdb

build-all: build-linux-amd64 build-linux-arm64

clean:
	rm -f $(BIN_DIR)/$(BINARY) $(BIN_DIR)/$(BINARY)-Linux-*

test:
	go test ./...