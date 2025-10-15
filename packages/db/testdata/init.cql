-- Create keyspace
CREATE KEYSPACE IF NOT EXISTS chat
WITH replication = {
  'class': 'SimpleStrategy',
  'replication_factor': '1'
};

USE chat;

-- Users table matching Go handler
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    name TEXT,
    alias_name TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    email TEXT,
    password TEXT
);

-- Optional: Insert some seed rows
INSERT INTO users (id, name, alias_name, created_at, updated_at, email, password)
VALUES (uuid(), 'Alice', 'ali', toTimestamp(now()), toTimestamp(now()), 'alice@example.com', 'pass123');

INSERT INTO users (id, name, alias_name, created_at, updated_at, email, password)
VALUES (uuid(), 'Bob', 'bobby', toTimestamp(now()), toTimestamp(now()), 'bob@example.com', 'pass456');
