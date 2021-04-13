ARG ONOS_BUILD_VERSION=undefined

FROM onosproject/golang-build:v0.6.7 as build

ENV GO111MODULE=on
ARG ONOS_MAKE_TARGET=build

COPY Makefile go.mod go.sum /go/src/github.com/onosproject/onos-mho/
COPY cmd/ /go/src/github.com/onosproject/onos-mho/cmd/
COPY pkg/ /go/src/github.com/onosproject/onos-mho/pkg/
COPY vendor/ /go/src/github.com/onosproject/onos-mho/vendor/
COPY api/ /go/src/github.com/onosproject/onos-mho/api

RUN cd /go/src/github.com/onosproject/onos-mho && GOFLAGS=-mod=vendor make ${ONOS_MAKE_TARGET}

FROM alpine:3.11
RUN apk add libc6-compat

USER nobody

COPY --from=build /go/src/github.com/onosproject/onos-mho/build/_output/onos-mho /usr/local/bin/onos-mho

ENTRYPOINT ["onos-mho"]
