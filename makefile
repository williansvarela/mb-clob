GO = go
APP = mb-clob

build:
	@echo "Building $(APP)..."
	$(GO) build -o bin/$(APP) ./cmd/main.go

run: build
	@echo "Running $(APP)..."
	./bin/$(APP)

clean:
	@echo "Cleaning up..."
	rm -f bin/$(APP)

.PHONY: build run clean
.DEFAULT_GOAL := build