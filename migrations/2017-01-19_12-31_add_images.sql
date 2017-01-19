-- +migrate Up
CREATE TABLE instances (
  id serial PRIMARY KEY,
  image_id integer NOT NULL,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL,

  FOREIGN KEY (image_id) REFERENCES images (id)
);

-- +migrate Down
DROP TABLE instances;
