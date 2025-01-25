# Makefile for the +3DOS CLI project

# Variables
PROJECT_NAME = plus3
OUTPUT_BIN = $(HOME)/bin/$(PROJECT_NAME)
MAIN_FILE = ./cmd/main.go

# Default target
all: build

# Build the CLI binary
build:
	@echo "Building $(PROJECT_NAME)..."
	mkdir -p $(HOME)/bin
	go build -o $(OUTPUT_BIN) $(MAIN_FILE)

# Run the CLI (default help output)
run:
	@echo "Running $(PROJECT_NAME)..."
	$(OUTPUT_BIN)

# Run the CLI with arguments
run-with-args:
	@echo "Running $(PROJECT_NAME) with arguments: $(ARGS)"
	$(OUTPUT_BIN) $(ARGS)

# Clean up the build
clean:
	@echo "Cleaning up..."
	rm -f $(OUTPUT_BIN)

# Run go fmt on all source files
fmt:
	@echo "Formatting source files..."
	go fmt ./...

# Run go vet to check for issues
vet:
	@echo "Running go vet..."
	go vet ./...

# Run tests
.PHONY: test

test:
	@echo "Running tests..."
	go test ./...

# Tidy dependencies
tidy:
	@echo "Tidying up dependencies..."
	go mod tidy

# Ensure clean environment
reset: clean tidy

# Help
help:
	@echo "Makefile for $(PROJECT_NAME)"
	@echo "Targets:"
	@echo "  all          - Build the project"
	@echo "  build        - Build the project"
	@echo "  run          - Run the project"
	@echo "  run-with-args ARGS='<args>' - Run the project with custom arguments"
	@echo "  clean        - Remove the built binary"
	@echo "  fmt          - Format the source files"
	@echo "  vet          - Check for issues using go vet"
	@echo "  test         - Run all tests"
	@echo "  tidy         - Clean up module dependencies"
	@echo "  reset        - Clean and tidy the environment"
