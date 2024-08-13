#!/bin/bash -eux

pushd dp-cli
  go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.57.2
  make lint
popd
