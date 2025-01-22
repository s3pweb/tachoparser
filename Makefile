# Company
COMPANY = s3pweb

# Command to build
CMD_NAME = dddhttp

# Docker image name and tag
IMAGE_NAME = $(COMPANY)/tachoparser-$(CMD_NAME)
IMAGE_TAG = 0.0.1

# Build docker image for the platform linux/amd64
.PHONY: docker.build
docker.build:
	docker build --platform linux/amd64 -t $(IMAGE_NAME):$(IMAGE_TAG) -f cmd/$(CMD_NAME)/Dockerfile .

# Run docker image
.PHONY: docker.run
docker.run:
	docker run -p 8080:8080 $(IMAGE_NAME):$(IMAGE_TAG)

.PHONY: docker.push
docker.push:
	docker push $(IMAGE_NAME):$(IMAGE_TAG)

# Build docker image for the current platform
.PHONY: docker.build.local
docker.build.local:
	docker build -t $(IMAGE_NAME)-local:$(IMAGE_TAG) -f cmd/$(CMD_NAME)/Dockerfile .

# Run docker image locally
.PHONY: docker.run.local
docker.run.local:
	docker run -p 8080:8080 $(IMAGE_NAME)-local:$(IMAGE_TAG)