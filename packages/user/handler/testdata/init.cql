-- Create keyspace
CREATE KEYSPACE IF NOT EXISTS chat
WITH replication = {
  'class': 'SimpleStrategy',
  'replication_factor': 1
};

-- Use keyspace
USE chat;

-- Force drop (test containers can reuse volumes)
DROP TABLE IF EXISTS users;

-- Recreate fresh schema including password
CREATE TABLE users (
    id UUID PRIMARY KEY,
    name text,
    alias_name text,
    email text,
    password text,         -- ✅ REQUIRED for tests
    created_at timestamp,
    updated_at timestamp
);
