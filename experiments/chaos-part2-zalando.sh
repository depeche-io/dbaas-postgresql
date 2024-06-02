#!/bin/bash

source $(dirname $0)/_operator-use-case.sh

NS=part2-zalando
SLEEP=120
COUNT="1 2 3 4 5 6 7 8 9 10"

LABEL_MASTER=spilo-role=master
LABEL_REPLICA=spilo-role=replica 

PrimaryPodDelete 

ReplicaPodDelete

RandomPvcDelete

PatroniSwitchOver