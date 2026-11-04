-- +goose Up
-- The custom event names a project owner counts as leads. Analytics no longer
-- guesses leads from event names: the owner picks them.
CREATE TABLE go_analytics_lead_events (
 site_id bigint NOT NULL REFERENCES go_analytics_sites(id) ON DELETE CASCADE,
 name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 64),
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (site_id, name)
);
ALTER TABLE go_analytics_lead_events ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON TABLE go_analytics_lead_events FROM PUBLIC;
-- +goose StatementBegin
DO $$
DECLARE client_role text;
BEGIN
 FOR client_role IN SELECT rolname FROM pg_roles WHERE rolname IN ('anon','authenticated') LOOP
  EXECUTE format('REVOKE ALL ON TABLE go_analytics_lead_events FROM %I',client_role);
 END LOOP;
END $$;
-- +goose StatementEnd
-- +goose Down
DROP TABLE go_analytics_lead_events;
