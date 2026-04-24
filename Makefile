# Makefile for Terraform Provider Scality

VERSION ?= dev
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

default: build

# Build the provider
build:
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o terraform-provider-scality

# Install the provider locally for development
install: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/scality/scality/$(VERSION)/$(GOOS)_$(GOARCH)/
	cp terraform-provider-scality ~/.terraform.d/plugins/registry.terraform.io/scality/scality/$(VERSION)/$(GOOS)_$(GOARCH)/

# Run tests
test:
	go test ./... -v

# Run acceptance tests
testacc:
	TF_ACC=1 go test ./... -v -count 1 -timeout 120m

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Generate documentation
docs:
	go generate ./...

# Clean build artifacts
clean:
	rm -f terraform-provider-scality

# Initialize Go modules
init:
	go mod download
	go mod tidy

.PHONY: build install test testacc fmt lint docs clean init
