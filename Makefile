BINARY_NAME=dualsense-mgr
VERSION=$(shell git describe --tags --always --dirty)

build:
	go build -ldflags="-s -w -X main.Version=$(VERSION)" -o $(BINARY_NAME) .

compress:
	upx --best --lzma $(BINARY_NAME)

clean:
	rm -f $(BINARY_NAME)

install:
	sudo cp $(BINARY_NAME) /usr/local/bin/

lint:
	golangci-lint run ./...