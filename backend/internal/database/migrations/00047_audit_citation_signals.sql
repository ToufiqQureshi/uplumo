-- +goose Up
-- Cite Fix: the per-page facts that decide whether an AI assistant can quote
-- a page (schema types, author, dates, answer-first lead, lists, statistics).
-- Written by the crawl with the page row; NULL on every audit crawled before
-- this column existed, which Cite Fix reports as "scan again", never as zero.
ALTER TABLE go_audit_pages ADD COLUMN citation_signals_json text;

-- +goose Down
ALTER TABLE go_audit_pages DROP COLUMN citation_signals_json;
