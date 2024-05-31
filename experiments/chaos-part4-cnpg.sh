#!/bin/bash

source $(dirname $0)/_operator-use-case.sh

NS=part4-cnpg-cluster
SLEEP=120
COUNT="1 2 3 4 5 6 7 8 9 10"

LABEL_MASTER=role=primary
LABEL_REPLICA=role=replica


PrimaryPodDelete 

ReplicaPodDelete

RandomPvcDelete

echo "Part 4 - Promote replica to primary"

for i in $COUNT; do
    array=()
    while IFS= read -r line; do
        array+=( "$line" )
    done < <( kubectl get po -o name -n $NS -l $LABEL_REPLICA | sed "s/pod\///" )

    size=${#array[@]}
    index=$(($RANDOM % $size))
    RANDOM_POD=${array[$index]}

    echo "Switch over to $(date) $RANDOM_POD"
    kubectl cnpg promote -n $NS  mycnpg $RANDOM_POD
    sleep $SLEEP
done