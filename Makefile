.PHONY: build run test lint clean

BINARY := macbroom
BUILD_DIR := ./bin

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/macbroom

run: build
	$(BUILD_DIR)/$(BINARY)

test:
	go test -race -cover ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BUILD_DIR)
