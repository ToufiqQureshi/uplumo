-- +goose Up
-- One IndexNow key per project. The key is not a secret: the customer hosts
-- it at https://{host}/{key}.txt to prove they own the site.
CREATE TABLE go_indexnow_keys (
    project_id text PRIMARY KEY REFERENCES projects (id) ON DELETE CASCADE,
    key text NOT NULL CHECK (key ~ '^[0-9a-f]{32}$'),
    created_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE go_indexnow_keys;
