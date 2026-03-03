# Makefile for Terraform Provider Scality

default: build

# Build the provider
build:
	go build -o terraform-provider-scality

# Install the provider locally for development
install: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/scality/scality/0.1.0/linux_amd64/
	cp terraform-provider-scality ~/.terraform.d/plugins/registry.terraform.io/scality/scality/0.1.0/linux_amd64/

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
