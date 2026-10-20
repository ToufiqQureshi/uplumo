-- +goose Up
CREATE TABLE go_analytics_browsers (
    site_id bigint NOT NULL REFERENCES go_analytics_sites(id) ON DELETE CASCADE,
    browser_hash bytea NOT NULL CHECK (length(browser_hash) = 32),
    first_seen timestamptz NOT NULL,
    PRIMARY KEY (site_id, browser_hash)
);
ALTER TABLE go_analytics_browsers ENABLE ROW LEVEL SECURITY;
-- These existing server-owned tables now carry browser recognition and link
-- destinations. Protect them from default Supabase Data API grants as well.
ALTER TABLE go_analytics_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE go_analytics_custom_events ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON go_analytics_browsers, go_analytics_events, go_analytics_custom_events FROM PUBLIC;
-- +goose StatementBegin
DO $$
DECLARE client_role text;
BEGIN
    FOR client_role IN SELECT rolname FROM pg_roles WHERE rolname IN ('anon', 'authenticated') LOOP
        EXECUTE format('REVOKE ALL ON go_analytics_browsers, go_analytics_events, go_analytics_custom_events FROM %I', client_role);
    END LOOP;
END $$;
-- +goose StatementEnd

ALTER TABLE go_analytics_events
    ADD COLUMN browser_hash bytea CHECK (browser_hash IS NULL OR length(browser_hash) = 32),
    ADD COLUMN page_host text;
CREATE INDEX go_analytics_events_site_browser_time
    ON go_analytics_events (site_id, browser_hash, occurred_at) WHERE browser_hash IS NOT NULL;

ALTER TABLE go_analytics_custom_events
    ADD COLUMN is_click boolean NOT NULL DEFAULT false,
    ADD COLUMN pageview_hash bytea CHECK (pageview_hash IS NULL OR length(pageview_hash) = 32),
    ADD COLUMN destination_host text,
    ADD COLUMN destination_path text,
    ADD COLUMN destination_type text NOT NULL DEFAULT 'unknown'
        CHECK (destination_type IN ('internal', 'external', 'unknown'));
CREATE INDEX go_analytics_custom_events_site_click_time
    ON go_analytics_custom_events (site_id, occurred_at) WHERE is_click;

-- +goose Down
DROP INDEX go_analytics_custom_events_site_click_time;
ALTER TABLE go_analytics_custom_events DROP COLUMN is_click, DROP COLUMN pageview_hash,
    DROP COLUMN destination_host, DROP COLUMN destination_path, DROP COLUMN destination_type;
DROP INDEX go_analytics_events_site_browser_time;
ALTER TABLE go_analytics_events DROP COLUMN browser_hash, DROP COLUMN page_host;
DROP TABLE go_analytics_browsers;
ALTER TABLE go_analytics_events DISABLE ROW LEVEL SECURITY;
ALTER TABLE go_analytics_custom_events DISABLE ROW LEVEL SECURITY;
