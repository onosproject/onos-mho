# SPDX-FileCopyrightText: 2019-present Open Networking Foundation <info@opennetworking.org>
#
# SPDX-License-Identifier: Apache-2.0

export CGO_ENABLED=1
export GO111MODULE=on

.PHONY: build

ONOS_MHO_VERSION := latest
ONOS_PROTOC_VERSION := v0.6.6
BUF_VERSION := 0.27.1

build: # @HELP build the Go binaries and run all validations (default)
build:
	GOPRIVATE="github.com/onosproject/*" go build -o build/_output/onos-mho ./cmd/onos-mho

build-tools:=$(shell if [ ! -d "./build/build-tools" ]; then cd build && git clone https://github.com/onosproject/build-tools.git; fi)
include ./build/build-tools/make/onf-common.mk

test: # @HELP run the unit tests and source code validation
test: build deps linters license_check_apache
	go test -race github.com/onosproject/onos-mho/pkg/...
	go test -race github.com/onosproject/onos-mho/cmd/...

jenkins-test:  # @HELP run the unit tests and source code validation producing a junit style report for Jenkins
jenkins-test: deps license_check_apache linters
	TEST_PACKAGES=github.com/onosproject/onos-mho/... ./build/build-tools/build/jenkins/make-unit

buflint: #@HELP run the "buf check lint" command on the proto files in 'api'
	docker run -it -v `pwd`:/go/src/github.com/onosproject/onos-mho \
		-w /go/src/github.com/onosproject/onos-mho/api \
		bufbuild/buf:${BUF_VERSION} check lint

protos: # @HELP compile the protobuf files (using protoc-go Docker)
protos:
	docker run -it -v `pwd`:/go/src/github.com/onosproject/onos-mho \
		-w /go/src/github.com/onosproject/onos-mho \
		--entrypoint build/bin/compile-protos.sh \
		onosproject/protoc-go:${ONOS_PROTOC_VERSION}

helmit-mho: integration-test-namespace # @HELP run MHO tests locally
	helmit test -n test ./cmd/onos-mho-test --timeout 30m --no-teardown \
			--secret sd-ran-username=${repo_user} --secret sd-ran-password=${repo_password} \
			--suite mho

helmit-ha: integration-test-namespace # @HELP run MHO HA tests locally
	helmit test -n test ./cmd/onos-mho-test --timeout 30m --no-teardown \
			--secret sd-ran-username=${repo_user} --secret sd-ran-password=${repo_password} \
			--suite ha

integration-tests: helmit-mho helmit-ha # @HELP run all MHO integration tests locally

onos-mho-docker: # @HELP build onos-mho Docker image
onos-mho-docker:
	@go mod vendor
	docker build . -f build/onos-mho/Dockerfile \
		-t onosproject/onos-mho:${ONOS_MHO_VERSION}
	@rm -rf vendor

images: # @HELP build all Docker images
images: build onos-mho-docker

kind: # @HELP build Docker images and add them to the currently configured kind cluster
kind: images
	@if [ "`kind get clusters`" = '' ]; then echo "no kind cluster found" && exit 1; fi
	kind load docker-image onosproject/onos-mho:${ONOS_MHO_VERSION}

all: build images

publish: # @HELP publish version on github and dockerhub
	./build/build-tools/publish-version ${VERSION} onosproject/onos-mho

jenkins-publish: jenkins-tools # @HELP Jenkins calls this to publish artifacts
	./build/bin/push-images
	./build/build-tools/release-merge-commit

clean:: # @HELP remove all the build artifacts
	rm -rf ./build/_output ./vendor ./cmd/onos-mho/onos-mho ./cmd/onos/onos
	go clean -testcache github.com/onosproject/onos-mho/...

