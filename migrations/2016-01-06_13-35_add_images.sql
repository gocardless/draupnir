-- +migrate Up
CREATE TABLE images (
  id serial PRIMARY KEY,
  backed_up_at timestamptz NOT NULL,
  ready boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL
);

-- +migrate Down
DROP TABLE images;
