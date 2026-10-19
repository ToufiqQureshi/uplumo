-- +goose Up
CREATE TABLE go_analytics_enquiry_goals (
 project_id text PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
 event_name text NOT NULL CHECK(length(event_name) BETWEEN 1 AND 64),
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE go_analytics_enquiry_changes (
 id text PRIMARY KEY,
 project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
 path text NOT NULL,
 event_name text NOT NULL,
 note text NOT NULL,
 baseline jsonb NOT NULL,
 baseline_start timestamptz NOT NULL,
 baseline_end timestamptz NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 request_key text NOT NULL,
 UNIQUE(project_id,request_key)
);
CREATE INDEX go_analytics_enquiry_changes_project ON go_analytics_enquiry_changes(project_id,created_at DESC);
CREATE INDEX go_analytics_custom_goal_time ON go_analytics_custom_events(site_id,name,occurred_at);
-- +goose Down
DROP INDEX go_analytics_custom_goal_time;
DROP TABLE go_analytics_enquiry_changes;
DROP TABLE go_analytics_enquiry_goals;
