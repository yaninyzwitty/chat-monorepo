-- Create a keyspace for testing
CREATE KEYSPACE IF NOT EXISTS test_keyspace
WITH replication = {
  'class': 'SimpleStrategy',
  'replication_factor': '1'
};

USE test_keyspace;

-- Create a simple table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    name TEXT,
    email TEXT
);

-- Insert some sample rows
INSERT INTO users (id, name, email)
VALUES (uuid(), 'Alice', 'alice@example.com');

INSERT INTO users (id, name, email)
VALUES (uuid(), 'Bob', 'bob@example.com');
