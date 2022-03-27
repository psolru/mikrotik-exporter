#!/bin/bash
set -e
set -x

DIR=$(pwd)
NAME=$(basename "${DIR}")
REVISION=$(git rev-parse --short HEAD)
VERSION=${VERSION:-$REVISION}

docker buildx build \
	--platform linux/amd64,linux/arm/v7,linux/arm64/v8 \
	--file Dockerfile \
	--tag ogi4i/"${NAME}":"${VERSION}" \
	--output=type=registry \
	.
