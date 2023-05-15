# VARIABLES
export SHELL:=/bin/bash

registry ?= breathbath/chatgpt
image_tag ?= $(shell cat VERSION )

.PHONY: image
image:
## Build docker image
	docker build -f docker/app/Dockerfile -t $(registry):$(image_tag) --build-arg version=$(image_tag) .


.PHONY: deploy
deploy:
## Deploy latest version
	git reset --hard
	git pull
	docker compose stop chatgpt
	docker compose pull
	APP_VERSION=$(image_tag) docker compose up -d chatgpt --remove-orphans

.PHONY: tag
tag:
## Crate tag from VERSION and push it github
	git tag -f $(image_tag) master
	git push origin $(image_tag) -f