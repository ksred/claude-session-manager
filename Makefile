# Claude Session Manager Makefile

# Docker Hub configuration
DOCKER_HUB_USER ?= ksred
IMAGE_NAME = claude-session-manager
FULL_IMAGE_NAME = $(DOCKER_HUB_USER)/$(IMAGE_NAME)

# Version management
VERSION ?= latest
GIT_COMMIT = $(shell git rev-parse --short HEAD)
BUILD_DATE = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Default target
.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Docker commands
.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "Building Docker image: $(FULL_IMAGE_NAME):$(VERSION)"
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(FULL_IMAGE_NAME):$(VERSION) \
		-t $(FULL_IMAGE_NAME):latest \
		.

.PHONY: docker-push
docker-push: ## Push Docker image to Docker Hub
	@echo "Pushing Docker image to Docker Hub: $(FULL_IMAGE_NAME):$(VERSION)"
	docker push $(FULL_IMAGE_NAME):$(VERSION)
	docker push $(FULL_IMAGE_NAME):latest

.PHONY: docker-build-push
docker-build-push: docker-build docker-push ## Build and push Docker image

.PHONY: docker-run
docker-run: ## Run Docker container locally
	@echo "Running Docker container: $(FULL_IMAGE_NAME):$(VERSION)"
	docker run -d \
		--name $(IMAGE_NAME) \
		-p 80:80 \
		-v ~/.claude:/data/.claude:rw \
		$(FULL_IMAGE_NAME):$(VERSION)
	@echo "Container started. Access the application at http://localhost"

.PHONY: docker-stop
docker-stop: ## Stop and remove Docker container
	@echo "Stopping Docker container: $(IMAGE_NAME)"
	-docker stop $(IMAGE_NAME)
	-docker rm $(IMAGE_NAME)

.PHONY: docker-logs
docker-logs: ## Show Docker container logs
	docker logs -f $(IMAGE_NAME)

.PHONY: docker-shell
docker-shell: ## Shell into running Docker container
	docker exec -it $(IMAGE_NAME) /bin/sh

# Docker Compose commands
.PHONY: compose-up
compose-up: ## Start services with docker-compose
	docker-compose up -d

.PHONY: compose-down
compose-down: ## Stop services with docker-compose
	docker-compose down

.PHONY: compose-logs
compose-logs: ## Show docker-compose logs
	docker-compose logs -f

# Development commands
.PHONY: dev-backend
dev-backend: ## Run backend in development mode
	cd backend && make run

.PHONY: dev-frontend
dev-frontend: ## Run frontend in development mode
	cd frontend && npm run dev

.PHONY: dev
dev: ## Run both frontend and backend in development mode (requires two terminals)
	@echo "Starting backend and frontend in development mode..."
	@echo "Run 'make dev-backend' in one terminal"
	@echo "Run 'make dev-frontend' in another terminal"

# Testing commands
.PHONY: test-backend
test-backend: ## Run backend tests
	cd backend && make test

.PHONY: test-frontend
test-frontend: ## Run frontend tests
	cd frontend && npm test

# Build commands
.PHONY: build-backend
build-backend: ## Build backend binary
	cd backend && make build

.PHONY: build-frontend
build-frontend: ## Build frontend assets
	cd frontend && npm run build

# Versioning commands
.PHONY: version
version: ## Show current version
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"

.PHONY: tag
tag: ## Create and push a git tag
	@if [ -z "$(VERSION)" ] || [ "$(VERSION)" = "latest" ]; then \
		echo "Error: VERSION must be set and not 'latest'"; \
		echo "Usage: make tag VERSION=1.0.0"; \
		exit 1; \
	fi
	git tag -a v$(VERSION) -m "Release version $(VERSION)"
	git push origin v$(VERSION)

# Release command
.PHONY: release
release: ## Create a new release (build, tag, and push)
	@if [ -z "$(VERSION)" ] || [ "$(VERSION)" = "latest" ]; then \
		echo "Error: VERSION must be set and not 'latest'"; \
		echo "Usage: make release VERSION=1.0.0"; \
		exit 1; \
	fi
	@echo "Creating release $(VERSION)..."
	$(MAKE) docker-build VERSION=$(VERSION)
	$(MAKE) tag VERSION=$(VERSION)
	$(MAKE) docker-push VERSION=$(VERSION)
	@echo "Release $(VERSION) complete!"

# Clean commands
.PHONY: clean
clean: ## Clean build artifacts
	cd backend && make clean
	cd frontend && rm -rf dist node_modules
	docker rmi $(FULL_IMAGE_NAME):$(VERSION) || true
	docker rmi $(FULL_IMAGE_NAME):latest || true

# Docker Hub login
.PHONY: docker-login
docker-login: ## Login to Docker Hub
	@echo "Logging in to Docker Hub..."
	@docker login

# Multi-arch build (for ARM and x86)
.PHONY: docker-buildx
docker-buildx: ## Build multi-architecture Docker image
	@echo "Building multi-arch Docker image: $(FULL_IMAGE_NAME):$(VERSION)"
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(FULL_IMAGE_NAME):$(VERSION) \
		-t $(FULL_IMAGE_NAME):latest \
		--push \
		.

# Setup buildx for multi-arch builds
.PHONY: setup-buildx
setup-buildx: ## Setup Docker buildx for multi-architecture builds
	docker buildx create --name multiarch --use || true
	docker buildx inspect --bootstrap
