#!/usr/bin/env bash

echo "Deploy ChartGPT project"

git reset --hard
git pull

VERSION=$(cat VERSION)
echo "VERSION ${VERSION}"
docker-compose build --build-arg version=${VERSION} chartgpt
docker-compose stop chartgpt
docker-compose up -d chartgpt