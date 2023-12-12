# SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
# SPDX-License-Identifier: Apache-2.0

REGISTRY                                     := europe-docker.pkg.dev/gardener-project/public/gardener
APISERVER_PROXY_SIDECAR_IMAGE_REPOSITORY     := $(REGISTRY)/apiserver-proxy
REPO_ROOT              	                     := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
HACK_DIR                                     := $(REPO_ROOT)/hack
VERSION                                      := $(shell cat VERSION)
EFFECTIVE_VERSION                            := $(VERSION)-$(shell git rev-parse HEAD)
LD_FLAGS                                     := "-X github.com/gardener/apiserver-proxy/internal/version.version=$(EFFECTIVE_VERSION)"
GOARCH                                       := amd64
#########################################
# Tools                                 #
#########################################

TOOLS_DIR := hack/tools
include vendor/github.com/gardener/gardener/hack/tools.mk

#################################################################
# Rules related to binary build, Docker image build and release #
#################################################################

.PHONY: build
build:
	@CGO_ENABLED=0 GOARCH=$(GOARCH) GO111MODULE=on go build -mod=vendor -ldflags $(LD_FLAGS) -o bin/apiserver-proxy-sidecar     ./cmd/apiserver-proxy-sidecar

.PHONY: docker-images
docker-images:
	@echo "Building docker images with version and tag $(EFFECTIVE_VERSION)"
	@docker build --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) -t $(APISERVER_PROXY_SIDECAR_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION)     -f cmd/Dockerfile --target apiserver-proxy .

#####################################################################
# Rules for verification, formatting, linting, testing and cleaning #
#####################################################################

.PHONY: revendor
revendor:
	@GO111MODULE=on go mod tidy
	@GO111MODULE=on go mod vendor
	@chmod +x $(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/*

.PHONY: check
check: $(GOIMPORTS) $(GOLANGCI_LINT)
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/check.sh ./cmd/... ./internal/...

.PHONY: format
format: $(GOIMPORTS) $(GOIMPORTSREVISER)
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/format.sh ./cmd ./internal

.PHONY: test
test:
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/test.sh ./cmd/... ./internal/...

.PHONY: test-cov
test-cov:
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/test-cover.sh ./cmd/... ./internal/...

.PHONY: test-cov-clean
test-cov-clean:
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/test-cover-clean.sh

.PHONY: verify
verify: check format test

.PHONY: verify-extended
verify-extended: check format test-cov test-cov-clean
