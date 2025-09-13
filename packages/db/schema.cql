CREATE TABLE IF NOT EXISTS users (
    id uuid primary key,
    name text,
    alias_name text,
    created_at timestamp,
    updated_at timestamp

);