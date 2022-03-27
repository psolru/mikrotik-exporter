#!/usr/bin/env bash

set -e
set -x

TAG=$(git describe --abbrev=0 --tags)
REPO_NAME="mikrotik-exporter"
IMAGE="ogi4i/$REPO_NAME"
PLATFORM="linux/amd64,linux/arm64,linux/arm"

docker login -u "$DOCKER_USERNAME" -p "$DOCKER_PASSWORD"

docker buildx create --name "$REPO_NAME" --use --append
docker buildx build --platform "$PLATFORM" -t "$IMAGE:$TAG" -t "$IMAGE:latest" --push .
docker buildx imagetools inspect "$IMAGE:latest"
