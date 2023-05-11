#!/usr/bin/env bash

echo "Deploy ChatGPT project"

git reset --hard
git pull

VERSION=$(cat VERSION)
echo "VERSION ${VERSION}"
docker-compose build --build-arg version=${VERSION} chatgpt --no-cache
docker-compose stop chatgpt
docker-compose up -d chatgpt