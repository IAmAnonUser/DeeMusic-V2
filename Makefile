# DeeMusic Build Makefile
# Cross-platform build automation

.PHONY: all build clean test install help version dll wpf installer portable release

# Default target
all: build

# Get version from VERSION file
VERSION := $(shell cat VERSION 2>/dev/null || echo "2.0.0")

# Build configuration
CONFIG ?= Release

help:
	@echo "DeeMusic Build System"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all         - Build application (default)"
	@echo "  build       - Build Go DLL and C# WPF"
	@echo "  dll         - Build Go DLL only"
	@echo "  wpf         - Build C# WPF only"
	@echo "  installer   - Build NSIS installer"
	@echo "  portable    - Build portable distribution"
	@echo "  release     - Build complete release packages"
	@echo "  test        - Run all tests"
	@echo "  clean       - Clean build artifacts"
	@echo "  version     - Show current version"
	@echo "  help        - Show this help message"
	@echo ""
	@echo "Variables:"
	@echo "  CONFIG      - Build configuration (Debug/Release, default: Release)"
	@echo "  VERSION     - Version override (default: from VERSION file)"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  make build CONFIG=Debug"
	@echo "  make release VERSION=2.1.0"
	@echo "  make test"

version:
	@echo "Current version: $(VERSION)"

dll:
	@echo "Building Go DLL..."
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File scripts/build-dll.ps1 -Release:$$([bool]"$(CONFIG)" -eq "Release") -Version "$(VERSION)"
else
	@echo "Error: DLL build is only supported on Windows"
	@exit 1
endif

wpf:
	@echo "Building C# WPF..."
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File scripts/build-wpf.ps1 -Configuration "$(CONFIG)" -Version "$(VERSION)"
else
	@echo "Error: WPF build is only supported on Windows"
	@exit 1
endif

build: dll wpf
	@echo "Build complete!"

installer:
	@echo "Building installer..."
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File scripts/build-installer.ps1 -Configuration "$(CONFIG)"
else
	@echo "Error: Installer build is only supported on Windows"
	@exit 1
endif

portable:
	@echo "Building portable distribution..."
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File scripts/build-portable.ps1 -Configuration "$(CONFIG)" -Version "$(VERSION)"
else
	@echo "Error: Portable build is only supported on Windows"
	@exit 1
endif

release:
	@echo "Building release packages..."
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File scripts/build-release.ps1 -Version "$(VERSION)" -Configuration "$(CONFIG)"
else
	@echo "Error: Release build is only supported on Windows"
	@exit 1
endif

test:
	@echo "Running tests..."
	@go test ./... -v -race -coverprofile=coverage.txt -covermode=atomic
ifeq ($(OS),Windows_NT)
	@if exist DeeMusic.Desktop.Tests (dotnet test DeeMusic.Desktop.Tests --verbosity minimal)
endif

clean:
	@echo "Cleaning build artifacts..."
	@go clean
ifeq ($(OS),Windows_NT)
	@if exist deemusic-core.dll del /F /Q deemusic-core.dll
	@if exist deemusic-core.h del /F /Q deemusic-core.h
	@if exist DeeMusic.Desktop\bin rmdir /S /Q DeeMusic.Desktop\bin
	@if exist DeeMusic.Desktop\obj rmdir /S /Q DeeMusic.Desktop\obj
	@if exist dist rmdir /S /Q dist
	@for %%f in (DeeMusic-Setup-*.exe) do del /F /Q "%%f"
	@echo "Clean complete!"
else
	@rm -f deemusic-core.dll deemusic-core.h
	@rm -rf DeeMusic.Desktop/bin DeeMusic.Desktop/obj
	@rm -rf dist
	@rm -f DeeMusic-Setup-*.exe
	@echo "Clean complete!"
endif

install:
	@echo "Installing dependencies..."
	@go mod download
ifeq ($(OS),Windows_NT)
	@dotnet restore DeeMusic.Desktop/DeeMusic.Desktop.csproj
endif
	@echo "Dependencies installed!"

# Version management targets
version-major:
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -Command ". .\scripts\version.ps1; Increment-Version -Part Major"
else
	@echo "Error: Version management is only supported on Windows"
	@exit 1
endif

version-minor:
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -Command ". .\scripts\version.ps1; Increment-Version -Part Minor"
else
	@echo "Error: Version management is only supported on Windows"
	@exit 1
endif

version-patch:
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -Command ". .\scripts\version.ps1; Increment-Version -Part Patch"
else
	@echo "Error: Version management is only supported on Windows"
	@exit 1
endif
