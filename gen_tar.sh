#!/bin/bash
set -e

version=$(git tag | tail -n 1)
mkdir -p ../nvwa-$version
cp -rf * ../nvwa-$version
tar -czvf nvwa-$version.tar.gz ../nvwa-$version
rm -rf ../nvwa-$version
