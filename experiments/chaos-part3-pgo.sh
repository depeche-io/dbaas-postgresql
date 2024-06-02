#!/bin/bash

source $(dirname $0)/_operator-use-case.sh

NS=part3-pgo-cluster
SLEEP=120
COUNT="1 2 3 4 5 6 7 8 9 10"

LABEL_MASTER=postgres-operator.crunchydata.com/role=master
LABEL_REPLICA=postgres-operator.crunchydata.com/role=replica 


#PrimaryPodDelete 
#
#ReplicaPodDelete
#
#PatroniSwitchOver

RandomPvcDelete