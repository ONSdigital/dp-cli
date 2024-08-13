#!/bin/bash -eux

pushd dp-cli
  make build
  cp build/dp-cli Dockerfile.concourse ../build
popd
