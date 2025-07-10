# Claude Session Manager Makefile

# Variables
BINARY_NAME=claude-session-manager
BUILD_DIR=./build
CMD_DIR=./cmd
PKG_DIR=./internal/...
MAIN_FILE=$(CMD_DIR)/main.go
CONFIG_DIR=$(HOME)/.config/claude-session-manager

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet
SWAG=swag

# Build flags
LDFLAGS=-ldflags "-s -w"
BUILD_FLAGS=-trimpath

# Default target
.PHONY: all
all: clean build

# Generate swagger documentation
.PHONY: swagger
swagger:
	@echo "Generating swagger documentation..."
	@if command -v swag >/dev/null 2>&1; then \
		$(SWAG) init -g $(MAIN_FILE) -o ./docs; \
		echo "Swagger documentation generated in ./docs"; \
	else \
		echo "swag not installed. Install with: go install github.com/swaggo/swag/cmd/swag@latest"; \
	fi

# Generate Postman collection from Swagger
.PHONY: postman
postman: swagger
	@echo "Converting Swagger to Postman collection..."
	@if command -v openapi2postmanv2 >/dev/null 2>&1; then \
		openapi2postmanv2 -s ./docs/swagger.json -o ./postman_collection.json -p -O folderStrategy=Tags,requestParametersResolution=Example,includeAuthInfoInExample=false; \
		echo "Postman collection generated: ./postman_collection.json"; \
		echo "Note: URLs use {{baseUrl}} variable. Set it to: http://localhost:8080/api/v1"; \
	else \
		echo "openapi2postmanv2 not installed. Install with: npm install -g openapi-to-postmanv2"; \
		echo "Alternative: Use online converter at https://www.postman.com/api-platform/api-import/"; \
	fi

# Generate Postman collection with absolute URLs (no variables)
.PHONY: postman-fixed
postman-fixed: postman
	@echo "Fixing Postman collection URLs..."
	@if command -v node >/dev/null 2>&1; then \
		node fix-postman-urls.js; \
		echo "✅ Import postman_collection_fixed.json into Postman for working URLs"; \
	else \
		echo "Node.js not found. Please install Node.js to fix URLs automatically."; \
		echo "Alternatively, manually set baseUrl variable in Postman to: http://localhost:8080/api/v1"; \
	fi

# Build the application
.PHONY: build
build: swagger
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for multiple platforms
.PHONY: build-all
build-all: clean swagger
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	
	# Linux AMD64
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	
	# Linux ARM64
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)
	
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	
	# macOS ARM64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)
	
	# Windows AMD64
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)
	
	@echo "Multi-platform build complete!"
	@ls -la $(BUILD_DIR)/

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	$(GOMOD) tidy
	$(GOMOD) download
	@echo "Dependencies installed!"

# Run the application
.PHONY: run
run:
	@echo "Running $(BINARY_NAME)..."
	$(GOCMD) run $(CMD_DIR) serve --debug

# Run with build
.PHONY: run-build
run-build: build
	@echo "Running built binary..."
	$(BUILD_DIR)/$(BINARY_NAME) serve

# Test the application
.PHONY: test
test:
	@echo "Running tests..."
	$(GOTEST) -v $(PKG_DIR)

# Test with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out $(PKG_DIR)
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Test with race detection
.PHONY: test-race
test-race:
	@echo "Running tests with race detection..."
	$(GOTEST) -v -race $(PKG_DIR)

# Benchmark tests
.PHONY: bench
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem $(PKG_DIR)

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Vet code
.PHONY: vet
vet:
	@echo "Vetting code..."
	$(GOVET) $(PKG_DIR)

# Lint code (requires golangci-lint)
.PHONY: lint
lint:
	@echo "Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Security scan (requires gosec)
.PHONY: security
security:
	@echo "Running security scan..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not installed. Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	rm -f docs/docs.go docs/swagger.json docs/swagger.yaml
	rm -f postman_collection.json postman_collection_absolute.json postman_collection_fixed.json
	@echo "Clean complete!"

# Install the binary to GOPATH/bin
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME) to GOPATH/bin..."
	cp $(BUILD_DIR)/$(BINARY_NAME) $(shell $(GOCMD) env GOPATH)/bin/
	@echo "Installation complete! Run with: $(BINARY_NAME)"

# Install locally (to /usr/local/bin)
.PHONY: install-local
install-local: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "Installation complete! Run with: $(BINARY_NAME)"

# Uninstall from GOPATH/bin
.PHONY: uninstall
uninstall:
	@echo "Uninstalling $(BINARY_NAME) from GOPATH/bin..."
	rm -f $(shell $(GOCMD) env GOPATH)/bin/$(BINARY_NAME)
	@echo "Uninstallation complete!"

# Uninstall from /usr/local/bin
.PHONY: uninstall-local
uninstall-local:
	@echo "Uninstalling $(BINARY_NAME) from /usr/local/bin..."
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "Uninstallation complete!"

# Development workflow
.PHONY: dev
dev: deps fmt vet test swagger build run-build

