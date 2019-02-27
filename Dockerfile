ARG GOLANG_BASE
ARG ALPINE_BASE
FROM ${GOLANG_BASE} as builder
ARG     GOMETALINTER_SHA=v2.0.6
RUN apk -v add --update ca-certificates jq curl git make bash gcc musl-dev linux-headers && \
    go get -d github.com/alecthomas/gometalinter && \
    cd /go/src/github.com/alecthomas/gometalinter && \
    git checkout -q "$GOMETALINTER_SHA" && \
    go build -v -o /usr/local/bin/gometalinter . && \
    gometalinter --install && \
    rm -rf /go/src/* /go/pkg/*
ARG     ESC_SHA=58d9cde84f237ecdd89bd7f61c2de2853f4c5c6e
RUN     go get -d github.com/mjibson/esc && \
        cd /go/src/github.com/mjibson/esc && \
        git checkout -q "$ESC_SHA" && \
        go build -v -o /usr/bin/esc . && \
        rm -rf /go/src/* /go/pkg/*

COPY . /go/src/github.com/docker/stacks
WORKDIR /go/src/github.com/docker/stacks

RUN go generate github.com/docker/stacks/pkg/compose/schema

RUN echo "Building standalone runtime..."

RUN go build -i ./e2e/... && \
    go build -o bin/standalone ./cmd/standalone

FROM builder as unit-test

# TODO - temporary unit test wiring...
RUN go test -coverpkg=./pkg/... -covermode=count -coverprofile=/cover.out -v $(go list ./pkg/...)

FROM builder as lint
RUN gometalinter --config gometalinter.json ./...

# End-to-end Integration Tests
FROM builder as e2e
RUN go test -c ./e2e/...
ENTRYPOINT ["./e2e.test", "-test.v"]

# Standalone server (for testing)
FROM ${ALPINE_BASE} as standalone
COPY --from=builder /go/src/github.com/docker/stacks/bin/standalone /standalone
ENTRYPOINT ["/standalone", "server"]
