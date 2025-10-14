# Makefile for scaffold-dsl

BINARY=scaffold
CMD_DIR=./cmd/scaffold

.PHONY: all build tidy clean test

all: build

build:
	@echo "Building $(BINARY)..."
	go build -o $(BINARY) $(CMD_DIR)

tidy:
	@echo "Tidying dependencies..."
	go mod tidy

test:
	@echo "Running tests..."
	go test ./...

clean:
	@echo "Cleaning..."
	rm -f $(BINARY)
