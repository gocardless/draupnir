-- +migrate Up
ALTER TABLE instances ADD COLUMN refresh_token text;

-- +migrate Down
ALTER TABLE instances DROP COLUMN refresh_token;
