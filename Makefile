
ORG=dockereng
CONTROLLER_IMAGE_NAME=stack-controller
E2E_IMAGE_NAME=stack-e2e
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

standalone:
	docker build $(BUILD_ARGS) -t $(ORG)/$(CONTROLLER_IMAGE_NAME):$(TAG) --target standalone .

e2e:
	docker build $(BUILD_ARGS) -t $(ORG)/$(E2E_IMAGE_NAME):$(TAG) --target e2e .

# For developers...


# Get coverage results in a web browser
cover: test
	docker create --name $(CONTROLLER_IMAGE_NAME)_cover $(ORG)/$(CONTROLLER_IMAGE_NAME):test  && \
	    docker cp $(CONTROLLER_IMAGE_NAME)_cover:/cover.out . && docker rm $(CONTROLLER_IMAGE_NAME)_cover
	go tool cover -html=cover.out

build-mocks:
	@echo "Generating mocks"
	mockgen -package=mocks github.com/docker/stacks/pkg/interfaces BackendClient | sed s,github.com/docker/stacks/vendor/,,g > pkg/mocks/mock_backend.go

generate: pkg/compose/schema/bindata.go

pkg/compose/schema/bindata.go: pkg/compose/schema/data/*.json
	docker build $(BUILD_ARGS) -t $(ORG)/$(CONTROLLER_IMAGE_NAME):build --target builder .
	docker create --name $(CONTROLLER_IMAGE_NAME)_schema $(ORG)/$(CONTROLLER_IMAGE_NAME):build && \
	    docker cp $(CONTROLLER_IMAGE_NAME)_schema:/go/src/github.com/docker/stacks/$@ $@ && docker rm $(CONTROLLER_IMAGE_NAME)_schema

.PHONY: e2e
