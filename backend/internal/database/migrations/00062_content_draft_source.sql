-- +goose Up
-- The page a draft was imported from, so a second import opens the same
-- draft instead of creating a copy.
ALTER TABLE go_content_drafts ADD COLUMN source_url text;

CREATE UNIQUE INDEX go_content_drafts_project_source_idx
    ON go_content_drafts (project_id, source_url)
    WHERE source_url IS NOT NULL;

-- +goose Down
DROP INDEX go_content_drafts_project_source_idx;
ALTER TABLE go_content_drafts DROP COLUMN source_url;
