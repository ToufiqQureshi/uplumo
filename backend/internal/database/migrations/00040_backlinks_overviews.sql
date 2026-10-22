-- +goose Up
-- Persisted backlinks overview (summary + history) so a cache miss answers from
-- Postgres instead of buying the summary and history tasks again. A snapshot
-- older than backlinksOverviewMaxAge (30 days in Go) is ignored on read and
-- deleted on the next write.
CREATE TABLE go_backlinks_overviews (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    organization_id text NOT NULL REFERENCES organization (id) ON DELETE CASCADE,
    api_target text NOT NULL,
    scope text NOT NULL,
    payload jsonb NOT NULL,
    fetched_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX go_backlinks_overviews_identity_idx
    ON go_backlinks_overviews (organization_id, api_target, scope);

-- Only the trusted server accesses this table. Browser and Data API roles get
-- no policies even with table-level grants.
ALTER TABLE go_backlinks_overviews ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON go_backlinks_overviews FROM PUBLIC;
-- +goose StatementBegin
DO $$
DECLARE client_role text;
BEGIN
    FOR client_role IN SELECT rolname FROM pg_roles WHERE rolname IN ('anon', 'authenticated') LOOP
        EXECUTE format('REVOKE ALL ON go_backlinks_overviews FROM %I', client_role);
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE go_backlinks_overviews;
