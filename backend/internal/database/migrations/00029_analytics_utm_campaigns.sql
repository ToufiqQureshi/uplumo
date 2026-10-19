-- +goose Up
ALTER TABLE go_analytics_events
    ADD COLUMN utm_source text CHECK (utm_source IS NULL OR length(utm_source) <= 100),
    ADD COLUMN utm_medium text CHECK (utm_medium IS NULL OR length(utm_medium) <= 100),
    ADD COLUMN utm_campaign text CHECK (utm_campaign IS NULL OR length(utm_campaign) <= 100),
    ADD COLUMN utm_term text CHECK (utm_term IS NULL OR length(utm_term) <= 100),
    ADD COLUMN utm_content text CHECK (utm_content IS NULL OR length(utm_content) <= 100);

-- +goose Down
ALTER TABLE go_analytics_events
    DROP COLUMN utm_source,
    DROP COLUMN utm_medium,
    DROP COLUMN utm_campaign,
    DROP COLUMN utm_term,
    DROP COLUMN utm_content;
