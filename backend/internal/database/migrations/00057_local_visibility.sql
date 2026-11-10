-- +goose Up
-- Local Visibility: a project's Google Business Profile, the paid checks run
-- on it (profile vs top 3, map rank grid, reviews, weekly competitor watch),
-- the DataForSEO tasks each check posted, and saved review reply drafts.

-- The business a project is about, picked by the customer from a Maps search.
CREATE TABLE go_local_businesses (
    project_id text PRIMARY KEY REFERENCES projects (id) ON DELETE CASCADE,
    organization_id text NOT NULL,
    cid text NOT NULL CHECK (cid ~ '^[0-9]{1,40}$'),
    place_id text NOT NULL DEFAULT '',
    title text NOT NULL,
    address text NOT NULL DEFAULT '',
    category text NOT NULL DEFAULT '',
    latitude double precision NOT NULL CHECK (latitude BETWEEN -90 AND 90),
    longitude double precision NOT NULL CHECK (longitude BETWEEN -180 AND 180),
    -- What customers search on Maps to find a business like this one.
    search_term text NOT NULL CHECK (search_term <> '' AND char_length(search_term) <= 80),
    language_code text NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- One paid check. A finished run is the cached answer for the same input for
-- 24 hours; only one run per project and kind can be running at a time.
CREATE TABLE go_local_runs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id text NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    organization_id text NOT NULL,
    kind text NOT NULL CHECK (kind IN ('find', 'check', 'grid', 'reviews', 'watch')),
    input_hash text NOT NULL,
    input jsonb NOT NULL,
    status text NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'done', 'failed')),
    result jsonb,
    error text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz
);
CREATE UNIQUE INDEX go_local_runs_one_running ON go_local_runs (project_id, kind) WHERE status = 'running';
CREATE INDEX go_local_runs_latest ON go_local_runs (project_id, kind, created_at DESC);

-- Every DataForSEO task a run posted, stored the moment it is accepted, so a
-- retried job collects it instead of paying for it again.
CREATE TABLE go_local_tasks (
    run_id uuid NOT NULL REFERENCES go_local_runs (id) ON DELETE CASCADE,
    task_key text NOT NULL,
    endpoint text NOT NULL,
    task_id text NOT NULL,
    state text NOT NULL DEFAULT 'posted' CHECK (state IN ('posted', 'done', 'failed')),
    result jsonb,
    PRIMARY KEY (run_id, task_key)
);

-- Weekly competitor watch, off until the customer turns it on.
CREATE TABLE go_local_watch (
    project_id text PRIMARY KEY REFERENCES projects (id) ON DELETE CASCADE,
    organization_id text NOT NULL,
    enabled boolean NOT NULL DEFAULT false,
    next_run_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- The last reply drafted for a review, so reopening the page costs nothing.
CREATE TABLE go_local_reply_drafts (
    project_id text NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    review_id text NOT NULL,
    reply text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, review_id)
);

ALTER TABLE go_local_businesses ENABLE ROW LEVEL SECURITY;
ALTER TABLE go_local_runs ENABLE ROW LEVEL SECURITY;
ALTER TABLE go_local_tasks ENABLE ROW LEVEL SECURITY;
ALTER TABLE go_local_watch ENABLE ROW LEVEL SECURITY;
ALTER TABLE go_local_reply_drafts ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON TABLE go_local_businesses, go_local_runs, go_local_tasks, go_local_watch, go_local_reply_drafts FROM PUBLIC;
-- +goose StatementBegin
DO $$
DECLARE
    client_role text;
BEGIN
    FOR client_role IN SELECT rolname FROM pg_roles WHERE rolname IN ('anon', 'authenticated') LOOP
        EXECUTE format('REVOKE ALL ON TABLE go_local_businesses, go_local_runs, go_local_tasks, go_local_watch, go_local_reply_drafts FROM %I', client_role);
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE go_local_reply_drafts;
DROP TABLE go_local_watch;
DROP TABLE go_local_tasks;
DROP TABLE go_local_runs;
DROP TABLE go_local_businesses;
