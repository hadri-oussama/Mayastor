#!/usr/bin/env bash

set -eux

SCRIPTSDIR=$(dirname "$(realpath "$0")")
TESTDIR=$(realpath "$SCRIPTSDIR/../test/e2e")
REGISTRY=$1
TESTS=${e2e_tests:-"install basic_volume_io csi uninstall"}

function cleanup {
    # Generic workaround for "troublesome" tests,
    # usually these will be 3rd party tests.
    # We want to run the tests, however a future
    # feature/bug-fix will remove the need for
    # external cleanup.
    #
    # affected tests:
    #   - csi (see CAS-566) - delete leftover MSVs
    if [[ -x ./cleanup.sh ]]; then
        ./cleanup.sh
    fi
}

# Run go test in directory specified as $1 (relative path)
function runGoTest {
    echo "Run go test in $PWD/\"$1\""
    if [ -z "$1" ] || [ ! -d "$1" ]; then
        return 1
    fi

    cd "$1"
    if ! go test -v . -ginkgo.v -ginkgo.progress -timeout 0; then
        cleanup
        return 1
    fi

    cleanup
    return 0
}


# TODO: Add proper argument parser
if [ -z "$REGISTRY" ]; then
  echo "Missing parameter registry"
  exit 1
fi

test_failed=
export e2e_docker_registry="$REGISTRY"
export e2e_pool_device=${e2e_pool_device:-/dev/nvme1n1}

for dir in $TESTS; do
  cd "$TESTDIR"
  case "$dir" in
      uninstall)
          # defer to next loop
          ;;
      *)
        if ! runGoTest "$dir" ; then
            test_failed=1
            break
        fi
        ;;
  esac
done

for dir in $TESTS; do
  cd "$TESTDIR"
  case "$dir" in
      uninstall)
        echo "Uninstalling mayastor....."
        if ! runGoTest "$dir" ; then
            test_failed=1
        fi
        ;;
      *)
        ;;
   esac
done

if [ -n "$test_failed" ]; then
    echo "At least one test has FAILED!"
  exit 1
fi

echo "All tests have PASSED!"
exit 0
