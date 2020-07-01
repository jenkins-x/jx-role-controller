SHELL := /bin/bash
NAME := jx-role-controller
GO := GO111MODULE=on GO15VENDOREXPERIMENT=1 go
GO_NOMOD := GO111MODULE=off go
GOTEST := $(GO) test
PACKAGE_DIRS := $(shell $(GO) list ./... | grep -v /vendor/)
GO_DEPENDENCIES := $(shell find . -type f -name '*.go')

build: 
	$(GO) build ./...

test: build
	$(GOTEST) -coverprofile=coverage.out ./...

test1: ## Runs single test specified by test name and optional package, eg 'make test1 TEST=TestGitCLI'
	$(GOTEST) -v ./pkg/log -run $(TEST)

get-fmt-deps: ## Install test dependencies
	$(GO_NOMOD) get golang.org/x/tools/cmd/goimports

fmt: importfmt ## Format the code
	@FORMATTED=`$(GO) fmt $(PACKAGE_DIRS)`
	@([[ ! -z "$(FORMATTED)" ]] && printf "Fixed unformatted files:\n$(FORMATTED)") || true

importfmt: get-fmt-deps
	@echo "Formatting the imports..."
	goimports -w $(GO_DEPENDENCIES)

.PHONY: lint
lint: ## Lint the code
	./hack/linter.sh

.PHONY: modtidy
modtidy:
	$(GO) mod tidy

.PHONY: coverage
coverage:
	$(GO) tool cover -html=coverage.out

.PHONY: cover
cover:
	$(GO) tool cover -func coverage.out | grep total

scratch:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(BUILDTAGS) $(BUILDFLAGS) -o build/$(NAME) main.go

linux:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=amd64 $(GO) build $(BUILDTAGS) $(BUILDFLAGS) -o build/$(NAME)-linux-amd64 main.go

docker: linux
	docker build -t jenkins-x/$(NAME) .
