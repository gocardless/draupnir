-- use createdb -o test test ?

CREATE user myapp_user WITH encrypted password '';
GRANT ALL privileges ON DATABASE test TO myapp_user;

CREATE DATABASE myapp OWNER myapp_user;

\c myapp

CREATE TABLE users (
  id SERIAL PRIMARY KEY,
  first_name TEXT,
  last_name TEXT,
  email TEXT UNIQUE NOT NULL
);

ALTER TABLE users OWNER TO myapp_user;
