
ORG=dockereng
CONTROLLER_IMAGE_NAME=stack-controller
TAG=latest # TODO work out versioning scheme
TEST_SCOPE?=./...
BUILD_ARGS= \
    --build-arg ALPINE_BASE=alpine:3.8 \
    --build-arg GOLANG_BASE=golang:1.11-alpine3.8

build:
	docker build $(BUILD_ARGS) -t $(ORG)/$(CONTROLLER_IMAGE_NAME):$(TAG) .

test:
	docker build $(BUILD_ARGS) -t $(ORG)/$(CONTROLLER_IMAGE_NAME):test --target unit-test .

lint:
	docker build $(BUILD_ARGS) -t $(ORG)/$(CONTROLLER_IMAGE_NAME):lint --target lint .

# For developers...


# Get coverage results in a web browser
cover: test
	docker create --name $(CONTROLLER_IMAGE_NAME)_cover $(ORG)/$(CONTROLLER_IMAGE_NAME):test  && \
	    docker cp $(CONTROLLER_IMAGE_NAME)_cover:/cover.out . && docker rm $(CONTROLLER_IMAGE_NAME)_cover
	go tool cover -html=cover.out

build-mocks:
	@echo "Generating mocks"
	mockgen -package=mocks -destination pkg/mocks/mock_backend.go github.com/docker/stacks/pkg/interfaces BackendClient

generate: pkg/compose/schema/bindata.go

pkg/compose/schema/bindata.go: pkg/compose/schema/data/*.json
	docker build $(BUILD_ARGS) -t $(ORG)/$(CONTROLLER_IMAGE_NAME):build --target builder .
	docker create --name $(CONTROLLER_IMAGE_NAME)_schema $(ORG)/$(CONTROLLER_IMAGE_NAME):build && \
	    docker cp $(CONTROLLER_IMAGE_NAME)_schema:/go/src/github.com/docker/stacks/$@ $@ && docker rm $(CONTROLLER_IMAGE_NAME)_schema
