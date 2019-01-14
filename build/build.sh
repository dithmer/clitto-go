#!/bin/bash
find -name "dist-*" -delete
set -xeou pipefail
cd "$(dirname "$0")/.."
docker run -t --rm -v "$PWD/clitto":/usr/src/myapp -v "$PWD/build":/build -w /usr/src/myapp golang:1.8 /build/buildrunner.sh clitto
docker run -t --rm -v "$PWD/clittod":/usr/src/myapp -v "$PWD/build":/build -w /usr/src/myapp golang:1.8 /build/buildrunner.sh clittod
mkdir -p dist
rm -rf dist/*
find -name "dist-*" -exec mv "{}" dist/ \;
cd dist
for file in dist*;
do
    mv "$file" "${file#dist-}"
done
