#!/bin/bash
set -eux -o pipefail

git clone https://github.com/izakp/helm.git
cd helm

make build

mv ./bin/helm /usr/local/bin/helm

helm version