# CI workflow
.PHONY: ci
ci: deps fmt vet lint test-race test-coverage security swagger build

# Pre-commit checks
.PHONY: pre-commit
pre-commit: fmt vet test swagger

# Create release package
.PHONY: release
release: clean build-all
	@echo "Creating release packages..."
	@mkdir -p $(BUILD_DIR)/release
	
	# Create tarballs for Unix systems
	cd $(BUILD_DIR) && tar -czf release/$(BINARY_NAME)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64
	cd $(BUILD_DIR) && tar -czf release/$(BINARY_NAME)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64
	cd $(BUILD_DIR) && tar -czf release/$(BINARY_NAME)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64
	cd $(BUILD_DIR) && tar -czf release/$(BINARY_NAME)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64
	
	# Create zip for Windows
	cd $(BUILD_DIR) && zip release/$(BINARY_NAME)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe
	
	@echo "Release packages created in $(BUILD_DIR)/release/"
	@ls -la $(BUILD_DIR)/release/

# Generate checksums for release
.PHONY: checksums
checksums: release
	@echo "Generating checksums..."
	cd $(BUILD_DIR)/release && shasum -a 256 * > checksums.txt
	@echo "Checksums generated: $(BUILD_DIR)/release/checksums.txt"

# Docker build (if Dockerfile exists)
.PHONY: docker-build
docker-build:
	@if [ -f Dockerfile ]; then \
		echo "Building Docker image..."; \
		docker build -t $(BINARY_NAME):latest .; \
	else \
		echo "No Dockerfile found"; \
	fi

# Configuration setup
.PHONY: config-init
config-init:
	@echo "Setting up configuration..."
	@mkdir -p $(CONFIG_DIR)
	@if [ ! -f $(CONFIG_DIR)/config.yaml ]; then \
		cp configs/example.yaml $(CONFIG_DIR)/config.yaml; \
		echo "Configuration file created at $(CONFIG_DIR)/config.yaml"; \
		echo "Please edit it to match your settings"; \
	else \
		echo "Configuration file already exists at $(CONFIG_DIR)/config.yaml"; \
	fi

# Show current configuration
.PHONY: config-show
config-show:
	@if [ -f $(CONFIG_DIR)/config.yaml ]; then \
		echo "Current configuration at $(CONFIG_DIR)/config.yaml:"; \
		echo "---"; \
		cat $(CONFIG_DIR)/config.yaml; \
	else \
		echo "No configuration file found at $(CONFIG_DIR)/config.yaml"; \
		echo "Run 'make config-init' to create one"; \
	fi

# Run with custom config
.PHONY: run-config
run-config: build
	@echo "Running with custom config..."
	$(BUILD_DIR)/$(BINARY_NAME) serve --config $(CONFIG_DIR)/config.yaml

# Show help
.PHONY: help
help:
	@echo "Claude Session Manager - Available Make targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  build          Build the application (includes swagger generation)"
	@echo "  build-all      Build for multiple platforms"
	@echo "  swagger        Generate swagger documentation"
	@echo "  postman        Generate Postman collection from swagger (uses variables)"
	@echo "  postman-fixed  Generate Postman collection with working absolute URLs"
	@echo "  clean          Clean build artifacts"
	@echo ""
	@echo "Development targets:"
	@echo "  run            Run the application directly"
	@echo "  run-build      Build and run the binary"
	@echo "  run-config     Build and run with custom config"
	@echo "  dev            Full development workflow (deps, format, vet, test, build, run)"
	@echo "  deps           Install dependencies"
	@echo ""
	@echo "Configuration targets:"
	@echo "  config-init    Initialize configuration file"
	@echo "  config-show    Show current configuration"
	@echo ""
	@echo "Testing targets:"
	@echo "  test           Run tests"
	@echo "  test-coverage  Run tests with coverage report"
	@echo "  test-race      Run tests with race detection"
	@echo "  bench          Run benchmark tests"
	@echo ""
	@echo "Code quality targets:"
	@echo "  fmt            Format code"
	@echo "  vet            Vet code"
	@echo "  lint           Lint code (requires golangci-lint)"
	@echo "  security       Run security scan (requires gosec)"
	@echo "  pre-commit     Run pre-commit checks"
	@echo ""
	@echo "Installation targets:"
	@echo "  install        Install to GOPATH/bin"
	@echo "  install-local  Install to /usr/local/bin"
	@echo "  uninstall      Uninstall from GOPATH/bin"
	@echo "  uninstall-local Uninstall from /usr/local/bin"
	@echo ""
	@echo "Release targets:"
	@echo "  release        Create release packages for all platforms"
	@echo "  checksums      Generate checksums for release packages"
	@echo ""
	@echo "CI targets:"
	@echo "  ci             Complete CI workflow"
	@echo ""
	@echo "Other targets:"
	@echo "  docker-build   Build Docker image (if Dockerfile exists)"
	@echo "  help           Show this help message"

# Default help when no target specified
.DEFAULT_GOAL := help
