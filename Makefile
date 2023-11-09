NAME := weops-kafka-adapter
HUB := docker-bkrepo.cwoa.net/ce1b09/weops-docker
TAG := dev

build:
	docker build -t $(HUB)/$(NAME):$(TAG) -f Dockerfile .

push:
	docker push $(HUB)/$(NAME):$(TAG)