-- +goose Up
-- One-time packs are either credits (lasting while the paid plan does) or
-- more websites (lasting a fixed number of days from payment).
ALTER TABLE go_billing_topups DROP CONSTRAINT go_billing_topups_credits_check;
ALTER TABLE go_billing_topups DROP CONSTRAINT go_billing_topups_expires_after_days_check;
ALTER TABLE go_billing_topups
    ADD COLUMN sites integer NOT NULL DEFAULT 0 CHECK (sites BETWEEN 0 AND 100),
    ADD COLUMN sites_until timestamptz,
    ADD CONSTRAINT go_billing_topups_credits_check CHECK (credits >= 0),
    ADD CONSTRAINT go_billing_topups_expires_after_days_check CHECK (expires_after_days BETWEEN 0 AND 730),
    ADD CONSTRAINT go_billing_topups_one_kind CHECK ((credits > 0) <> (sites > 0));
CREATE INDEX go_billing_topups_paid_sites ON go_billing_topups (organization_id, sites_until)
    WHERE status = 'paid' AND sites > 0;

-- +goose Down
DROP INDEX go_billing_topups_paid_sites;
ALTER TABLE go_billing_topups
    DROP CONSTRAINT go_billing_topups_one_kind,
    DROP CONSTRAINT go_billing_topups_expires_after_days_check,
    DROP CONSTRAINT go_billing_topups_credits_check,
    DROP COLUMN sites_until,
    DROP COLUMN sites;
ALTER TABLE go_billing_topups
    ADD CONSTRAINT go_billing_topups_credits_check CHECK (credits > 0),
    ADD CONSTRAINT go_billing_topups_expires_after_days_check CHECK (expires_after_days BETWEEN 1 AND 730);
