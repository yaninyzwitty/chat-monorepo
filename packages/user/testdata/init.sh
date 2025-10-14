#!/bin/bash
set -e

echo "🚀 Initializing Cassandra keyspace and users table..."

until cqlsh -e "DESCRIBE KEYSPACES" >/dev/null 2>&1; do
  echo "⌛ Waiting for Cassandra to be ready..."
  sleep 2
done

cqlsh <<EOF
CREATE KEYSPACE IF NOT EXISTS init_sh_keyspace
WITH REPLICATION = {'class': 'SimpleStrategy', 'replication_factor': 1};

CREATE TABLE IF NOT EXISTS init_sh_keyspace.users (
    id UUID PRIMARY KEY,
    name TEXT,
    alias_name TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    email TEXT,
    password TEXT
);
EOF

echo "✅ Cassandra init complete."
