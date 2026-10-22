-- +goose Up
-- Persisted SERP snapshots so a cache miss answers from Postgres instead of
-- buying the crawl again. A snapshot older than serpSnapshotMaxAge (7 days in
-- Go) is ignored on read and deleted on the next write.
CREATE TABLE go_keyword_serp_snapshots (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    project_id text NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    keyword text NOT NULL,
    location_code integer NOT NULL,
    location_name text NOT NULL DEFAULT '',
    language_code text NOT NULL,
    depth integer NOT NULL,
    items jsonb NOT NULL,
    fetched_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX go_keyword_serp_snapshots_identity_idx
    ON go_keyword_serp_snapshots (project_id, keyword, location_code, location_name, language_code, depth);

-- Every write deletes expired rows (fetched_at < now - max age); without this
-- index that delete scans the whole table.
CREATE INDEX go_keyword_serp_snapshots_fetched_at_idx
    ON go_keyword_serp_snapshots (fetched_at);

-- Only the trusted server accesses this table. Browser and Data API roles get
-- no policies even with table-level grants.
ALTER TABLE go_keyword_serp_snapshots ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON go_keyword_serp_snapshots FROM PUBLIC;
-- +goose StatementBegin
DO $$
DECLARE client_role text;
BEGIN
    FOR client_role IN SELECT rolname FROM pg_roles WHERE rolname IN ('anon', 'authenticated') LOOP
        EXECUTE format('REVOKE ALL ON go_keyword_serp_snapshots FROM %I', client_role);
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE go_keyword_serp_snapshots;
