# Copyright (c) Benjamin Zores
# Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
# SPDX-License-Identifier: Apache-2.0

# Make sure GOPATH is NOT set to this folder or we'll get an error "$GOPATH/go.mod exists but should not"
#export GOPATH = ""
export GO111MODULE = on
BINDIR = bin

GOLINT = $(BINDIR)/golangci-lint
GOLINT_VERSION = v1.64.6

GOVULNCHECK = $(BINDIR)/govulncheck
GOVULNCHECK_VERSION = v1.1.4

GOSEC = $(BINDIR)/gosec
GOSEC_VERSION = v2.22.2

V = 0
Q = $(if $(filter 1,$V),,@)
PROD = 0
ifeq ($(PROD),1)
DEBUG = -w -s
endif
M = $(shell printf "\033[34;1m▶\033[0m")

ifeq ($(V), 1)
  OUT = ""
else
  OUT = ">/dev/null"
endif

# This is our default target
# it do all the steps required to build the debian package or docker image
# it does not build/run the tests
.PHONY: all
all: mod fmt vet lint build ; @ ## Do all
	$Q echo "done"

# This target grabs the necessary go modules
.PHONY: mod
mod: ; $(info $(M) collecting modules…) @
	$Q go mod download
	$Q go mod tidy

# Updates all go modules
update: ; $(info $(M) updating modules…) @
	$Q go get -u ./...
	$Q go mod tidy

# Makes sure bin directory is created
.PHONY: bin
bin: ; $(info $(M) create local bin directory) @
	$Q mkdir -p $(BINDIR)

.PHONY: clocker
clocker: bin ; $(info $(M) building Clocker…) @
	$Q go build \
                -ldflags='$(DEBUG)' \
                -o $(BINDIR) ./cmd/clocker

.PHONY: build
build: clocker

.PHONY: tests
tests: ; $(info $(M) testing Clocker suite…) @
	$Q go test ./... -count=1

.PHONY: get-lint
get-lint: ; $(info $(M) downloading go-lint…) @
	$Q test -x $(GOLINT) || sh -c $(GOLINT) --version 2> /dev/null| grep $(GOLINT_VERSION)  || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s $(GOLINT_VERSION)

.PHONY: lint
lint: get-lint ; $(info $(M) running go-lint…) @
	$Q $(GOLINT) run -e SA1019 ./... ; exit 0

.PHONY: get-govulncheck
get-govulncheck: ; $(info $(M) downloading govulncheck…) @
	$Q test -x $(GOVULNCHECK) || GOBIN="$(PWD)/$(BINDIR)/" go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

.PHONY: vuln
vuln: get-govulncheck ; $(info $(M) running govulncheck…) @ ## Check for known vulnerabilities
	$Q $(GOVULNCHECK) ./... ; exit 0

.PHONY: get-gosec
get-gosec: ; $(info $(M) downloading gosec…) @
	$Q test -x $(GOSEC) || GOBIN="$(PWD)/$(BINDIR)/" go install github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION)

.PHONY: sec
sec: get-gosec ; $(info $(M) running gosec…) @ ## AST / SSA code checks
	$Q $(GOSEC) -terse -exclude=G101,G115 ./... ; exit 0

.PHONY: vet
vet: ; $(info $(M) running go vet…) @
	$Q go vet ./... ; exit 0

.PHONY: fmt
fmt: ; $(info $(M) running go fmt…) @
	$Q go fmt ./...

.PHONY: clean
clean: ; $(info $(M) cleaning…) @ ## Cleanup everything
	$Q rm -rf $(BINDIR)
