#!/bin/bash

# Loop 50 times to spawn 50 independent workers
for ((i=0; i<50; i++))
do
    # Run an infinite loop inside a subshell, pushed to the background
    (
        while true; do
            # Using single quotes for -d and escaping to let Bash evaluate $RANDOM dynamically
            curl -X POST http://localhost:8080/logs \
                 -H "Content-Type: application/json" \
                 -m 2 \
                 -d '{"service": {"name": "Database", "id": '$RANDOM'}, "status": "ERROR : Connection Refused"}' \
                 -s -o /dev/null
        done
    ) &
done

echo "Firehose started. Hitting target..."
wait
