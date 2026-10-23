-- +goose Up
-- One saved card layout per user per project: which dashboard cards show and
-- in what order. Server-only storage, like every other table (migration 42).
CREATE TABLE go_dashboard_layouts (
    user_id text NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,
    project_id text NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    cards jsonb NOT NULL CHECK (jsonb_typeof(cards) = 'array'),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, project_id)
);

CREATE INDEX go_dashboard_layouts_project_idx ON go_dashboard_layouts (project_id);

ALTER TABLE go_dashboard_layouts ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON TABLE go_dashboard_layouts FROM PUBLIC;
-- +goose StatementBegin
DO $$
DECLARE
    client_role text;
BEGIN
    FOR client_role IN SELECT rolname FROM pg_roles WHERE rolname IN ('anon', 'authenticated') LOOP
        EXECUTE format('REVOKE ALL ON TABLE go_dashboard_layouts FROM %I', client_role);
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE go_dashboard_layouts;
