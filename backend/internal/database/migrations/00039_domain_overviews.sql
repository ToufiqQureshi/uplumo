-- +goose Up
-- Persisted domain overview so a cache miss answers from Postgres instead of
-- buying the Labs overview again. A snapshot older than domainOverviewMaxAge
-- (30 days in Go) is ignored on read and deleted on the next write.
CREATE TABLE go_domain_overviews (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    organization_id text NOT NULL REFERENCES organization (id) ON DELETE CASCADE,
    project_id text NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    hostname text NOT NULL,
    location_code integer NOT NULL,
    language_code text NOT NULL,
    payload jsonb NOT NULL,
    fetched_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX go_domain_overviews_identity_idx
    ON go_domain_overviews (organization_id, project_id, hostname, location_code, language_code);

-- Only the trusted server accesses this table. Browser and Data API roles get
-- no policies even with table-level grants.
ALTER TABLE go_domain_overviews ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON go_domain_overviews FROM PUBLIC;
-- +goose StatementBegin
DO $$
DECLARE client_role text;
BEGIN
    FOR client_role IN SELECT rolname FROM pg_roles WHERE rolname IN ('anon', 'authenticated') LOOP
        EXECUTE format('REVOKE ALL ON go_domain_overviews FROM %I', client_role);
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE go_domain_overviews;
