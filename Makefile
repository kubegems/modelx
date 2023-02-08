# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

BUILD_DATE?=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_VERSION?=$(shell git describe --tags --dirty --abbrev=0 2>/dev/null || git symbolic-ref --short HEAD)
GIT_COMMIT?=$(shell git rev-parse HEAD 2>/dev/null)
GIT_BRANCH?=$(shell git symbolic-ref --short HEAD 2>/dev/null)
# semver version
VERSION?=$(shell echo "${GIT_VERSION}" | sed -e 's/^v//')
# semver version
SEMVER_VERSION?=$(shell echo "${GIT_VERSION}" | sed -e 's/^v//')
BIN_DIR = ${PWD}/bin

IMAGE_REGISTRY?=docker.io
IMAGE_TAG=${GIT_VERSION}
ifeq (${IMAGE_TAG},main)
   IMAGE_TAG = latest
endif
# Image URL to use all building/pushing image targets
IMG ?=  ${IMAGE_REGISTRY}/kubegems/modelx:$(IMAGE_TAG)
DLIMG ?=  ${IMAGE_REGISTRY}/kubegems/modelxdl:$(IMAGE_TAG)

GOPACKAGE=$(shell go list -m)
ldflags+=-w -s
ldflags+=-X '${GOPACKAGE}/pkg/version.gitVersion=${GIT_VERSION}'
ldflags+=-X '${GOPACKAGE}/pkg/version.gitCommit=${GIT_COMMIT}'
ldflags+=-X '${GOPACKAGE}/pkg/version.buildDate=${BUILD_DATE}'
ldflags+=-extldflags=-static

PUSH?=false

##@ All

all: build image ## build all

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

check: linter ## Static code check.
	${LINTER} run ./...

define build
	@echo "Building ${1}/${2}"
	@GOOS=${1} GOARCH=$(2) go build -gcflags=all="-N -l" -ldflags="${ldflags}" -o ${BIN_DIR}/modelx-$(1)-$(2) ${GOPACKAGE}/cmd/modelx
	@GOOS=${1} GOARCH=$(2) go build -gcflags=all="-N -l" -ldflags="${ldflags}" -o ${BIN_DIR}/modelxd-$(1)-$(2) ${GOPACKAGE}/cmd/modelxd
	@GOOS=${1} GOARCH=$(2) go build -gcflags=all="-N -l" -ldflags="${ldflags}" -o ${BIN_DIR}/modelxdl-$(1)-$(2) ${GOPACKAGE}/cmd/modelxdl
endef

##@ Build
OS:=$(shell go env GOOS)
ARCH:=$(shell go env GOARCH)
build: ## Build binaries.
	$(call build,${OS},${ARCH})
	@cp ${BIN_DIR}/modelx-${OS}-${ARCH} ${BIN_DIR}/modelx

build-all:
	$(call build,linux,amd64)
	$(call build,linux,arm64)
	$(call build,darwin,amd64)
	$(call build,darwin,arm64)
	$(call build,windows,amd64)

image:
	docker buildx build --platform=${OS}/${ARCH} --tag ${IMG} --push=${PUSH} -f Dockerfile ${BIN_DIR}
	docker buildx build --platform=${OS}/${ARCH} --tag ${DLIMG} --push=${PUSH} -f Dockerfile.dl ${BIN_DIR}

PLATFORM?=linux/amd64,linux/arm64,darwin/arm64,darwin/amd64,windows/amd64
image-all: ## Build container image.
	docker buildx build --platform=${PLATFORM} --tag ${IMG} --push=${PUSH} -f Dockerfile ${BIN_DIR}
	docker buildx build --platform=${PLATFORM} --tag ${DLIMG} --push=${PUSH} -f Dockerfile.dl ${BIN_DIR}

helm-package:
	helm package charts/modelx --version=${SEMVER_VERSION} --app-version=${SEMVER_VERSION} 

HELM_REPO_USERNAME?=kubegems
HELM_REPO_PASSWORD?=
CHARTMUSEUM_ADDR?=https://${HELM_REPO_USERNAME}:${HELM_REPO_PASSWORD}@charts.kubegems.io/kubegems
helm-push:
	curl --data-binary "@modelx-${SEMVER_VERSION}.tgz" ${CHARTMUSEUM_ADDR}/api/charts

clean:
	- rm -rf ${BIN_DIR}
