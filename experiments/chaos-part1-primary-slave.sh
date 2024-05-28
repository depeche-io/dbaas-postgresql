#!/bin/bash

NS=part1-bitnami-only-primary

echo "PART 1 - Primary pod kill"

for i in {1..5}; do
    echo "delete Primary $(date)"
    kubectl delete -n $NS $(kubectl get po -o name -n $NS -l app.kubernetes.io/component=primary)
    sleep 120
done

for i in {1..5}; do
    array=()
    while IFS= read -r line; do
        array+=( "$line" )
    done < <( kubectl get po -o name -n $NS -l app.kubernetes.io/component=read )

    size=${#array[@]}
    index=$(($RANDOM % $size))
    RANDOM_POD=${array[$index]}

    echo "delete random Slave $(date) $RANDOM_POD"
    kubectl delete -n $NS $RANDOM_POD
    sleep 120
done
