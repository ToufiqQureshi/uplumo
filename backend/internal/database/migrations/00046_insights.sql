-- +goose Up
-- Next Actions: which ranked actions a project's team marked done or ignored.
-- A row only exists for a decision; "to do" is the absence of a row. Shared by
-- the project's members. Server-only storage, like every other table
-- (migration 42).
CREATE TABLE go_next_action_states (
    project_id text NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    organization_id text NOT NULL REFERENCES organization (id) ON DELETE CASCADE,
    action_key text NOT NULL CHECK (char_length(action_key) BETWEEN 1 AND 120),
    status text NOT NULL CHECK (status IN ('done', 'ignored')),
    updated_by text NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, action_key)
);

CREATE INDEX go_next_action_states_org_idx ON go_next_action_states (organization_id);

ALTER TABLE go_next_action_states ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON TABLE go_next_action_states FROM PUBLIC;
-- +goose StatementBegin
DO $$
DECLARE
    client_role text;
BEGIN
    FOR client_role IN SELECT rolname FROM pg_roles WHERE rolname IN ('anon', 'authenticated') LOOP
        EXECUTE format('REVOKE ALL ON TABLE go_next_action_states FROM %I', client_role);
    END LOOP;
END $$;
-- +goose StatementEnd

-- AI Sentiment: how one saved AI answer speaks about the tracked brand,
-- classified once on request and kept with the answer it describes. Empty
-- means not classified yet.
ALTER TABLE go_ai_tracking_answers
    ADD COLUMN sentiment text NOT NULL DEFAULT '' CHECK (sentiment IN ('', 'positive', 'neutral', 'negative', 'not_mentioned')),
    ADD COLUMN sentiment_quote text NOT NULL DEFAULT '' CHECK (char_length(sentiment_quote) <= 400),
    ADD COLUMN sentiment_reason text NOT NULL DEFAULT '' CHECK (char_length(sentiment_reason) <= 400);

-- +goose Down
ALTER TABLE go_ai_tracking_answers
    DROP COLUMN sentiment_reason,
    DROP COLUMN sentiment_quote,
    DROP COLUMN sentiment;
DROP TABLE go_next_action_states;
