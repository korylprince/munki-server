#!/bin/bash
set -e

version=$1

tag="korylprince/munki-server"

docker build --no-cache --build-arg "VERSION=$version" --tag "$tag:$version" .

docker push "$tag:$version"

if [ "$2" = "latest" ]; then
    docker tag "$tag:$version" "$tag:latest"
    docker push "$tag:latest"
fi
