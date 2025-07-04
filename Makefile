SHELL := /bin/bash

.PHONY: default clean build shellcheck abcgo golangci-lint style run test cover rest_api_tests rules_content sqlite_db license before-commit openapi-check help install-addlicense install-golangci-lint

SOURCES:=$(shell find . -name '*.go')
BINARY:=insights-results-smart-proxy
DOCFILES:=$(addprefix docs/packages/, $(addsuffix .html, $(basename ${SOURCES})))

default: build

clean: ## Run go clean
	@go clean

build: ${BINARY} ## Build binary containing service executable

build-cover:	${SOURCES}  ## Build binary with code coverage detection support
	./build.sh -cover

${BINARY}: ${SOURCES}
	./build.sh

shellcheck: ## Run shellcheck
	shellcheck --exclude=SC1090,SC2086,SC2034,SC1091,SC2317 *.sh

abcgo: ## Run ABC metrics checker
	@echo "Run ABC metrics checker"
	./abcgo.sh

openapi-check:
	./check_openapi.sh

style: shellcheck abcgo golangci-lint ## Run all the formatting related commands (fmt, vet, lint, cyclo) + check shell scripts

run: clean build ## Build the project and executes the binary
	./insights-results-smart-proxy

test: clean build ## Run the unit tests
	./unit-tests.sh

cover: test
	@go tool cover -html=coverage.out

coverage:
	@go tool cover -func=coverage.out

license: install-addlicense
	addlicense -c "Red Hat, Inc" -l "apache" -v ./

before-commit: style test license openapi-check
	./check_coverage.sh

help: ## Show this help screen
	@echo 'Usage: make <OPTIONS> ... <TARGETS>'
	@echo ''
	@echo 'Available targets are:'
	@echo ''
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ''

function-list: ${BINARY} ## List all functions in generated binary file
	go tool objdump ${BINARY} | grep ^TEXT | sed "s/^TEXT\s//g"

install-addlicense:
	[[ `command -v addlicense` ]] || go install github.com/google/addlicense

golangci-lint: install-golangci-lint
	@echo "Running linters and formatters with auto-fix...";
	@echo "-----------------------------------------------------------------------"; 
	@echo -e "\033[1;33mReview golangci-lint fixes and resolve any issues it couldn’t auto-fix\033[0m"
	@echo "-----------------------------------------------------------------------"; 
	golangci-lint run --fix
	golangci-lint fmt

install-golangci-lint:
	@if [ "$$(uname)" = "Darwin" ]; then \
		brew install golangci-lint; \
		brew upgrade golangci-lint; \
	else \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.0.2; \
	fi