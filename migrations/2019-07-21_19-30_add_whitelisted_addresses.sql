-- +migrate Up
CREATE TABLE whitelisted_addresses (
  ip_address inet NOT NULL,
  instance_id integer NOT NULL REFERENCES instances (id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL,
  PRIMARY KEY (ip_address, instance_id)
);

-- +migrate Down
DROP TABLE whitelisted_addresses;
