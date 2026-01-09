.PHONY: build clean all linux windows darwin

VERSION := 1.0.0
BINARY := nexus-agent
DIST := dist

# Build for current platform
build:
	CGO_ENABLED=1 go build -o $(BINARY) ./cmd/agent

# Build all platforms
all: clean linux windows darwin

# Linux AMD64
linux:
	mkdir -p $(DIST)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(DIST)/$(BINARY)-linux-amd64 ./cmd/agent

# Windows AMD64
windows:
	mkdir -p $(DIST)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o $(DIST)/$(BINARY)-windows-amd64.exe ./cmd/agent

# macOS AMD64
darwin:
	mkdir -p $(DIST)
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o $(DIST)/$(BINARY)-darwin-amd64 ./cmd/agent

# Clean build artifacts
clean:
	rm -rf $(DIST)
	rm -f $(BINARY) $(BINARY).exe

# Run tests
test:
	go test -v ./...

# Install dependencies
deps:
	go mod download
	go mod tidy
