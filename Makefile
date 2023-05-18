# VARIABLES
export SHELL:=/bin/bash

registry ?= breathbath/chatgpt
image_tag ?= $(shell cat VERSION )
GOLANGCI_LINT_VERSION=1.52.2

.PHONY: image
image:
## Build docker image
	docker build -f docker/app/Dockerfile -t $(registry):$(image_tag) --build-arg version=$(image_tag) .


.PHONY: deploy
deploy:
## Deploy latest version
	git reset --hard
	git pull
	APP_VERSION=$(image_tag) docker compose stop chatgpt
	APP_VERSION=$(image_tag) docker compose pull
	APP_VERSION=$(image_tag) docker compose up -d chatgpt --remove-orphans

.PHONY: tag
tag:
## Crate tag from VERSION and push it github
	git tag -f $(image_tag) master
	git push origin $(image_tag) -f

.PHONY: fmt
fmt: ##
## Run gofumpt
	$(call check_version,gofumpt,v$(GOFUMPT_VERSION),$$(gofumpt -version | cut -d' ' -f1))
	gofumpt -l -w -e .


.PHONY: lint
lint: ##
## Run golangci-lint
	$(call check_version,golangci-lint,$(GOLANGCI_LINT_VERSION),$$(golangci-lint version| grep -oE "version.*?[[:digit:].]+" | grep -oE "[[:digit:]]+\.[[:digit:]]+\.[[:digit:]]+"))
	golangci-lint run