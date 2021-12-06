ARG ONOS_BUILD_VERSION=undefined

FROM onosproject/golang-build:v1.0 as build

ENV GO111MODULE=on
ARG ONOS_MAKE_TARGET=build

COPY Makefile go.mod go.sum /go/src/github.com/onosproject/onos-mho/
COPY cmd/ /go/src/github.com/onosproject/onos-mho/cmd/
COPY pkg/ /go/src/github.com/onosproject/onos-mho/pkg/
COPY vendor/ /go/src/github.com/onosproject/onos-mho/vendor/
COPY build/build-tools /go/src/github.com/onosproject/onos-mho/build/build-tools/

RUN cd /go/src/github.com/onosproject/onos-mho && GOFLAGS=-mod=vendor make ${ONOS_MAKE_TARGET}

FROM alpine:3.11
RUN apk add libc6-compat

USER nobody

COPY --from=build /go/src/github.com/onosproject/onos-mho/build/_output/onos-mho /usr/local/bin/onos-mho

ENTRYPOINT ["onos-mho"]
