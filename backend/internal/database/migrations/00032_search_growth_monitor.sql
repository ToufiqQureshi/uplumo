-- +goose Up
CREATE TABLE go_search_growth_monitors (
 project_id text PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
 organization_id text NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
 enabled boolean NOT NULL DEFAULT false,
 goal_event text NOT NULL DEFAULT '' CHECK(length(goal_event)<=40),
 property_id text NOT NULL,
 site_url text NOT NULL,
 revision bigint NOT NULL DEFAULT 1,
 next_check_at timestamptz NOT NULL DEFAULT now(),
 checked_at timestamptz,
 digest jsonb,
 last_error text,
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX go_search_growth_monitors_due ON go_search_growth_monitors(next_check_at) WHERE enabled;
CREATE INDEX go_search_growth_monitors_scope ON go_search_growth_monitors(organization_id,project_id);
ALTER TABLE go_search_growth_monitors ENABLE ROW LEVEL SECURITY;
-- +goose Down
DROP TABLE go_search_growth_monitors;
