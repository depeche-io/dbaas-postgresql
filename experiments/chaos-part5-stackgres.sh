#!/bin/bash

source $(dirname $0)/_operator-use-case.sh

NS=part5-stackgres-cluster
SLEEP=120
#COUNT="1 2 3 4 5"
COUNT="1"

LABEL_MASTER=role=master
LABEL_REPLICA=role=replica

PrimaryPodDelete 

ReplicaPodDelete

RandomPvcDelete

PatroniSwitchOver