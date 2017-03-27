-- +migrate Up
ALTER TABLE images ADD COLUMN anon text;

-- +migrate Down
ALTER TABLE images DROP COLUMN anon;
