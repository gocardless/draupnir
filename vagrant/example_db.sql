-- use createdb -o test test ?

CREATE DATABASE test;

CREATE user test WITH encrypted password '';
GRANT ALL privileges ON DATABASE test TO test;

\c test

CREATE TABLE users (
  id SERIAL PRIMARY KEY,
  first_name TEXT,
  last_name TEXT,
  email TEXT UNIQUE NOT NULL
);
