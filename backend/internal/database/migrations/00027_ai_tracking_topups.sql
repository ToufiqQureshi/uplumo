-- +goose Up
CREATE TABLE go_ai_trackers (
    project_id text PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
    organization_id text NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    configuration jsonb NOT NULL,
    next_check_at timestamptz,
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX go_ai_trackers_due ON go_ai_trackers(next_check_at) WHERE next_check_at IS NOT NULL;
CREATE INDEX go_ai_trackers_org ON go_ai_trackers(organization_id);
CREATE TABLE go_ai_tracking_runs (
    id text PRIMARY KEY,
    project_id text NOT NULL REFERENCES go_ai_trackers(project_id) ON DELETE CASCADE,
    organization_id text NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    configuration jsonb NOT NULL,
    scope_key text NOT NULL,
    trigger text NOT NULL CHECK (trigger IN ('manual','scheduled')),
    status text NOT NULL DEFAULT 'queued' CHECK (status IN ('queued','running','completed','partial','failed')),
    error_code text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz,
    request_key text NOT NULL,
    UNIQUE(organization_id, project_id, request_key)
);
CREATE UNIQUE INDEX go_ai_tracking_active ON go_ai_tracking_runs(project_id) WHERE status IN ('queued','running');
CREATE INDEX go_ai_tracking_history ON go_ai_tracking_runs(project_id, created_at DESC);
CREATE TABLE go_ai_tracking_answers (
    id text PRIMARY KEY,
    run_id text NOT NULL REFERENCES go_ai_tracking_runs(id) ON DELETE CASCADE,
    prompt text NOT NULL,
    topic text NOT NULL,
    engine text NOT NULL,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','submitting','submitted','completed','failed')),
    task_id text NOT NULL DEFAULT '',
    answer text NOT NULL DEFAULT '',
    citations jsonb NOT NULL DEFAULT '[]',
    brands jsonb NOT NULL DEFAULT '[]',
    error_code text NOT NULL DEFAULT '',
    collected_at timestamptz,
    UNIQUE(run_id,prompt,engine)
);
CREATE INDEX go_ai_tracking_answers_run ON go_ai_tracking_answers(run_id);
CREATE TABLE go_billing_topups (
    id text PRIMARY KEY,
    organization_id text NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    pack_id text NOT NULL,
    amount_paise bigint NOT NULL CHECK(amount_paise > 0),
    credits bigint NOT NULL CHECK(credits > 0),
    expires_after_days integer NOT NULL CHECK(expires_after_days BETWEEN 1 AND 730),
    order_id text UNIQUE,
    payment_id text UNIQUE,
    status text NOT NULL DEFAULT 'creating' CHECK(status IN ('creating','ready','paid','failed')),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX go_billing_topups_pending ON go_billing_topups(organization_id) WHERE status IN ('creating','ready');

-- +goose Down
DROP TABLE go_billing_topups;
DROP TABLE go_ai_tracking_answers;
DROP TABLE go_ai_tracking_runs;
DROP TABLE go_ai_trackers;
