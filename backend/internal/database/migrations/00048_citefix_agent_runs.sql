-- +goose Up
-- Cite Fix agent: one paid AI run on one page. The row is the idempotency
-- record (a retried request with the same key never pays twice), the lock
-- (one running run per page and mode), and the cache (the same page content
-- in the same mode is served again for 0 credits).
CREATE TABLE go_citefix_agent_runs (
    id text PRIMARY KEY,
    organization_id text NOT NULL REFERENCES organization (id) ON DELETE CASCADE,
    project_id text NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    url text NOT NULL CHECK (char_length(url) BETWEEN 1 AND 2048),
    mode text NOT NULL CHECK (mode IN ('quick', 'deep')),
    idempotency_key text NOT NULL CHECK (char_length(idempotency_key) BETWEEN 1 AND 128),
    content_hash text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'completed', 'failed')),
    credits bigint NOT NULL DEFAULT 0 CHECK (credits >= 0),
    -- cached rows replay an earlier answer for free; they do not count
    -- towards the project's run limit.
    cached boolean NOT NULL DEFAULT false,
    result_json text,
    error_code text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz,
    UNIQUE (organization_id, idempotency_key)
);

CREATE UNIQUE INDEX go_citefix_agent_runs_one_running ON go_citefix_agent_runs (project_id, url, mode) WHERE status = 'running';
CREATE INDEX go_citefix_agent_runs_cache ON go_citefix_agent_runs (project_id, url, mode, content_hash) WHERE status = 'completed';
CREATE INDEX go_citefix_agent_runs_project ON go_citefix_agent_runs (project_id, created_at DESC);

ALTER TABLE go_citefix_agent_runs ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON TABLE go_citefix_agent_runs FROM PUBLIC;
-- +goose StatementBegin
DO $$
DECLARE
    client_role text;
BEGIN
    FOR client_role IN SELECT rolname FROM pg_roles WHERE rolname IN ('anon', 'authenticated') LOOP
        EXECUTE format('REVOKE ALL ON TABLE go_citefix_agent_runs FROM %I', client_role);
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE go_citefix_agent_runs;
