-- +goose Up
CREATE TABLE go_support_reports (
 id text PRIMARY KEY,
 organization_id text NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
 user_id text NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
 submission_id text NOT NULL,
 payload_hash bytea NOT NULL,
 kind text NOT NULL CHECK (kind IN ('bug','billing','suggestion')),
 description text NOT NULL CHECK (char_length(description) BETWEEN 10 AND 5000),
 page_path text NOT NULL,
 app_version text NOT NULL,
 request_id text NOT NULL,
 screenshot bytea CHECK (octet_length(screenshot)<=1048576),
 screenshot_mime text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 notified_at timestamptz,
 notification_attempts integer NOT NULL DEFAULT 0,
 next_notification_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE (organization_id,user_id,submission_id)
);
CREATE INDEX go_support_reports_user_created ON go_support_reports(user_id,created_at);
CREATE INDEX go_support_reports_notification_due ON go_support_reports(next_notification_at) WHERE notified_at IS NULL AND notification_attempts<10;
ALTER TABLE go_support_reports ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON TABLE go_support_reports FROM PUBLIC;
-- +goose StatementBegin
DO $$
DECLARE client_role text;
BEGIN
 FOR client_role IN SELECT rolname FROM pg_roles WHERE rolname IN ('anon','authenticated') LOOP
  EXECUTE format('REVOKE ALL ON TABLE go_support_reports FROM %I',client_role);
 END LOOP;
END $$;
-- +goose StatementEnd
-- +goose Down
DROP TABLE go_support_reports;
