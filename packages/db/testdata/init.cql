-- Create keyspace
CREATE KEYSPACE IF NOT EXISTS chat
WITH replication = {
  'class': 'SimpleStrategy',
  'replication_factor': '1'
};

USE chat;

-- Users table matching Go handler expectations
DROP TABLE IF EXISTS users;

CREATE TABLE users (
    id UUID PRIMARY KEY,
    name TEXT,
    alias_name TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    email TEXT,
    password TEXT
);
