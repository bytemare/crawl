#
# Some copyright
#

BINARY  :=  $(PROJECT_NAME)

# Project path and name
PROJECT_REPO := $(shell go list -m)
PROJECT_NAME := $(shell basename $(PROJECT_REPO))

# Get the list of packages in the project
PACKAGES    := $(shell go list ./...)

# Use git tags to set version, and commit hash to identify the build
VERSION     := $(shell git describe --tags --always --dirty)
BUILD       := $(shell git rev-parse HEAD)
LD_FLAGS    := -ldflags='-X "main.Version=$(VERSION)" -X "main.BuildTime=$(BUILD)"'

# Go env vars
OS          := $(if $(GOOS),$(GOOS),$(shell go env GOOS))
ARCH        := $(if $(GOARCH),$(GOARCH),$(shell go env GOARCH))
PLATFORMS   ?= linux_amd64 linux_arm64

# Directories for builds and tests
BUILD_DIR   := bin/$(OS)_$(ARCH)
COVERAGE    := coverage

#
#   Commands
#

.PHONY: install lint test cover version clean

# Create directories and build project
$(BINARY):
	@mkdir -p $@

$(BINARY):
	go build -v $(LD_FLAGS) -o $(BUILD_DIR)/$(BINARY)

install:
	go install -v $(LD_FLAGS) -o $(BUILD_DIR)/$(BINARY)

lint:
	golangci-lint run -v ./...

test:
	go test -v -i -race $(PACKAGES)

cover:
	@for PACK in $(PACKAGES); do \
        	echo "Testing $(PACK)"
		go test -v -i -race -covermode=atomic \
                	-coverpkg=$(PACKAGES) \
	                -coverprofile=$(COVERAGE)/unit-`echo $$PACK | tr "/" "_"`.out
	done

version:
	@echo $(VERSION)

clean:
	rm -rf $(BUILD_DIR)
