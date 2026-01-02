BINARY_NAME=dualsense-mgr

build:
	go build -ldflags="-s -w" -o $(BINARY_NAME) .

compress:
	upx --best --lzma $(BINARY_NAME)

clean:
	rm -f $(BINARY_NAME)

install:
	sudo cp $(BINARY_NAME) /usr/local/bin/