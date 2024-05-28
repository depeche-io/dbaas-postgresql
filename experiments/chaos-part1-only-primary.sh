#!/bin/bash

NS=part1-bitnami-only-primary

echo "PART 1 - Primary pod kill"

for i in {1..5}; do
    echo "delete Primary $(date)"
    kubectl delete -n $NS $(kubectl get po -o name -n $NS -l app.kubernetes.io/component=primary)
    sleep 120
done
