BINARY_NAME := dots
BUILD_DIR := .
CMD_PATH := ./cmd/dots

.PHONY: build install clean run

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)

install: build
	install -m 755 $(BUILD_DIR)/$(BINARY_NAME) $(HOME)/.local/bin/$(BINARY_NAME)

clean:
	rm -f $(BUILD_DIR)/$(BINARY_NAME)

run: build
	./$(BINARY_NAME)

fmt:
	go fmt ./...

lint:
	go vet ./...

tidy:
	go mod tidy
