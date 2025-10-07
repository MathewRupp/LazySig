.PHONY: build install clean test help

# Binary name
BINARY=lazysig

# Build the binary
build:
	go build -o $(BINARY)

# Install to system
install: build
	sudo mv $(BINARY) /usr/local/bin/

# Install to user directory (no sudo required)
install-user: build
	mkdir -p ~/.local/bin
	mv $(BINARY) ~/.local/bin/
	@echo "Installed to ~/.local/bin/$(BINARY)"
	@echo "Make sure ~/.local/bin is in your PATH"

# Clean build artifacts
clean:
	rm -f $(BINARY)
	rm -f *.csv *.sr

# Run tests
test:
	go test ./...

# Run the application
run: build
	./$(BINARY)

# Show help
help:
	@echo "LazySig Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  build        - Build the binary"
	@echo "  install      - Build and install to /usr/local/bin (requires sudo)"
	@echo "  install-user - Build and install to ~/.local/bin (no sudo)"
	@echo "  clean        - Remove build artifacts"
	@echo "  run          - Build and run the application"
	@echo "  test         - Run tests"
	@echo "  help         - Show this help message"
