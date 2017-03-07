-- +migrate Up
ALTER TABLE instances ADD COLUMN port integer NOT NULL;

-- +migrate Down
ALTER TABLE instances DROP COLUMN port integer;
