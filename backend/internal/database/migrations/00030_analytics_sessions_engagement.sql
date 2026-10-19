-- +goose Up
ALTER TABLE go_analytics_events DROP CONSTRAINT go_analytics_events_channel_check;
ALTER TABLE go_analytics_events ADD CONSTRAINT go_analytics_events_channel_check
    CHECK (channel IN ('ai', 'search', 'social', 'paid', 'email', 'direct', 'referral', 'internal'));

ALTER TABLE go_analytics_events
    ADD COLUMN pageview_hash bytea CHECK (pageview_hash IS NULL OR length(pageview_hash) = 32),
    ADD COLUMN session_hash bytea CHECK (session_hash IS NULL OR length(session_hash) = 32),
    ADD COLUMN engaged_ms bigint NOT NULL DEFAULT 0 CHECK (engaged_ms BETWEEN 0 AND 86400000);

CREATE UNIQUE INDEX go_analytics_events_site_pageview
    ON go_analytics_events (site_id, pageview_hash) WHERE pageview_hash IS NOT NULL;
CREATE INDEX go_analytics_events_site_session_time
    ON go_analytics_events (site_id, session_hash, occurred_at, pageview_hash) WHERE session_hash IS NOT NULL;

ALTER TABLE go_analytics_custom_events
    ADD COLUMN event_hash bytea CHECK (event_hash IS NULL OR length(event_hash) = 32),
    ADD COLUMN session_hash bytea CHECK (session_hash IS NULL OR length(session_hash) = 32);

CREATE UNIQUE INDEX go_analytics_custom_events_site_event
    ON go_analytics_custom_events (site_id, event_hash) WHERE event_hash IS NOT NULL;
CREATE INDEX go_analytics_custom_events_site_session_time
    ON go_analytics_custom_events (site_id, occurred_at, session_hash) WHERE session_hash IS NOT NULL;

-- +goose Down
DROP INDEX go_analytics_custom_events_site_session_time;
DROP INDEX go_analytics_custom_events_site_event;
ALTER TABLE go_analytics_custom_events DROP COLUMN event_hash, DROP COLUMN session_hash;
DROP INDEX go_analytics_events_site_session_time;
DROP INDEX go_analytics_events_site_pageview;
ALTER TABLE go_analytics_events DROP COLUMN pageview_hash, DROP COLUMN session_hash, DROP COLUMN engaged_ms;
UPDATE go_analytics_events SET channel = 'direct' WHERE channel IN ('paid', 'email');
ALTER TABLE go_analytics_events DROP CONSTRAINT go_analytics_events_channel_check;
ALTER TABLE go_analytics_events ADD CONSTRAINT go_analytics_events_channel_check
    CHECK (channel IN ('ai', 'search', 'social', 'direct', 'referral', 'internal'));
