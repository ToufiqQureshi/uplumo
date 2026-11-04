-- +goose Up
-- A blocked user cannot sign in or use a session, API key or MCP token.
-- Deleting the user removes the block with them.
CREATE TABLE go_user_blocks (
 user_id text PRIMARY KEY REFERENCES "user"(id) ON DELETE CASCADE,
 blocked_by text NOT NULL,
 reason text NOT NULL DEFAULT '' CHECK (char_length(reason) <= 500),
 blocked_at timestamptz NOT NULL DEFAULT now()
);
-- Who did what in the staff console. target_user_id stays after an erasure
-- (opaque id, no email), so the log survives the user it describes.
CREATE TABLE go_staff_actions (
 id bigserial PRIMARY KEY,
 actor_user_id text NOT NULL,
 action text NOT NULL CHECK (action IN ('block','unblock','delete','sync_payments')),
 target_user_id text NOT NULL DEFAULT '',
 detail text NOT NULL DEFAULT '',
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX go_staff_actions_created ON go_staff_actions(created_at DESC);
ALTER TABLE go_user_blocks ENABLE ROW LEVEL SECURITY;
ALTER TABLE go_staff_actions ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON TABLE go_user_blocks, go_staff_actions FROM PUBLIC;
-- +goose StatementBegin
DO $$
DECLARE client_role text;
BEGIN
 FOR client_role IN SELECT rolname FROM pg_roles WHERE rolname IN ('anon','authenticated') LOOP
  EXECUTE format('REVOKE ALL ON TABLE go_user_blocks, go_staff_actions FROM %I',client_role);
 END LOOP;
END $$;
-- +goose StatementEnd
-- +goose Down
DROP TABLE go_staff_actions;
DROP TABLE go_user_blocks;
