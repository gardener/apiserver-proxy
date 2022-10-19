# SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
# SPDX-License-Identifier: Apache-2.0

REGISTRY                                     := eu.gcr.io/gardener-project/gardener
APISERVER_PROXY_POD_WEBHOOK_IMAGE_REPOSITORY := $(REGISTRY)/apiserver-proxy-pod-webhook
APISERVER_PROXY_SIDECAR_IMAGE_REPOSITORY     := $(REGISTRY)/apiserver-proxy
VERSION                                      := $(shell cat VERSION)
EFFECTIVE_VERSION                            := $(VERSION)-$(shell git rev-parse HEAD)
LD_FLAGS                                     := "-X github.com/gardener/apiserver-proxy/internal/version.version=$(EFFECTIVE_VERSION)"
GOARCH                                       := amd64

.PHONY: build
build:
	@CGO_ENABLED=0 GOARCH=$(GOARCH) GO111MODULE=on go build -mod=vendor -ldflags $(LD_FLAGS) -o bin/apiserver-proxy-pod-webhook ./cmd/apiserver-proxy-pod-webhook 
	@CGO_ENABLED=0 GOARCH=$(GOARCH) GO111MODULE=on go build -mod=vendor -ldflags $(LD_FLAGS) -o bin/apiserver-proxy-sidecar     ./cmd/apiserver-proxy-sidecar

.PHONY: test
test:
	@GO111MODULE=on go test -mod=vendor ./...

.PHONY: revendor
revendor:
	@GO111MODULE=on go mod tidy
	@GO111MODULE=on go mod vendor

.PHONY: docker-images
docker-images:
	@echo "Building docker images with version and tag $(EFFECTIVE_VERSION)"
	@docker build --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) -t $(APISERVER_PROXY_POD_WEBHOOK_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f cmd/Dockerfile --target apiserver-proxy-pod-webhook .
	@docker build --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) -t $(APISERVER_PROXY_SIDECAR_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION)     -f cmd/Dockerfile --target apiserver-proxy .
