-- +goose Up
-- Only the trusted server/migration owner accesses these tables. Browser and
-- Data API roles receive no policies, even if they have table-level grants.
ALTER TABLE go_search_growth_reports ENABLE ROW LEVEL SECURITY;
ALTER TABLE go_search_growth_actions ENABLE ROW LEVEL SECURITY;

-- +goose Down
ALTER TABLE go_search_growth_actions DISABLE ROW LEVEL SECURITY;
ALTER TABLE go_search_growth_reports DISABLE ROW LEVEL SECURITY;
