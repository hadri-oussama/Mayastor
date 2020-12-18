#!/usr/bin/env bash
go test -v -timeout=0 . -ginkgo.v -ginkgo.progress
./cleanup.sh
