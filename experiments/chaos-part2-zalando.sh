#!/bin/bash

NS=part2-zalando
SLEEP=10
COUNT=2

echo "PART 1 - Primary pod delete"

for i in {1..$COUNT}; do
    echo "delete Primary $(date)"
    kubectl delete -n $NS $(kubectl get po -o name -n $NS -l app.kubernetes.io/component=primary)
    sleep $SLEEP
done

echo "PART 2 - Slave pod delete"

for i in {1..$COUNT}; do
    array=()
    while IFS= read -r line; do
        array+=( "$line" )
    done < <( kubectl get po -o name -n $NS -l app.kubernetes.io/component=read )

    size=${#array[@]}
    index=$(($RANDOM % $size))
    RANDOM_POD=${array[$index]}

    echo "delete random Slave $(date) $RANDOM_POD"
    kubectl delete -n $NS $RANDOM_POD
    sleep $SLEEP
done

echo "PART 3 - Random PVC delete"

for i in {1..$COUNT}; do
    array=()
    while IFS= read -r line; do
        array+=( "$line" )
    done < <( kubectl get pods -n $NS -o=json | jq -c '.items[] | {name: .metadata.name, namespace: .metadata.namespace, claimName: .spec |  select( has ("volumes") ).volumes[] | select( has ("persistentVolumeClaim") ).persistentVolumeClaim.claimName }' )

    size=${#array[@]}
    index=$(($RANDOM % $size))
    RANDOM_LINE=${array[$index]}

    echo "delete random PVC and pod $(date) $RANDOM_LINE"
    PVC_NAME=$( echo $RANDOM_LINE | jq -r .claimName )
    kubectl delete -n $NS pvc $PVC_NAME &
    sleep 1
    kubectl patch -n $NS pvc $PVC_NAME -p '{"metadata":{"finalizers":null}}' --type=merge

    kubectl delete -n $NS pod $( echo $RANDOM_LINE | jq -r .name )
    sleep $SLEEP
done
