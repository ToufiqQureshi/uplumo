-- +goose Up
-- Store lifetime totals only. There is no individual exit-survey record.
CREATE TABLE go_account_deletion_reason_counts (
    reason text PRIMARY KEY CHECK (reason IN (
        'just_testing', 'no_longer_needed', 'too_expensive', 'missing_feature',
        'hard_to_use', 'technical_problem', 'other'
    )),
    total bigint NOT NULL CHECK (total > 0)
);

ALTER TABLE go_account_deletion_reason_counts ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON TABLE go_account_deletion_reason_counts FROM PUBLIC;
-- +goose StatementBegin
DO $$
DECLARE client_role text;
BEGIN
    FOR client_role IN SELECT rolname FROM pg_roles WHERE rolname IN ('anon', 'authenticated') LOOP
        EXECUTE format('REVOKE ALL ON TABLE go_account_deletion_reason_counts FROM %I', client_role);
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE go_account_deletion_reason_counts;
