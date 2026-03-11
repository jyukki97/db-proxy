#!/bin/bash
set -e

# Wait for primary to be ready
until pg_isready -h primary -p 5432 -U postgres; do
  echo "Waiting for primary..."
  sleep 1
done

# Clean data directory and create base backup
rm -rf /var/lib/postgresql/data/*
pg_basebackup -h primary -p 5432 -U replicator -D /var/lib/postgresql/data -Fp -Xs -P -R

# Ensure proper permissions
chmod 700 /var/lib/postgresql/data

# Start postgres in replica mode
exec postgres -c hot_standby=on
