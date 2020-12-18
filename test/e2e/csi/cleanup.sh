#!/usr/bin/env bash

# Required until CAS-566
# "Mayastor volumes not destroyed when PV is destroyed if storage class reclaim policy is Retain"
# is fixed.
echo "Workaround cleanup for CSI tests, see CAS-566"
kubectl -n mayastor delete msv --all
