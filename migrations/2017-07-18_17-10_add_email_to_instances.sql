-- +migrate Up
ALTER TABLE instances ADD COLUMN user_email text;

-- +migrate Down
ALTER TABLE instances DROP COLUMN user_email;
