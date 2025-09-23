CREATE TABLE IF NOT EXISTS users (
    id uuid primary key,
    name text,
    alias_name text,
    email text,
    password text,
    created_at timestamp,
    updated_at timestamp

);
CREATE CUSTOM INDEX query_by_email_index ON chat.users(email) USING 'StorageAttachedIndex';