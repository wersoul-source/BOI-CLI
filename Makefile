.PHONY: build run test clean

BINARY_NAME=boi
BUILD_DIR=bin

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/boi

run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

test:
	go test ./...

clean:
	rm -rf $(BUILD_DIR)

install: build
	go install ./cmd/boi

lint:
	go vet ./...
