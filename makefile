GO = go
APP = mb-clob

# Docker targets
IMAGE_NAME ?= mb-clob
TAG ?= latest


build:
	@echo "Building $(APP)..."
	$(GO) build -o bin/$(APP) ./cmd/main.go

run: build
	@echo "Running $(APP)..."
	./bin/$(APP)

clean:
	@echo "Cleaning up..."
	rm -f bin/$(APP)

docker-build:
	@echo "Building Docker image..."
	docker build -t $(IMAGE_NAME):$(TAG) .

docker-run:
	@echo "Running Docker container..."
	docker run --rm -p 8080:8080 --name $(APP) $(IMAGE_NAME):$(TAG)

docker-run-bg:
	@echo "Running Docker container in background..."
	docker run -d -p 8080:8080 --name $(APP) $(IMAGE_NAME):$(TAG)

docker-stop:
	@echo "Stopping Docker container..."
	docker stop $(APP) || true
	docker rm $(APP) || true

docker-clean:
	@echo "Cleaning Docker resources..."
	docker stop $(APP) || true
	docker rm $(APP) || true
	docker rmi $(IMAGE_NAME):$(TAG) || true
	docker system prune -f

help:
	@echo "Available targets:"
	@echo ""
	@echo "Build & Run:"
	@echo "  build             - Build the application"
	@echo "  run               - Build and run the application"
	@echo "  clean             - Clean build artifacts and coverage files"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build      - Build Docker image"
	@echo "  docker-run        - Run Docker container"
	@echo "  docker-run-bg     - Run Docker container in background"
	@echo "  docker-stop       - Stop and remove Docker container"
	@echo "  docker-clean      - Clean all Docker resources"
	@echo ""
	@echo "  help              - Show this help message"

.PHONY: build run clean help docker-build docker-run docker-run-bg docker-stop docker-clean
.DEFAULT_GOAL := help