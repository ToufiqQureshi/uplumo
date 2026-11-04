-- +goose Up
-- Cite Fix outreach pitch: one paid AI draft asking a list page's owner to
-- add the customer. The row is the idempotency record (a retried request
-- never pays twice), the lock (one running draft per page) and the cache
-- (a page already pitched is shown again for 0 credits).
CREATE TABLE go_citefix_pitches (
    id text PRIMARY KEY,
    organization_id text NOT NULL REFERENCES organization (id) ON DELETE CASCADE,
    project_id text NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    page_url text NOT NULL CHECK (char_length(page_url) BETWEEN 1 AND 2048),
    idempotency_key text NOT NULL CHECK (char_length(idempotency_key) BETWEEN 1 AND 128),
    status text NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'completed', 'failed')),
    credits bigint NOT NULL DEFAULT 0 CHECK (credits >= 0),
    subject text NOT NULL DEFAULT '',
    body text NOT NULL DEFAULT '',
    error_code text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz,
    UNIQUE (organization_id, idempotency_key)
);

CREATE UNIQUE INDEX go_citefix_pitches_one_running ON go_citefix_pitches (project_id, page_url) WHERE status = 'running';
CREATE INDEX go_citefix_pitches_done ON go_citefix_pitches (project_id, page_url, created_at DESC) WHERE status = 'completed';

ALTER TABLE go_citefix_pitches ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON TABLE go_citefix_pitches FROM PUBLIC;
-- +goose StatementBegin
DO $$
DECLARE
    client_role text;
BEGIN
    FOR client_role IN SELECT rolname FROM pg_roles WHERE rolname IN ('anon', 'authenticated') LOOP
        EXECUTE format('REVOKE ALL ON TABLE go_citefix_pitches FROM %I', client_role);
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE go_citefix_pitches;
