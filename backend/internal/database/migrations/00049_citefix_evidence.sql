-- +goose Up
CREATE TABLE go_audit_javascript_render_pages (
    audit_id text NOT NULL REFERENCES go_audits(id) ON DELETE CASCADE,
    page_url text NOT NULL,
    status text NOT NULL CHECK (status IN ('claimed', 'completed', 'failed')),
    cost_usd numeric(12, 6) NOT NULL DEFAULT 0 CHECK (cost_usd >= 0),
    completed_at timestamptz,
    PRIMARY KEY (audit_id, page_url)
);
ALTER TABLE go_audit_javascript_render_pages ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON TABLE go_audit_javascript_render_pages FROM PUBLIC;
-- +goose StatementBegin
DO $$
DECLARE client_role text;
BEGIN
    FOR client_role IN SELECT rolname FROM pg_roles WHERE rolname IN ('anon', 'authenticated') LOOP
        EXECUTE format('REVOKE ALL ON TABLE go_audit_javascript_render_pages FROM %I', client_role);
    END LOOP;
END $$;
-- +goose StatementEnd

CREATE TABLE go_citefix_rival_pages (
    organization_id text NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    page_url text NOT NULL,
    domain text NOT NULL,
    title text NOT NULL DEFAULT '',
    app_shell boolean NOT NULL DEFAULT false,
    fetched_at timestamptz NOT NULL,
    content_hash text NOT NULL,
    signals_json jsonb NOT NULL,
    PRIMARY KEY (project_id, page_url, content_hash)
);
CREATE INDEX go_citefix_rival_pages_latest_idx
    ON go_citefix_rival_pages(project_id, page_url, fetched_at DESC);

CREATE TABLE go_citefix_ai_mentions (
    organization_id text NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    fetched_at timestamptz NOT NULL,
    source text NOT NULL,
    result_json jsonb NOT NULL,
    PRIMARY KEY (project_id)
);

CREATE TABLE go_citefix_ai_overviews (
    id text PRIMARY KEY,
    organization_id text NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    prompt_hash text NOT NULL,
    prompt text NOT NULL,
    task_id text NOT NULL DEFAULT '',
    status text NOT NULL CHECK (status IN ('unknown', 'pending', 'completed', 'failed')),
    requested_at timestamptz NOT NULL,
    completed_at timestamptz,
    answer text NOT NULL DEFAULT '',
    citations_json jsonb NOT NULL DEFAULT '[]'::jsonb,
    UNIQUE (project_id, prompt_hash)
);
CREATE INDEX go_citefix_ai_overviews_latest_idx
    ON go_citefix_ai_overviews(project_id, requested_at DESC);

ALTER TABLE go_citefix_rival_pages ENABLE ROW LEVEL SECURITY;
ALTER TABLE go_citefix_ai_mentions ENABLE ROW LEVEL SECURITY;
ALTER TABLE go_citefix_ai_overviews ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON TABLE go_citefix_rival_pages FROM PUBLIC;
REVOKE ALL ON TABLE go_citefix_ai_mentions FROM PUBLIC;
REVOKE ALL ON TABLE go_citefix_ai_overviews FROM PUBLIC;
-- +goose StatementBegin
DO $$
DECLARE
    client_role text;
BEGIN
    FOR client_role IN SELECT rolname FROM pg_roles WHERE rolname IN ('anon', 'authenticated') LOOP
        EXECUTE format('REVOKE ALL ON TABLE go_citefix_rival_pages FROM %I', client_role);
        EXECUTE format('REVOKE ALL ON TABLE go_citefix_ai_mentions FROM %I', client_role);
        EXECUTE format('REVOKE ALL ON TABLE go_citefix_ai_overviews FROM %I', client_role);
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE go_citefix_ai_overviews;
DROP TABLE go_citefix_ai_mentions;
DROP TABLE go_citefix_rival_pages;
DROP TABLE go_audit_javascript_render_pages;
