# SPDX-FileCopyrightText: 2019-present Open Networking Foundation <info@opennetworking.org>
#
# SPDX-License-Identifier: Apache-2.0

ARG ONOS_BUILD_VERSION=undefined

FROM onosproject/golang-build:v1.3.0 as build

ENV GO111MODULE=on
ARG ONOS_MAKE_TARGET=build

COPY Makefile go.mod go.sum /go/src/github.com/onosproject/onos-mho/
COPY cmd/ /go/src/github.com/onosproject/onos-mho/cmd/
COPY pkg/ /go/src/github.com/onosproject/onos-mho/pkg/
COPY vendor/ /go/src/github.com/onosproject/onos-mho/vendor/
COPY build/ /go/src/github.com/onosproject/onos-mho/build

RUN cd /go/src/github.com/onosproject/onos-mho && GOFLAGS=-mod=vendor make ${ONOS_MAKE_TARGET}

FROM frolvlad/alpine-glibc:alpine-3.20

USER nobody

COPY --from=build /go/src/github.com/onosproject/onos-mho/build/_output/onos-mho /usr/local/bin/onos-mho

ENTRYPOINT ["onos-mho"]
