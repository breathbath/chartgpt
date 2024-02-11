#!/bin/bash

# VARIABLES
registry="breathbath/chatgpt"
GOLANGCI_LINT_VERSION="1.56.1"

image() {
    # Build docker image
    image_tag=$(cat VERSION)
    docker build -f docker/app/Dockerfile -t "breathbath/chatgpt":$image_tag --build-arg version=$image_tag .
}

deploy() {
    # Deploy latest version
    git reset --hard
    git pull
    image_tag=$(cat VERSION)
    APP_VERSION=$image_tag docker compose stop chatgpt grafana
    APP_VERSION=$image_tag docker compose pull
    APP_VERSION=$image_tag docker compose up -d chatgpt grafana --remove-orphans
}

tag() {
    # Create tag from VERSION and push it to GitHub
    image_tag=$(cat VERSION)
    cur_branch=$(git rev-parse --abbrev-ref HEAD)
    git tag -f $image_tag $cur_branch
    git push origin $image_tag -f
}

fmt() {
    # Run gofumpt
    check_version "gofumpt" "v$GOFUMPT_VERSION" "$(gofumpt -version | cut -d' ' -f1)"
    gofumpt -l -w -e .
}

lint() {
    # Run golangci-lint
    check_version "golangci-lint" $GOLANGCI_LINT_VERSION "$(golangci-lint version | grep -oE "version.*?[[:digit:].]+" | grep -oE "[[:digit:]]+\.[[:digit:]]+\.[[:digit:]]+")"
    golangci-lint run
}

check_version() {
    local tool_name=$1
    local required_version=$2
    local installed_version=$3

    if [[ "$required_version" != "$installed_version" ]]; then
        echo "Error: $tool_name version $required_version is required, but found $installed_version"
        exit 1
    fi
}

case "$1" in
    "image")
        image "${@:2}"
        ;;
    "deploy")
        deploy "${@:2}"
        ;;
    "tag")
        tag "${@:2}"
        ;;
    "fmt")
        fmt "${@:2}"
        ;;
    "lint")
        lint "${@:2}"
        ;;
     *)
        echo "Invalid command. Options: image, deploy, tag, fmt, lint"
        exit 1
        ;;
esac