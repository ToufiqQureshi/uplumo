-- +goose Up
-- Keep the whole organic SERP a rank check already paid for, not just the
-- target's own position.
ALTER TABLE go_rank_snapshots
    ADD COLUMN competitors jsonb NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE go_rank_snapshots
    DROP COLUMN competitors;
