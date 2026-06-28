# Makefile for plus3.
#
# Thin wrappers over the Go toolchain and the existing build.sh / release.sh
# scripts, so there is a single source of truth for each operation. The version
# is read from the VERSION file and injected at build time, matching build.sh.

PROJECT_NAME := plus3
MODULE       := github.com/ha1tch/plus3
VERSION      := $(shell tr -d ' \t\r\n' < VERSION)
LDFLAGS      := -X $(MODULE)/internal/version.Version=$(VERSION)
BIN          := plus3
DISTDIR      := dist

# Cross-compile targets (mirrors .github/workflows/release.yml).
PLATFORMS := \
	linux/amd64 linux/arm64 \
	darwin/amd64 darwin/arm64 \
	windows/amd64 \
	freebsd/amd64 freebsd/arm64 \
	openbsd/amd64 openbsd/arm64 \
	netbsd/amd64

.DEFAULT_GOAL := build

# Build the CLI binary into the current directory, with the version injected.
.PHONY: build
build:
	@echo "Building $(PROJECT_NAME) $(VERSION)..."
	go build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd

# Run the CLI (default help output).
.PHONY: run
run: build
	./$(BIN)

# Run the CLI with arguments: make run-with-args ARGS='list disk.dsk'
.PHONY: run-with-args
run-with-args: build
	./$(BIN) $(ARGS)

# Run the test suite.
.PHONY: test
test:
	go test ./... -count=1

# Run go vet.
.PHONY: vet
vet:
	go vet ./...

# Vet and test together (the CI quality gate).
.PHONY: check
check: vet test

# Format all Go source (gofmt, matching the rest of the project).
.PHONY: fmt
fmt:
	gofmt -w .

# Verify module checksums (catches an incomplete go.sum).
.PHONY: verify
verify:
	go mod download
	go mod verify

# Tidy module dependencies.
.PHONY: tidy
tidy:
	go mod tidy

# Cross-compile binaries for all release platforms into dist/.
.PHONY: cross
cross:
	@mkdir -p $(DISTDIR)
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; arch=$${platform#*/}; \
		ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
		out="$(DISTDIR)/$(BIN)-$$os-$$arch$$ext"; \
		echo "  building $$out"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch \
			go build -ldflags "$(LDFLAGS)" -o "$$out" ./cmd || exit 1; \
	done

# Run the full release pipeline (validate -> build -> verify -> package).
.PHONY: release
release:
	sh release.sh $(VERSION)

# Remove build artefacts.
.PHONY: clean
clean:
	@echo "Cleaning up..."
	rm -f $(BIN)
	rm -rf $(DISTDIR)
	rm -f plus3-v*.zip

# Clean and tidy the environment.
.PHONY: reset
reset: clean tidy

# Print the current version.
.PHONY: version
version:
	@echo $(VERSION)

# Help.
.PHONY: help
help:
	@echo "Makefile for $(PROJECT_NAME)"
	@echo "Targets:"
	@echo "  build         - Build the binary (version injected) into ./$(BIN)"
	@echo "  run           - Build and run (help output)"
	@echo "  run-with-args - Build and run with ARGS='<args>'"
	@echo "  test          - Run the test suite"
	@echo "  vet           - Run go vet"
	@echo "  check         - Vet and test (CI quality gate)"
	@echo "  fmt           - Format source with gofmt"
	@echo "  verify        - Verify module checksums"
	@echo "  tidy          - Tidy module dependencies"
	@echo "  cross         - Cross-compile all release platforms into dist/"
	@echo "  release       - Run the full release pipeline"
	@echo "  clean         - Remove build artefacts"
	@echo "  reset         - Clean and tidy"
	@echo "  version       - Print the current version"
