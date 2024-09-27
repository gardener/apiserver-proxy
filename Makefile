# SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
# SPDX-License-Identifier: Apache-2.0

ENSURE_GARDENER_MOD         := $(shell go get github.com/gardener/gardener@$$(go list -m -f "{{.Version}}" github.com/gardener/gardener))
GARDENER_HACK_DIR           := $(shell go list -m -f "{{.Dir}}" github.com/gardener/gardener)/hack
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
include $(GARDENER_HACK_DIR)/tools.mk

#################################################################
# Rules related to binary build, Docker image build and release #
#################################################################

.PHONY: build
build:
	@CGO_ENABLED=0 GOARCH=$(GOARCH) GO111MODULE=on go build -ldflags $(LD_FLAGS) -o bin/apiserver-proxy-sidecar ./cmd/apiserver-proxy-sidecar

.PHONY: docker-images
docker-images:
	@echo "Building docker images with version and tag $(EFFECTIVE_VERSION)"
	@docker build --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) -t $(APISERVER_PROXY_SIDECAR_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) --platform=linux/$(GOARCH)    -f cmd/Dockerfile --target apiserver-proxy .

#####################################################################
# Rules for verification, formatting, linting, testing and cleaning #
#####################################################################


.PHONY: tidy
tidy:
	@go mod tidy
	@mkdir -p $(REPO_ROOT)/.ci/hack && cp $(GARDENER_HACK_DIR)/.ci/* $(GARDENER_HACK_DIR)/generate-controller-registration.sh $(REPO_ROOT)/.ci/hack/ && chmod +xw $(REPO_ROOT)/.ci/hack/*
	@cp $(GARDENER_HACK_DIR)/cherry-pick-pull.sh $(HACK_DIR)/cherry-pick-pull.sh && chmod +xw $(HACK_DIR)/cherry-pick-pull.sh

.PHONY: check
check: $(GOIMPORTS) $(GOLANGCI_LINT)
	@bash $(GARDENER_HACK_DIR)//check.sh ./cmd/... ./internal/...

.PHONY: format
format: $(GOIMPORTS) $(GOIMPORTSREVISER)
	@bash $(GARDENER_HACK_DIR)/format.sh ./cmd ./internal

.PHONY: test
test:
	@bash $(GARDENER_HACK_DIR)/test.sh ./cmd/... ./internal/...

.PHONY: test-cov
test-cov:
	@bash $(GARDENER_HACK_DIR)/test-cover.sh ./cmd/... ./internal/...

.PHONY: test-cov-clean
test-cov-clean:
	@bash $(GARDENER_HACK_DIR)/test-cover-clean.sh

.PHONY: verify
verify: check format test

.PHONY: verify-extended
verify-extended: check format test-cov test-cov-clean
