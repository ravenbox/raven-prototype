# Project name
PROJECT_NAME := raven

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get

# Binary name
BINARY_NAME := $(PROJECT_NAME)

# Build directory
BUILD_DIR := build

# Main package path
MAIN_PACKAGE := ./cmd/$(PROJECT_NAME)

# Determine the operating system
ifeq ($(OS),Windows_NT)
	BINARY_NAME := $(BINARY_NAME).exe
    RM := del /Q
    MKDIR := mkdir
else
    RM := rm -f
    MKDIR := mkdir -p
endif

.PHONY: all build clean test deps

all: test build

build:
	$(MKDIR) $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)

clean:
	$(GOCLEAN)
	$(RM) $(BUILD_DIR)/$(BINARY_NAME)

test:
	$(GOTEST) -v ./...

deps:
	$(GOGET) github.com/spf13/cobra@latest
	$(GOGET) github.com/spf13/viper@latest

# Run the application
run:
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	./$(BUILD_DIR)/$(BINARY_NAME)

# Cross compilation
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)

build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PACKAGE)

build-mac:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)
