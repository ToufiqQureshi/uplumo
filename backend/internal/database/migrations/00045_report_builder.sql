-- +goose Up
ALTER TABLE reports ADD COLUMN IF NOT EXISTS builder_definition jsonb;

-- +goose Down
ALTER TABLE reports DROP COLUMN builder_definition;
