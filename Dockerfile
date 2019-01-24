ARG GOLANG_BASE
ARG ALPINE_BASE
FROM ${GOLANG_BASE} as builder
RUN apk -v add --update ca-certificates jq curl git make bash gcc musl-dev


COPY . /go/src/github.com/docker/stacks
WORKDIR /go/src/github.com/docker/stacks
RUN echo "TODO Would be building"

FROM builder as unit-test
# TODO - temporary unit test wiring...
RUN go test -covermode=count -coverprofile=/cover.out -v $(go list ./pkg/...)
RUN echo "TODO would be doing lint stuff here"

FROM ${ALPINE_BASE} as controller

RUN echo "Would be copying build results from builder"
