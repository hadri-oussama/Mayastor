#!/usr/bin/env bash

set -eux

SCRIPTDIR=$(dirname "$(realpath "$0")")
REGISTRY=$1
TESTS="install basic_volume_io"

# TODO: Add proper argument parser
if [ -z "$REGISTRY" ]; then
  echo "Missing parameter registry"
  exit 1
fi

test_failed=
export e2e_docker_registry="$REGISTRY"
export e2e_pool_device=/dev/nvme1n1

for dir in $TESTS; do
  cd "$SCRIPTDIR/../test/e2e/$dir"
  if ! go test -v . -ginkgo.v -ginkgo.progress -timeout 0 ; then
    test_failed=1
    break
  fi
done

# must always run uninstall test in order to clean up the cluster
cd "$SCRIPTDIR/../test/e2e/uninstall"
go test

if [ -n "$test_failed" ]; then
  exit 1
fi

echo "All tests have passed"
exit 0
