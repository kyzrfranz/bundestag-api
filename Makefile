# generate all the make targets to build cmd/server.go
GOROOT=$(shell go env GOROOT)
GO=$(GOROOT)/bin/go


GOFLAGS = ''

PROJECT := buntesdach-api
BUILD_DIR = ./build
BUILD_TARGET = $(BUILD_DIR)/buntesdach-api
PUB_TARGET=$(PUB_DIR)/$(PROJECT)

dev:
	$(GO) run ./cmd/server.go

default: help

all: clean test build

version:
	echo $(VERSION)

.PHONY: help
## help: prints this help message
help:
	@echo "Usage: \n"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' |  sed -e 's/^/ /'


.PHONY: build
## build: builds for linux/mac on arm64 & arm
build: build-linux-amd64 build-linux-armv7 build-darwin-amd64 build-darwin-arm64

build-linux-amd64:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags $(GOFLAGS) -o $(BUILD_DIR)/$(PROJECT)-amd64-linux ./cmd/server.go

build-linux-armv7:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm $(GO) build -ldflags $(GOFLAGS) -o $(BUILD_DIR)/$(PROJECT)-armv7-linux ./cmd/server.go

build-darwin-amd64:
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build -ldflags $(GOFLAGS) -o $(BUILD_DIR)/$(PROJECT)-amd64-darwin ./cmd/server.go

build-darwin-arm64:
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 $(GO) build -ldflags $(GOFLAGS) -o $(BUILD_DIR)/$(PROJECT)-arm64-darwin ./cmd/server.go

.PHONY: clean
## clean: call Felix ;)
clean:
	$(GO) mod tidy
	rm -rf $(BUILD_DIR)/*

dockerize-local:
	docker buildx build --platform linux/amd64 -t eu.gcr.io/buntesdach/buntesdach-api:latest . 

dockerize:
	docker buildx build --platform linux/amd64 -t eu.gcr.io/buntesdach/buntesdach-api:latest . --push