ARG GOLANG_BASE
ARG ALPINE_BASE
FROM ${GOLANG_BASE} as builder
ARG     GOMETALINTER_SHA=v2.0.6
RUN apk -v add --update ca-certificates jq curl git make bash gcc musl-dev && \
    go get -d github.com/alecthomas/gometalinter && \
    cd /go/src/github.com/alecthomas/gometalinter && \
    git checkout -q "$GOMETALINTER_SHA" && \
    go build -v -o /usr/local/bin/gometalinter . && \
    gometalinter --install && \
    rm -rf /go/src/* /go/pkg/*


COPY . /go/src/github.com/docker/stacks
WORKDIR /go/src/github.com/docker/stacks
RUN echo "TODO Would be building"

FROM builder as unit-test
# TODO - temporary unit test wiring...
RUN go test -covermode=count -coverprofile=/cover.out -v $(go list ./pkg/...)

FROM builder as lint
RUN gometalinter --config gometalinter.json ./...

FROM ${ALPINE_BASE} as controller

RUN echo "Would be copying build results from builder"
