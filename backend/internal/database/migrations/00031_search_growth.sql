-- +goose Up
CREATE TABLE go_search_growth_reports (
 project_id text PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
 organization_id text NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
 cache_key text NOT NULL,
 report jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE go_search_growth_actions (
 id text PRIMARY KEY,
 project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
 organization_id text NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
 page text NOT NULL CHECK(length(page) BETWEEN 1 AND 2048),
 note text NOT NULL CHECK(length(note) BETWEEN 1 AND 1000),
 status text NOT NULL DEFAULT 'planned' CHECK(status IN ('planned','applied','dismissed')),
 baseline jsonb NOT NULL,
 source jsonb NOT NULL,
 baseline_range jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 applied_at timestamptz,
 UNIQUE(project_id,page)
);
CREATE INDEX go_search_growth_actions_scope ON go_search_growth_actions(organization_id,project_id,created_at DESC);
-- +goose Down
DROP TABLE go_search_growth_actions;
DROP TABLE go_search_growth_reports;
