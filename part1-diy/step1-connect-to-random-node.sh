#!/bin/bash

FIRST_RUNNING_POD=$(kubectl get pods -n part1-diy  | grep Running | cut -f 1 -d ' ' | head -n 1)

echo "Password is 'SecurePassword'"

kubectl exec -it -n part1-diy $FIRST_RUNNING_POD -- psql -h localhost -U ps_user --password -p 5432 ps_db
