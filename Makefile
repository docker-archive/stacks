
ORG=dockereng
CONTROLLER_IMAGE_NAME=stack-controller
TAG=latest # TODO work out versioning scheme
BUILD_ARGS= \
    --build-arg ALPINE_BASE=alpine:3.8 \
    --build-arg GOLANG_BASE=golang:1.11-alpine3.8

build:
	docker build $(BUILD_ARGS) -t $(ORG)/$(CONTROLLER_IMAGE_NAME):$(TAG) .

test:
	docker build $(BUILD_ARGS) -t $(ORG)/$(CONTROLLER_IMAGE_NAME):test --target unit-test .

# For developers...
cover: test
	docker create --name $(CONTROLLER_IMAGE_NAME)_cover $(ORG)/$(CONTROLLER_IMAGE_NAME):test --target unit-test && \
	    docker cp $(CONTROLLER_IMAGE_NAME)_cover:/cover.out . && docker rm $(CONTROLLER_IMAGE_NAME)_cover 
	go tool cover -html=cover.out


lint:
	echo "not yet implemented yet"
