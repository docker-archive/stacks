
ORG=dockereng
CONTROLLER_IMAGE_NAME=stack-controller
TAG=latest # TODO work out versioning scheme
BUILD_ARGS=--build-arg ALPINE_BASE=alpine:3.8

build:
	docker build $(BUILD_ARGS) -t $(ORG)/$(CONTROLLER_IMAGE_NAME):$(TAG) .

test:
	docker build $(BUILD_ARGS) -t $(ORG)/$(CONTROLLER_IMAGE_NAME):test --target unit-test .


lint:
	echo "not yet implemented yet"
