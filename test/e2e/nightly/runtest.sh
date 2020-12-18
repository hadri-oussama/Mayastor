#!/usr/bin/env bash
set -eux

go test -v ./... -ginkgo.v -ginkgo.progress -timeout 0
