# VARIABLES
export SHELL:=/bin/bash

registry ?= breathbath/chatgpt
image_tag ?= $(shell git symbolic-ref --short HEAD -q || echo "deatached" )

.PHONY: image
image: ##
## Build docker image
	docker build -f docker/app/Dockerfile -t $(registry):$(image_tag) --build-arg version=$(image_tag) .


.PHONY: deploy
deploy: ##
## Deploy latest version
	git reset --hard
	git pull
	VERSION=$(cat VERSION)
	echo "VERSION ${VERSION}"
	docker-compose build --build-arg version=${VERSION} chatgpt --no-cache
	docker-compose stop chatgpt
	docker-compose up -d chatgpt --force-recreate