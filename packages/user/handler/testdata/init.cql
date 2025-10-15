-- Create keyspace
CREATE KEYSPACE IF NOT EXISTS chat
WITH replication = {
  'class': 'SimpleStrategy',
  'replication_factor': 1
};

-- Use keyspace
USE init_sh_keyspace;

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    name text,
    alias_name text,
    email text,
    created_at timestamp,
    updated_at timestamp
);