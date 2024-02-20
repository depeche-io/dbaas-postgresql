#!/bin/bash

FIRST_RUNNING_POD=$(kubectl get pods -n part1-diy  | grep Running | cut -f 1 -d ' ' | head -n 1)

kubectl exec -it -n part1-diy $FIRST_RUNNING_POD -- pg_dump -U ps_user -d ps_db > db_backup.sql

echo "See: ./db_backup.sql"

