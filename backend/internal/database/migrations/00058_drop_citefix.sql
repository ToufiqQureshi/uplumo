-- +goose Up
-- Cite Fix was removed from the product (owner, 10 Nov 2026); its tables go
-- with it. go_audit_pages.citation_signals_json and
-- go_audit_javascript_render_pages stay: the site audit still uses them.
DROP TABLE IF EXISTS go_citefix_pitches;
DROP TABLE IF EXISTS go_citefix_ai_overviews;
DROP TABLE IF EXISTS go_citefix_ai_mentions;
DROP TABLE IF EXISTS go_citefix_rival_pages;
DROP TABLE IF EXISTS go_citefix_agent_runs;

-- +goose Down
-- Forward only: the dropped Cite Fix data cannot be restored.
SELECT 1;
