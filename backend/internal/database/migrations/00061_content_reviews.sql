-- +goose Up
CREATE TABLE go_content_reviews (
    organization_id text NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    input_hash text NOT NULL CHECK (length(input_hash) = 64),
    report text NOT NULL CHECK (report <> ''),
    expires_at timestamptz NOT NULL,
    PRIMARY KEY (organization_id, input_hash)
);
ALTER TABLE go_content_reviews ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON go_content_reviews FROM PUBLIC;
-- +goose StatementBegin
DO $$
DECLARE client_role text;
BEGIN
    FOR client_role IN SELECT rolname FROM pg_roles WHERE rolname IN ('anon', 'authenticated') LOOP
        EXECUTE format('REVOKE ALL ON TABLE go_content_reviews FROM %I', client_role);
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE go_content_reviews;
