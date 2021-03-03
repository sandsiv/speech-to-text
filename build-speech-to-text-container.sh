#!/bin/bash
version=`git describe --tags --abbrev=0`
version_name=${TAG:-$version}
version_stripped=${version_name//\//_}
container_tag=${version_stripped#"origin_"}

pkgname=speech-to-text

echo Building version: $version_name
set -e
set -x

docker build -t $pkgname -f Dockerfile .

PROJECT_ID=${GCP_PROJECT_ID:-"seraphic-vertex-179007"}

docker tag ${pkgname} eu.gcr.io/${PROJECT_ID}/${pkgname}:${container_tag}
docker push eu.gcr.io/${PROJECT_ID}/${pkgname}:${container_tag}

