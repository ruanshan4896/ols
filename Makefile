BINARY_NAME=ols-cli
BUILD_DIR=bin

.PHONY: all build build-linux test clean

all: test build

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) main.go

build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 main.go
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 main.go

test:
	go test -v ./...

clean:
	rm -rf $(BUILD_DIR)
