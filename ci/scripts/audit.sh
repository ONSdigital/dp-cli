#!/bin/bash -eux

export cwd=$(pwd)

pushd $cwd/dp-cli
  make audit
popd
